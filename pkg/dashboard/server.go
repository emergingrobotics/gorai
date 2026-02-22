package dashboard

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorai/gorai/pkg/dashboard/cameras"
	"github.com/gorai/gorai/pkg/dashboard/models"
	"github.com/gorai/gorai/pkg/dashboard/static"
)

// securityHeaders adds security headers to all responses.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// setupRoutes configures the Chi router with all dashboard routes.
func (d *Dashboard) setupRoutes() {
	r := chi.NewRouter()

	// Middleware
	r.Use(securityHeaders)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Static files (embedded)
	staticFS, err := fs.Sub(static.FS, ".")
	if err == nil {
		r.Handle("/static/*", http.StripPrefix("/static/",
			http.FileServer(http.FS(staticFS))))
	}

	// Dashboard pages
	r.Get("/", d.handleIndex)
	r.Get("/health", d.handleHealth)

	// API endpoints
	r.Route("/api", func(r chi.Router) {
		r.Get("/status", d.handleStatus)
		r.Get("/cameras", d.handleCamerasAPI)
		r.Post("/components/{name}/command", d.handleComponentCommand)
	})

	// WebSocket for general updates
	r.Get("/ws", d.wsHub.HandleWebSocket)

	// Create camera handlers
	streamHandler := cameras.NewStreamHandler(
		d.nats,
		d.topics,
		d.cameraMonitor,
		d.logger,
		d.getMaxFPS(),
	)

	cameraHandler := cameras.NewHandler(
		d.cameraMonitor,
		d.robotCfg,
		d.logger,
	)

	// Camera endpoints
	r.Route("/cameras", func(r chi.Router) {
		r.Get("/", cameraHandler.HandleList)
		r.Get("/{name}/stream", streamHandler.HandleStream)
		r.Get("/{name}/snapshot", streamHandler.HandleSnapshot)
	})

	// Camera status WebSocket
	r.Get("/ws/cameras", cameraHandler.HandleWebSocket)

	// Create model handlers
	modelStreamHandler := models.NewStreamHandler(
		d.nats,
		d.topics,
		d.logger,
		d.getMaxFPS(),
	)

	modelHandler := models.NewHandler(
		d.modelMonitor,
		d.robotCfg,
		d.logger,
	)

	// Model endpoints
	r.Route("/models", func(r chi.Router) {
		r.Get("/", modelHandler.HandleList)
		r.Get("/detections", modelHandler.HandleDetections)
		r.Get("/{name}/stream", modelStreamHandler.HandleStream)
		r.Get("/{name}/snapshot", modelStreamHandler.HandleSnapshot)
	})

	// Model status WebSocket
	r.Get("/ws/models", modelHandler.HandleWebSocket)

	d.router = r
}

// getMaxFPS returns the configured max FPS for video streaming.
func (d *Dashboard) getMaxFPS() float64 {
	if d.cfg != nil && d.cfg.Video != nil && d.cfg.Video.MaxFPS > 0 {
		return float64(d.cfg.Video.MaxFPS)
	}
	return 30.0 // default
}
