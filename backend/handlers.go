package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Server wires the store and the realtime hub into HTTP handlers.
type Server struct {
	store    *Store
	hub      *Hub
	upgrader websocket.Upgrader
}

// NewServer builds a Server with sensible WebSocket defaults.
func NewServer(store *Store, hub *Hub) *Server {
	return &Server{
		store: store,
		hub:   hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Origin is validated by the CORS middleware on the REST side;
			// allow the upgrade here so the dev proxy works out of the box.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// RegisterRoutes mounts all API and WebSocket routes onto the router.
func (s *Server) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.POST("/boards", s.createBoard)
		api.GET("/boards/:id", s.getBoard)
		api.POST("/boards/:id/cards", s.createCard)
		api.PATCH("/cards/:id", s.updateCard)
		api.DELETE("/cards/:id", s.deleteCard)
		api.POST("/cards/:id/vote", s.voteCard)
	}

	r.GET("/ws/boards/:id", s.handleWS)
}

func (s *Server) createBoard(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&body)

	board, err := s.store.CreateBoard(body.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, board)
}

func (s *Server) getBoard(c *gin.Context) {
	board, err := s.store.GetBoard(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if board == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "board not found"})
		return
	}
	c.JSON(http.StatusOK, board)
}

func (s *Server) createCard(c *gin.Context) {
	var body struct {
		ColumnID string `json:"columnId"`
		Text     string `json:"text"`
		Author   string `json:"author"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ColumnID == "" || body.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "columnId and text are required"})
		return
	}

	boardID, err := s.store.CreateCard(body.ColumnID, body.Text, body.Author)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if boardID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "column not found"})
		return
	}
	s.broadcastBoard(boardID)
	c.Status(http.StatusCreated)
}

func (s *Server) updateCard(c *gin.Context) {
	var body struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}

	boardID, err := s.store.UpdateCard(c.Param("id"), body.Text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if boardID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}
	s.broadcastBoard(boardID)
	c.Status(http.StatusOK)
}

func (s *Server) deleteCard(c *gin.Context) {
	boardID, err := s.store.DeleteCard(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if boardID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}
	s.broadcastBoard(boardID)
	c.Status(http.StatusOK)
}

func (s *Server) voteCard(c *gin.Context) {
	var body struct {
		VoterID string `json:"voterId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.VoterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voterId is required"})
		return
	}

	boardID, err := s.store.ToggleVote(c.Param("id"), body.VoterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if boardID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}
	s.broadcastBoard(boardID)
	c.Status(http.StatusOK)
}

// broadcastBoard reloads the board and pushes it to all connected clients.
func (s *Server) broadcastBoard(boardID string) {
	board, err := s.store.GetBoard(boardID)
	if err != nil || board == nil {
		return
	}
	s.hub.Broadcast(board)
}

func (s *Server) handleWS(c *gin.Context) {
	boardID := c.Param("id")
	board, err := s.store.GetBoard(boardID)
	if err != nil || board == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "board not found"})
		return
	}

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	cl := &client{conn: conn, boardID: boardID, send: make(chan []byte, 16)}
	s.hub.add(cl)

	go s.writePump(cl)
	s.readPump(cl)
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// readPump drains incoming frames (and keepalive pongs) until the client leaves.
func (s *Server) readPump(cl *client) {
	defer func() {
		s.hub.remove(cl)
		cl.conn.Close()
	}()

	cl.conn.SetReadDeadline(time.Now().Add(pongWait))
	cl.conn.SetPongHandler(func(string) error {
		cl.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Send the current board snapshot immediately on connect.
	if board, err := s.store.GetBoard(cl.boardID); err == nil && board != nil {
		s.hub.Broadcast(board)
	}

	for {
		if _, _, err := cl.conn.ReadMessage(); err != nil {
			break
		}
	}
}

// writePump streams queued board updates and periodic pings to the client.
func (s *Server) writePump(cl *client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		cl.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-cl.send:
			cl.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				cl.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := cl.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			cl.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := cl.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
