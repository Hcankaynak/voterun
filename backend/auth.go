package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = 7 * 24 * time.Hour

// Claims is the JWT payload. Subject holds the user id.
type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (s *Server) generateToken(u *User) (string, error) {
	claims := Claims{
		Email: u.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Server) parseToken(raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// requireAuth is gin middleware that rejects requests without a valid bearer
// token and stores the authenticated user id in the context.
func (s *Server) requireAuth(c *gin.Context) {
	const prefix = "Bearer "
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, prefix) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	claims, err := s.parseToken(strings.TrimPrefix(header, prefix))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}
	c.Set("userID", claims.Subject)
	c.Next()
}

func (s *Server) issueAuth(c *gin.Context, user *User, status int) {
	token, err := s.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create session"})
		return
	}
	c.JSON(status, gin.H{"token": token, "user": user})
}

func (s *Server) register(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	if !strings.Contains(email, "@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a valid email is required"})
		return
	}
	if len(body.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 6 characters"})
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = strings.SplitN(email, "@", 2)[0]
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
		return
	}

	user, err := s.store.CreateUser(email, string(hash), name)
	if errors.Is(err, ErrEmailTaken) {
		c.JSON(http.StatusConflict, gin.H{"error": "an account with that email already exists"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.issueAuth(c, user, http.StatusCreated)
}

func (s *Server) login(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	user, hash, err := s.store.GetUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Compare even when the user is missing to keep timing consistent.
	if user == nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	s.issueAuth(c, user, http.StatusOK)
}

func (s *Server) me(c *gin.Context) {
	user, err := s.store.GetUserByID(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}
