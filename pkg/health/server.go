// Package health provides an HTTP health server for process-compose readiness
// and liveness probes. The server exposes /healthz and /livez endpoints that
// report the controller's readiness state.
package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
	idleTimeout  = 30 * time.Second
)

const defaultListenAddress = "127.0.0.1:4180"

// Config holds configuration for the health server.
type Config struct {
	// Listen is the address the health server binds to.
	// Defaults to "127.0.0.1:4180" when empty.
	Listen string
	Logger *slog.Logger
}

// Server is an HTTP health server that exposes readiness and liveness
// endpoints for process-compose probes.
type Server struct {
	listenAddress string
	logger        *slog.Logger
	ready         atomic.Bool
	httpServer    *http.Server
	listener      net.Listener
}

type statusResponse struct {
	Status string `json:"status"`
}

// New creates a health server with the given configuration.
func New(config Config) *Server {
	listenAddress := config.Listen
	if listenAddress == "" {
		listenAddress = defaultListenAddress
	}
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		listenAddress: listenAddress,
		logger:        logger,
	}
}

// Start begins serving health check endpoints in a background goroutine.
// The server is immediately available for liveness checks; readiness checks
// return 503 until SetReady is called.
func (server *Server) Start() error {
	listener, err := net.Listen("tcp", server.listenAddress)
	if err != nil {
		return err
	}
	server.listener = listener
	server.httpServer = &http.Server{
		Handler:      server.handler(),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
	server.logger.Info("health server started", "address", listener.Addr().String())
	go func() {
		if err := server.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			server.logger.Error("health server failed", "error", err)
		}
	}()
	return nil
}

// Shutdown gracefully stops the health server, waiting for in-flight requests
// to complete or the context to expire.
func (server *Server) Shutdown(ctx context.Context) error {
	if server.httpServer == nil {
		return nil
	}
	server.logger.Info("health server shutting down")
	return server.httpServer.Shutdown(ctx)
}

// SetReady marks the controller as ready. After this call, /healthz returns 200.
func (server *Server) SetReady() {
	server.ready.Store(true)
}

// SetNotReady marks the controller as not ready. After this call, /healthz returns 503.
func (server *Server) SetNotReady() {
	server.ready.Store(false)
}

// IsReady reports whether the controller is currently marked as ready.
func (server *Server) IsReady() bool {
	return server.ready.Load()
}

// Address returns the actual address the server is listening on.
// This is useful when the server was started with port 0 to get an
// ephemeral port. Returns an empty string if the server has not started.
func (server *Server) Address() string {
	if server.listener == nil {
		return ""
	}
	return server.listener.Addr().String()
}

// handler builds the HTTP handler mux for the health server.
func (server *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.handleHealthz)
	mux.HandleFunc("GET /livez", server.handleLivez)
	return mux
}

func (server *Server) handleHealthz(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	if server.ready.Load() {
		writer.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(writer).Encode(statusResponse{Status: "ready"}); err != nil {
			server.logger.Error("failed to write healthz response", "error", err)
		}
		return
	}
	writer.WriteHeader(http.StatusServiceUnavailable)
	if err := json.NewEncoder(writer).Encode(statusResponse{Status: "not_ready"}); err != nil {
		server.logger.Error("failed to write healthz response", "error", err)
	}
}

func (server *Server) handleLivez(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode(statusResponse{Status: "alive"}); err != nil {
		server.logger.Error("failed to write livez response", "error", err)
	}
}
