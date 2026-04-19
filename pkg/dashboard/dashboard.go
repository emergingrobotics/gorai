// Package dashboard provides a web dashboard for Gorai robots.
package dashboard

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/dashboard/cameras"
	"github.com/gorai/gorai/pkg/dashboard/models"
	gorainats "github.com/gorai/gorai/pkg/nats"
	"github.com/gorai/gorai/pkg/subjects"
)

// Dashboard represents the web dashboard service.
type Dashboard struct {
	cfg      *config.DashboardConfig
	robotCfg *config.RDL
	logger   *slog.Logger
	nats     *gorainats.Client
	subjects *subjects.Builder

	server *http.Server
	router *chi.Mux

	// Camera monitoring
	cameraMonitor *cameras.Monitor

	// Model service monitoring
	modelMonitor *models.Monitor

	// WebSocket hub for real-time updates
	wsHub *WebSocketHub

	mu      sync.RWMutex
	running bool
}

// Option configures a Dashboard.
type Option func(*Dashboard)

// WithNATS sets the NATS client.
func WithNATS(client *gorainats.Client) Option {
	return func(d *Dashboard) {
		d.nats = client
	}
}

// WithSubjects sets the subject builder.
func WithSubjects(builder *subjects.Builder) Option {
	return func(d *Dashboard) {
		d.subjects = builder
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(d *Dashboard) {
		d.logger = logger
	}
}

// New creates a new Dashboard service.
func New(cfg *config.DashboardConfig, robotCfg *config.RDL, opts ...Option) (*Dashboard, error) {
	if cfg == nil {
		cfg = &config.DashboardConfig{}
	}

	// Apply defaults — bind to localhost only for safety
	listen := cfg.Listen
	if listen == "" {
		listen = "127.0.0.1:8080"
	}

	d := &Dashboard{
		cfg:      cfg,
		robotCfg: robotCfg,
		logger:   slog.Default(),
		wsHub:    NewWebSocketHub(),
	}

	for _, opt := range opts {
		opt(d)
	}

	// Create camera monitor
	d.cameraMonitor = cameras.NewMonitor(
		d.nats,
		d.subjects,
		d.robotCfg,
		cameras.WithMonitorLogger(d.logger),
	)

	// Set up status change callback to broadcast via WebSocket
	d.cameraMonitor.OnStatusChange(func(status cameras.CameraStatus) {
		d.wsHub.BroadcastJSON(status)
	})

	// Create model service monitor
	d.modelMonitor = models.NewMonitor(
		d.nats,
		d.subjects,
		d.logger,
	)

	// Set up routes
	d.setupRoutes()

	// Create HTTP server
	d.server = &http.Server{
		Addr:         listen,
		Handler:      d.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // Disabled for streaming
		IdleTimeout:  120 * time.Second,
	}

	return d, nil
}

// Start starts the dashboard HTTP server.
func (d *Dashboard) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("dashboard already running")
	}
	d.running = true
	d.mu.Unlock()

	// Start WebSocket hub
	go d.wsHub.Run(ctx)

	// Start camera monitor
	if err := d.cameraMonitor.Start(ctx); err != nil {
		d.logger.Warn("Failed to start camera monitor", "error", err)
	}

	// Start model monitor
	if err := d.modelMonitor.Start(ctx); err != nil {
		d.logger.Warn("Failed to start model monitor", "error", err)
	}

	// Warn if binding to all interfaces without authentication
	if strings.HasPrefix(d.server.Addr, ":") || strings.HasPrefix(d.server.Addr, "0.0.0.0:") {
		d.logger.Warn("Dashboard is bound to all network interfaces with no authentication",
			"addr", d.server.Addr,
			"recommendation", "set listen to 127.0.0.1:<port> or add authentication",
		)
	}

	// Start HTTP server in background
	go func() {
		d.logger.Info("Dashboard server starting", "addr", d.server.Addr)
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			d.logger.Error("Dashboard server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully stops the dashboard.
func (d *Dashboard) Stop(ctx context.Context) error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = false
	d.mu.Unlock()

	// Stop camera monitor
	d.cameraMonitor.Stop()

	// Stop model monitor
	d.modelMonitor.Stop()

	// Shutdown HTTP server
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := d.server.Shutdown(shutdownCtx); err != nil {
		d.logger.Warn("Dashboard shutdown error", "error", err)
		return err
	}

	d.logger.Info("Dashboard stopped")
	return nil
}

// IsRunning returns true if the dashboard is running.
func (d *Dashboard) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}
