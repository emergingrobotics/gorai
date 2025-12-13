package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// WebSocketHub manages WebSocket connections for real-time updates.
type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	mu         sync.RWMutex
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

// NewWebSocketHub creates a new WebSocket hub.
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run starts the hub's message loop.
func (h *WebSocketHub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Close all connections
			h.mu.Lock()
			for conn := range h.clients {
				conn.Close(websocket.StatusGoingAway, "server shutdown")
			}
			h.clients = make(map[*websocket.Conn]bool)
			h.mu.Unlock()
			return

		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close(websocket.StatusNormalClosure, "")
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := make([]*websocket.Conn, 0, len(h.clients))
			for conn := range h.clients {
				clients = append(clients, conn)
			}
			h.mu.RUnlock()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			for _, conn := range clients {
				if err := conn.Write(ctx, websocket.MessageText, message); err != nil {
					h.unregister <- conn
				}
			}
			cancel()
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *WebSocketHub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	default:
		// Channel full, drop message
	}
}

// BroadcastJSON marshals and broadcasts a JSON message.
func (h *WebSocketHub) BroadcastJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.Broadcast(data)
}

// HandleWebSocket upgrades HTTP to WebSocket and manages the connection.
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow connections from any origin
	})
	if err != nil {
		return
	}

	h.register <- conn

	// Keep connection open
	ctx := r.Context()
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}

	h.unregister <- conn
}

// ClientCount returns the number of connected clients.
func (h *WebSocketHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
