package cameras

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	gorainats "github.com/gorai/gorai/pkg/nats"
	"github.com/gorai/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// StreamHandler bridges NATS camera frames to HTTP MJPEG.
type StreamHandler struct {
	nats     *gorainats.Client
	subjects *subjects.Builder
	monitor *Monitor
	logger  *slog.Logger
	maxFPS  float64
}

// NewStreamHandler creates a new stream handler.
func NewStreamHandler(natsClient *gorainats.Client, subjectsBuilder *subjects.Builder, monitor *Monitor, logger *slog.Logger, maxFPS float64) *StreamHandler {
	if maxFPS <= 0 {
		maxFPS = 30.0
	}
	return &StreamHandler{
		nats:     natsClient,
		subjects: subjectsBuilder,
		monitor: monitor,
		logger:  logger,
		maxFPS:  maxFPS,
	}
}

// HandleStream serves MJPEG stream for a camera.
func (h *StreamHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	cameraName := chi.URLParam(r, "name")
	if cameraName == "" {
		http.Error(w, "camera name required", http.StatusBadRequest)
		return
	}

	if h.nats == nil {
		http.Error(w, "NATS not available", http.StatusServiceUnavailable)
		return
	}

	// Get the correct subject for this camera (handles remote cameras)
	var subject string
	if h.monitor != nil {
		subject = h.monitor.GetCameraSubject(cameraName)
	} else if h.subjects != nil {
		subject = h.subjects.ComponentData(cameraName)
	} else {
		http.Error(w, "Subjects not configured", http.StatusServiceUnavailable)
		return
	}

	// Set MJPEG headers
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Connection", "close")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Rate limiter
	limiter := newRateLimiter(h.maxFPS)

	// Frame channel with buffer
	frameCh := make(chan []byte, 2)

	// Subscribe to camera frames
	sub, err := h.nats.Subscribe(subject, func(msg *nats.Msg) {
		if limiter.Allow() {
			// Non-blocking send
			select {
			case frameCh <- msg.Data:
			default:
				// Drop frame if channel full
			}
		}
	})
	if err != nil {
		h.logger.Error("Failed to subscribe to camera", "camera", cameraName, "error", err)
		http.Error(w, "Failed to subscribe to camera", http.StatusInternalServerError)
		return
	}
	defer sub.Unsubscribe()

	h.logger.Debug("MJPEG stream started", "camera", cameraName, "client", r.RemoteAddr)

	// Stream frames until client disconnects
	for {
		select {
		case <-r.Context().Done():
			h.logger.Debug("MJPEG stream ended", "camera", cameraName, "client", r.RemoteAddr)
			return

		case frame := <-frameCh:
			if err := h.writeFrame(w, frame); err != nil {
				h.logger.Debug("MJPEG write error", "camera", cameraName, "error", err)
				return
			}
		}
	}
}

// writeFrame writes a single MJPEG frame to the response.
func (h *StreamHandler) writeFrame(w http.ResponseWriter, frame []byte) error {
	// Write MJPEG boundary and headers
	if _, err := fmt.Fprintf(w, "--frame\r\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Type: image/jpeg\r\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(frame)); err != nil {
		return err
	}

	// Write frame data
	if _, err := w.Write(frame); err != nil {
		return err
	}

	// End boundary
	if _, err := fmt.Fprintf(w, "\r\n"); err != nil {
		return err
	}

	// Flush to client
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return nil
}

// HandleSnapshot serves a single JPEG frame.
func (h *StreamHandler) HandleSnapshot(w http.ResponseWriter, r *http.Request) {
	cameraName := chi.URLParam(r, "name")
	if cameraName == "" {
		http.Error(w, "camera name required", http.StatusBadRequest)
		return
	}

	if h.nats == nil {
		http.Error(w, "NATS not available", http.StatusServiceUnavailable)
		return
	}

	// Get the correct subject for this camera (handles remote cameras)
	var subject string
	if h.monitor != nil {
		subject = h.monitor.GetCameraSubject(cameraName)
	} else if h.subjects != nil {
		subject = h.subjects.ComponentData(cameraName)
	} else {
		http.Error(w, "Subjects not configured", http.StatusServiceUnavailable)
		return
	}

	// Create a channel to receive one frame
	frameCh := make(chan []byte, 1)
	timeout := time.After(5 * time.Second)

	// Subscribe to get one frame
	sub, err := h.nats.Subscribe(subject, func(msg *nats.Msg) {
		select {
		case frameCh <- msg.Data:
		default:
		}
	})
	if err != nil {
		http.Error(w, "Failed to subscribe to camera", http.StatusInternalServerError)
		return
	}
	defer sub.Unsubscribe()

	// Wait for frame or timeout
	select {
	case <-r.Context().Done():
		return
	case <-timeout:
		http.Error(w, "Timeout waiting for frame", http.StatusGatewayTimeout)
		return
	case frame := <-frameCh:
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(frame)))
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(frame)
	}
}

// rateLimiter provides frame rate limiting.
type rateLimiter struct {
	mu          sync.Mutex
	maxFPS      float64
	minInterval time.Duration
	lastSent    time.Time
}

// newRateLimiter creates a new rate limiter.
func newRateLimiter(maxFPS float64) *rateLimiter {
	return &rateLimiter{
		maxFPS:      maxFPS,
		minInterval: time.Duration(float64(time.Second) / maxFPS),
		lastSent:    time.Time{},
	}
}

// Allow returns true if a frame should be sent.
func (r *rateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if now.Sub(r.lastSent) >= r.minInterval {
		r.lastSent = now
		return true
	}
	return false
}
