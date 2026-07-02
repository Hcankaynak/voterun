package main

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// defaultBroadcastDelay is the trailing-edge debounce window used to coalesce
// per-board broadcasts. Bursts of mutations within this window collapse into a
// single board reload + fan-out instead of one per mutation.
const defaultBroadcastDelay = 150 * time.Millisecond

// Server wires the store and the realtime hub into HTTP handlers.
type Server struct {
	store     Repository
	hub       *Hub
	jwtSecret []byte
	upgrader  websocket.Upgrader

	// Per-board broadcast coalescing: a board with a pending timer absorbs
	// further mutations until the timer fires and flushes the latest state.
	bmu            sync.Mutex
	pending        map[string]*time.Timer
	broadcastDelay time.Duration
}

// NewServer builds a Server with sensible WebSocket defaults.
func NewServer(store Repository, hub *Hub, jwtSecret []byte) *Server {
	return &Server{
		store:          store,
		hub:            hub,
		jwtSecret:      jwtSecret,
		pending:        make(map[string]*time.Timer),
		broadcastDelay: broadcastDelayFromEnv(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Origin is validated by the CORS middleware on the REST side;
			// allow the upgrade here so the dev proxy works out of the box.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// broadcastDelayFromEnv reads BROADCAST_DEBOUNCE_MS, falling back to the
// default. A value of 0 disables coalescing (flush immediately).
func broadcastDelayFromEnv() time.Duration {
	raw := os.Getenv("BROADCAST_DEBOUNCE_MS")
	if raw == "" {
		return defaultBroadcastDelay
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 0 {
		return defaultBroadcastDelay
	}
	return time.Duration(ms) * time.Millisecond
}

// RegisterRoutes mounts all API and WebSocket routes onto the router.
func (s *Server) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		// Authentication.
		api.POST("/auth/register", s.register)
		api.POST("/auth/login", s.login)
		api.GET("/auth/me", s.requireAuth, s.me)

		// Boards owned by the authenticated user.
		api.GET("/boards", s.requireAuth, s.listMyBoards)
		api.POST("/boards", s.requireAuth, s.createBoard)
		api.PATCH("/boards/:id/status", s.requireAuth, s.setBoardStatus)

		// Open endpoints: invited participants join a board via its link.
		api.GET("/boards/:id", s.getBoard)
		api.POST("/boards/:id/cards", s.createCard)
		api.PATCH("/cards/:id", s.updateCard)
		api.DELETE("/cards/:id", s.deleteCard)
		api.POST("/cards/:id/vote", s.voteCard)
	}

	r.GET("/ws/boards/:id", s.handleWS)
}

func (s *Server) listMyBoards(c *gin.Context) {
	boards, err := s.store.ListBoardsByOwner(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, boards)
}

func (s *Server) createBoard(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&body)

	board, err := s.store.CreateBoard(body.Name, c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, board)
}

func (s *Server) setBoardStatus(c *gin.Context) {
	var body struct {
		Closed bool `json:"closed"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "closed is required"})
		return
	}

	ok, err := s.store.SetBoardClosed(c.Param("id"), c.GetString("userID"), body.Closed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "board not found or not owned by you"})
		return
	}
	s.scheduleBroadcast(c.Param("id"))
	c.Status(http.StatusOK)
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
	if errors.Is(err, ErrBoardClosed) {
		c.JSON(http.StatusForbidden, gin.H{"error": "this retro is closed"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if boardID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "column not found"})
		return
	}
	s.scheduleBroadcast(boardID)
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
	if errors.Is(err, ErrBoardClosed) {
		c.JSON(http.StatusForbidden, gin.H{"error": "this retro is closed"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if boardID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}
	s.scheduleBroadcast(boardID)
	c.Status(http.StatusOK)
}

func (s *Server) deleteCard(c *gin.Context) {
	boardID, err := s.store.DeleteCard(c.Param("id"))
	if errors.Is(err, ErrBoardClosed) {
		c.JSON(http.StatusForbidden, gin.H{"error": "this retro is closed"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if boardID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}
	s.scheduleBroadcast(boardID)
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
	if errors.Is(err, ErrBoardClosed) {
		c.JSON(http.StatusForbidden, gin.H{"error": "this retro is closed"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if boardID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}
	s.scheduleBroadcast(boardID)
	c.Status(http.StatusOK)
}

// scheduleBroadcast coalesces board updates. The first mutation for a board
// schedules a flush after broadcastDelay; further mutations within that window
// are absorbed, so a burst produces a single reload + fan-out. When the delay is
// zero, it flushes immediately.
func (s *Server) scheduleBroadcast(boardID string) {
	if s.broadcastDelay <= 0 {
		s.flushBoard(boardID)
		return
	}

	s.bmu.Lock()
	defer s.bmu.Unlock()
	if _, ok := s.pending[boardID]; ok {
		// A flush is already queued; it will pick up the latest state.
		return
	}
	s.pending[boardID] = time.AfterFunc(s.broadcastDelay, func() {
		s.bmu.Lock()
		delete(s.pending, boardID)
		s.bmu.Unlock()
		s.flushBoard(boardID)
	})
}

// flushBoard reloads the board and pushes it to all connected clients.
func (s *Server) flushBoard(boardID string) {
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

	// Send the current snapshot only to this client, reusing the board already
	// loaded for the 404 check above. Avoids a second GetBoard and a full-room
	// re-broadcast on every connect.
	s.hub.sendSnapshot(cl, board)

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
