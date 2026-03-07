package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const maxWebSocketClients = 100

// AllowedOriginPatterns returns WebSocket origin patterns derived from a listen
// address. Localhost addresses allow localhost origins; all others allow only the
// specific address. This prevents Cross-Site WebSocket Hijacking.
func AllowedOriginPatterns(listenAddr string) []string {
	if strings.HasPrefix(listenAddr, "127.0.0.1:") || strings.HasPrefix(listenAddr, "localhost:") || strings.HasPrefix(listenAddr, ":") {
		return []string{"http://127.0.0.1:*", "http://localhost:*"}
	}
	return []string{"http://" + listenAddr}
}

// WebSocketHub manages WebSocket connections for real-time updates.
type WebSocketHub struct {
	clients        map[*websocket.Conn]bool
	mu             sync.RWMutex
	broadcast      chan []byte
	register       chan *websocket.Conn
	unregister     chan *websocket.Conn
	done           chan struct{}
	OriginPatterns []string
}

// NewWebSocketHub creates a new WebSocket hub.
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:        make(map[*websocket.Conn]bool),
		broadcast:      make(chan []byte, 256),
		register:       make(chan *websocket.Conn),
		unregister:     make(chan *websocket.Conn),
		done:           make(chan struct{}),
		OriginPatterns: []string{"http://127.0.0.1:*", "http://localhost:*"},
	}
}

// Run starts the hub's message loop.
func (h *WebSocketHub) Run(ctx context.Context) {
	defer close(h.done)

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

			writeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			for _, conn := range clients {
				if err := conn.Write(writeCtx, websocket.MessageText, message); err != nil {
					// Remove failed client directly instead of sending to unregister channel
					// to avoid deadlock (we are the reader of that channel)
					h.mu.Lock()
					if _, ok := h.clients[conn]; ok {
						delete(h.clients, conn)
						conn.Close(websocket.StatusNormalClosure, "")
					}
					h.mu.Unlock()
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
	// Reject if too many clients are connected
	if h.ClientCount() >= maxWebSocketClients {
		http.Error(w, "too many WebSocket connections", http.StatusServiceUnavailable)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.OriginPatterns,
	})
	if err != nil {
		return
	}

	select {
	case h.register <- conn:
	case <-h.done:
		conn.Close(websocket.StatusGoingAway, "server shutdown")
		return
	}

	// Keep connection open
	ctx := r.Context()
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}

	select {
	case h.unregister <- conn:
	case <-h.done:
	}
}

// ClientCount returns the number of connected clients.
func (h *WebSocketHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
