// Package models provides dashboard functionality for AI/ML model services.
package models

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	gorainats "github.com/gorai/gorai/pkg/nats"
	"github.com/gorai/gorai/pkg/topics"
	"github.com/nats-io/nats.go"
)

// ModelStatus represents the status of a model service.
type ModelStatus struct {
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Status          string    `json:"status"`    // running, offline, error
	InputTopic      string    `json:"input_topic"`
	OutputTopic     string    `json:"output_topic"`
	FPS             float64   `json:"fps"`
	InferenceMs     float64   `json:"inference_ms"`
	FramesProcessed uint64    `json:"frames_processed"`
	TotalDetections uint64    `json:"total_detections"`
	LastSeen        time.Time `json:"last_seen"`
	UptimeSeconds   float64   `json:"uptime_seconds"`
}

// DetectionEvent represents a detection event from a model.
type DetectionEvent struct {
	Timestamp     string      `json:"timestamp"`
	FrameID       uint64      `json:"frame_id"`
	InferenceMs   float64     `json:"inference_time_ms"`
	TotalMs       float64     `json:"total_time_ms"`
	ImageWidth    int         `json:"image_width"`
	ImageHeight   int         `json:"image_height"`
	Detections    []Detection `json:"detections"`
}

// Detection represents a single detection in an event.
type Detection struct {
	ClassID    int     `json:"class_id"`
	ClassName  string  `json:"class_name"`
	Confidence float64 `json:"confidence"`
	BBox       BBox    `json:"bbox"`
}

// BBox represents a bounding box.
type BBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Monitor tracks model service status via NATS messages.
type Monitor struct {
	nats         *gorainats.Client
	topics       *topics.Builder
	logger       *slog.Logger

	models       map[string]*ModelStatus
	modelsMu     sync.RWMutex

	// Recent detections for display
	recentDetections []DetectionEvent
	detectionsMu     sync.RWMutex
	maxDetections    int

	// Subscriptions
	heartbeatSub  *nats.Subscription
	detectionSubs []*nats.Subscription

	ctx    context.Context
	cancel context.CancelFunc
}

// NewMonitor creates a new model service monitor.
func NewMonitor(nats *gorainats.Client, topicsBuilder *topics.Builder, logger *slog.Logger) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Monitor{
		nats:             nats,
		topics:           topicsBuilder,
		logger:           logger,
		models:           make(map[string]*ModelStatus),
		recentDetections: make([]DetectionEvent, 0),
		maxDetections:    100,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start begins monitoring model services.
func (m *Monitor) Start(ctx context.Context) error {
	if m.nats == nil || m.topics == nil {
		m.logger.Warn("NATS or topics not configured, model monitor disabled")
		return nil
	}

	// Subscribe to heartbeat messages
	heartbeatTopic := m.topics.SystemHeartbeat()
	m.logger.Debug("Subscribing to heartbeat topic", "topic", heartbeatTopic)

	sub, err := m.nats.Subscribe(heartbeatTopic, func(msg *nats.Msg) {
		m.handleHeartbeat(msg.Data)
	})
	if err != nil {
		return err
	}
	m.heartbeatSub = sub

	// Subscribe to detection events (wildcard)
	// Pattern: gorai.<robot>.*.detections
	detectionPattern := m.topics.AllComponents("detections")
	m.logger.Debug("Subscribing to detection events", "pattern", detectionPattern)

	detSub, err := m.nats.Subscribe(detectionPattern, func(msg *nats.Msg) {
		m.handleDetection(msg.Data)
	})
	if err != nil {
		m.logger.Warn("Could not subscribe to detection events", "error", err)
	} else {
		m.detectionSubs = append(m.detectionSubs, detSub)
	}

	// Start offline detection goroutine
	go m.checkOffline()

	return nil
}

// handleHeartbeat processes a heartbeat message from a model service.
func (m *Monitor) handleHeartbeat(data []byte) {
	var heartbeat struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		Status  string `json:"status"`
		Metrics struct {
			FramesProcessed uint64  `json:"frames_processed"`
			TotalDetections uint64  `json:"total_detections"`
			FPS             float64 `json:"fps"`
			InferenceMs     float64 `json:"inference_ms"`
			UptimeSeconds   float64 `json:"uptime_seconds"`
		} `json:"metrics"`
	}

	if err := json.Unmarshal(data, &heartbeat); err != nil {
		m.logger.Warn("Failed to parse heartbeat", "error", err)
		return
	}

	// Only track service heartbeats (not component)
	if heartbeat.Type != "service" {
		return
	}

	// Only track object_detection services for now
	if heartbeat.Subtype != "object_detection" {
		return
	}

	m.modelsMu.Lock()
	defer m.modelsMu.Unlock()

	status, exists := m.models[heartbeat.Name]
	if !exists {
		status = &ModelStatus{
			Name: heartbeat.Name,
			Type: heartbeat.Subtype,
		}
		m.models[heartbeat.Name] = status
	}

	status.Status = heartbeat.Status
	status.FPS = heartbeat.Metrics.FPS
	status.InferenceMs = heartbeat.Metrics.InferenceMs
	status.FramesProcessed = heartbeat.Metrics.FramesProcessed
	status.TotalDetections = heartbeat.Metrics.TotalDetections
	status.UptimeSeconds = heartbeat.Metrics.UptimeSeconds
	status.LastSeen = time.Now()
}

// handleDetection processes a detection event.
func (m *Monitor) handleDetection(data []byte) {
	var event DetectionEvent
	if err := json.Unmarshal(data, &event); err != nil {
		m.logger.Warn("Failed to parse detection event", "error", err)
		return
	}

	m.detectionsMu.Lock()
	defer m.detectionsMu.Unlock()

	// Add to front of list
	m.recentDetections = append([]DetectionEvent{event}, m.recentDetections...)

	// Trim to max size
	if len(m.recentDetections) > m.maxDetections {
		m.recentDetections = m.recentDetections[:m.maxDetections]
	}
}

// checkOffline marks models as offline if no heartbeat received.
func (m *Monitor) checkOffline() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.modelsMu.Lock()
			now := time.Now()
			for _, status := range m.models {
				if now.Sub(status.LastSeen) > 15*time.Second {
					status.Status = "offline"
				}
			}
			m.modelsMu.Unlock()
		}
	}
}

// Stop stops the monitor.
func (m *Monitor) Stop() {
	m.cancel()

	// Unsubscribe from heartbeats
	if m.heartbeatSub != nil {
		m.heartbeatSub.Unsubscribe()
	}

	// Unsubscribe from detections
	for _, sub := range m.detectionSubs {
		sub.Unsubscribe()
	}
}

// GetModels returns all tracked model statuses.
func (m *Monitor) GetModels() []ModelStatus {
	m.modelsMu.RLock()
	defer m.modelsMu.RUnlock()

	result := make([]ModelStatus, 0, len(m.models))
	for _, status := range m.models {
		result = append(result, *status)
	}
	return result
}

// GetRecentDetections returns recent detection events.
func (m *Monitor) GetRecentDetections(limit int) []DetectionEvent {
	m.detectionsMu.RLock()
	defer m.detectionsMu.RUnlock()

	if limit <= 0 || limit > len(m.recentDetections) {
		limit = len(m.recentDetections)
	}

	result := make([]DetectionEvent, limit)
	copy(result, m.recentDetections[:limit])
	return result
}
