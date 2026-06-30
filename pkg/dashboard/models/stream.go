package models

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	gorainats "github.com/emergingrobotics/gorai/pkg/nats"
	"github.com/emergingrobotics/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// StreamHandler handles MJPEG streaming of annotated model outputs.
type StreamHandler struct {
	nats     *gorainats.Client
	subjects *subjects.Builder
	logger *slog.Logger
	maxFPS float64
}

// NewStreamHandler creates a new model stream handler.
func NewStreamHandler(nats *gorainats.Client, subjectsBuilder *subjects.Builder, logger *slog.Logger, maxFPS float64) *StreamHandler {
	return &StreamHandler{
		nats:     nats,
		subjects: subjectsBuilder,
		logger: logger,
		maxFPS: maxFPS,
	}
}

// HandleStream serves an MJPEG stream of annotated model output.
// The stream comes from gorai.<robot>.<model>.annotated topic.
func (h *StreamHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "name")
	if modelName == "" {
		http.Error(w, "model name required", http.StatusBadRequest)
		return
	}

	if h.nats == nil || h.subjects == nil {
		http.Error(w, "NATS not available", http.StatusServiceUnavailable)
		return
	}

	// Build subject for annotated output
	subject := h.subjects.Component(modelName, "annotated")
	h.logger.Debug("Starting model stream", "model", modelName, "subject", subject)

	// Set MJPEG headers
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Connection", "close")

	// Rate limiting
	minInterval := time.Duration(0)
	if h.maxFPS > 0 {
		minInterval = time.Duration(float64(time.Second) / h.maxFPS)
	}

	var lastFrame time.Time
	var frameMu sync.Mutex

	// Frame channel with buffer
	frameCh := make(chan []byte, 2)

	// Subscribe to annotated frames
	sub, err := h.nats.Subscribe(subject, func(msg *nats.Msg) {
		frameMu.Lock()
		defer frameMu.Unlock()

		// Rate limit
		if minInterval > 0 && time.Since(lastFrame) < minInterval {
			return
		}

		lastFrame = time.Now()

		// Non-blocking send
		select {
		case frameCh <- msg.Data:
		default:
			// Drop frame if channel full
		}
	})
	if err != nil {
		h.logger.Error("Failed to subscribe to model stream", "error", err, "subject", subject)
		http.Error(w, "Failed to connect to stream", http.StatusInternalServerError)
		return
	}
	defer sub.Unsubscribe()

	h.logger.Debug("Model MJPEG stream started", "model", modelName, "client", r.RemoteAddr)

	// Stream frames until client disconnects
	for {
		select {
		case <-r.Context().Done():
			h.logger.Debug("Model stream client disconnected", "model", modelName)
			return
		case frame := <-frameCh:
			if err := h.writeFrame(w, frame); err != nil {
				h.logger.Debug("Model MJPEG write error", "model", modelName, "error", err)
				return
			}
		}
	}
}

// writeFrame writes a single MJPEG frame to the response.
func (h *StreamHandler) writeFrame(w http.ResponseWriter, frame []byte) error {
	if _, err := fmt.Fprintf(w, "--frame\r\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Type: image/jpeg\r\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(frame)); err != nil {
		return err
	}
	if _, err := w.Write(frame); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\r\n"); err != nil {
		return err
	}
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

// HandleSnapshot serves a single frame from the model output.
func (h *StreamHandler) HandleSnapshot(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "name")
	if modelName == "" {
		http.Error(w, "model name required", http.StatusBadRequest)
		return
	}

	if h.nats == nil || h.subjects == nil {
		http.Error(w, "NATS not available", http.StatusServiceUnavailable)
		return
	}

	// Build subject for annotated output
	subject := h.subjects.Component(modelName, "annotated")

	frameChan := make(chan []byte, 1)
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	// Subscribe and get one frame
	sub, err := h.nats.Subscribe(subject, func(msg *nats.Msg) {
		select {
		case frameChan <- msg.Data:
		default:
		}
	})
	if err != nil {
		http.Error(w, "Failed to connect to stream", http.StatusInternalServerError)
		return
	}
	defer sub.Unsubscribe()

	select {
	case frame := <-frameChan:
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(frame)))
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(frame)
	case <-timeout.C:
		http.Error(w, "Timeout waiting for frame", http.StatusGatewayTimeout)
	case <-r.Context().Done():
		return
	}
}
