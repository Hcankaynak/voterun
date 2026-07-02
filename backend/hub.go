package main

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub tracks connected WebSocket clients grouped by board id and
// fans board updates out to every participant viewing that board.
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*client]struct{}
}

type client struct {
	conn    *websocket.Conn
	boardID string
	send    chan []byte
}

// wsMessage is the envelope broadcast to clients on every board change.
type wsMessage struct {
	Type  string `json:"type"`
	Board *Board `json:"board"`
}

// NewHub creates an empty hub.
func NewHub() *Hub {
	return &Hub{rooms: make(map[string]map[*client]struct{})}
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[c.boardID] == nil {
		h.rooms[c.boardID] = make(map[*client]struct{})
	}
	h.rooms[c.boardID][c] = struct{}{}
	wsActiveConns.Inc()
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[c.boardID]; ok {
		if _, ok := room[c]; ok {
			delete(room, c)
			close(c.send)
			wsActiveConns.Dec()
		}
		if len(room) == 0 {
			delete(h.rooms, c.boardID)
		}
	}
}

// snapshotBytes marshals a board into the wire envelope used for both
// broadcasts and per-client snapshots.
func snapshotBytes(board *Board) ([]byte, error) {
	return json.Marshal(wsMessage{Type: "board", Board: board})
}

// sendSnapshot enqueues the current board state to a single client (used on
// connect) without fanning out to the whole room.
func (h *Hub) sendSnapshot(c *client, board *Board) {
	if board == nil {
		return
	}
	payload, err := snapshotBytes(board)
	if err != nil {
		return
	}
	select {
	case c.send <- payload:
	default:
		// Drop if the client's buffer is full to avoid blocking.
	}
}

// Broadcast sends the latest board snapshot to everyone in the board's room.
func (h *Hub) Broadcast(board *Board) {
	if board == nil {
		return
	}
	payload, err := snapshotBytes(board)
	if err != nil {
		return
	}
	wsBroadcasts.Inc()

	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[board.ID] {
		select {
		case c.send <- payload:
		default:
			// Drop the message if the client's buffer is full to avoid blocking.
		}
	}
}
