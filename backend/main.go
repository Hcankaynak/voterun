package main

import (
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	port := env("PORT", "8080")
	databaseURL := env("DATABASE_URL", "postgres://voterun:voterun@localhost:5432/voterun?sslmode=disable")
	corsOrigin := env("CORS_ORIGIN", "http://localhost:5173")
	jwtSecret := env("JWT_SECRET", "dev-insecure-secret-change-me")

	if jwtSecret == "dev-insecure-secret-change-me" {
		log.Println("WARNING: using the default JWT_SECRET; set JWT_SECRET in production")
	}

	store, err := NewPostgresStore(databaseURL)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer store.Close()

	hub := NewHub()
	server := NewServer(store, hub, []byte(jwtSecret))

	r := gin.Default()
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}
	if strings.TrimSpace(corsOrigin) == "*" {
		// Dev convenience: reflect any origin (Vite may pick any free port).
		// This reflects the request's Origin rather than sending a literal "*",
		// so it still works with AllowCredentials. Do NOT use "*" in production.
		corsConfig.AllowOriginFunc = func(origin string) bool { return true }
	} else {
		corsConfig.AllowOrigins = strings.Split(corsOrigin, ",")
	}
	r.Use(cors.New(corsConfig))
	r.Use(metricsMiddleware())

	// Prometheus scrape endpoint. Kept internal: Prometheus reaches it over the
	// Compose network, and the host nginx blocks it on api.voterun.app.
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	server.RegisterRoutes(r)

	log.Printf("VoteRun backend listening on :%s (postgres)", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
