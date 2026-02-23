// Package services provides service status monitoring for the dashboard.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	gorainats "github.com/gorai/gorai/pkg/nats"
	"github.com/gorai/gorai/pkg/topics"
	"github.com/nats-io/nats.go"
)

// ServiceData holds the cached heartbeat data for a service.
type ServiceData struct {
	StatusValue     string         `json:"status_value"`
	StatusValueType string         `json:"status_value_type"`
	Data            map[string]any `json:"data"`
	LastSeen        time.Time      `json:"last_seen"`
}

// StatusChangeCallback is called when a service's status value changes.
type StatusChangeCallback func(serviceName string, data *ServiceData)

// Monitor subscribes to service heartbeat topics and caches the latest
// data for each service. The dashboard queries this cache when rendering
// the service list.
type Monitor struct {
	mu       sync.RWMutex
	services map[string]*ServiceData

	onStatusChange StatusChangeCallback

	nats   *gorainats.Client
	topics *topics.Builder
	logger *slog.Logger

	sub    *nats.Subscription
	ctx    context.Context
	cancel context.CancelFunc
}

// NewMonitor creates a new service status monitor.
func NewMonitor(natsClient *gorainats.Client, topicsBuilder *topics.Builder, logger *slog.Logger) *Monitor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Monitor{
		services: make(map[string]*ServiceData),
		nats:     natsClient,
		topics:   topicsBuilder,
		logger:   logger,
	}
}

// OnStatusChange registers a callback that fires when a service's status value
// is updated. Only one callback is supported; subsequent calls replace the previous.
func (m *Monitor) OnStatusChange(fn StatusChangeCallback) {
	m.mu.Lock()
	m.onStatusChange = fn
	m.mu.Unlock()
}

// Start begins monitoring service heartbeat topics via NATS wildcard subscription.
func (m *Monitor) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	if m.nats == nil || m.topics == nil {
		m.logger.Warn("Service monitor: no NATS or topics configured, running without subscriptions")
		return nil
	}

	// Subscribe to gorai.<robot>.*.heartbeat
	topic := m.topics.AllComponents("heartbeat")
	sub, err := m.nats.Subscribe(topic, func(msg *nats.Msg) {
		m.handleHeartbeatMessage(msg)
	})
	if err != nil {
		return fmt.Errorf("subscribe to %s: %w", topic, err)
	}
	m.sub = sub

	m.logger.Info("Service monitor started", "topic", topic)
	return nil
}

// Stop stops the service monitor.
func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.sub != nil {
		m.sub.Unsubscribe()
		m.sub = nil
	}
	m.logger.Info("Service monitor stopped")
}

// handleHeartbeatMessage parses a NATS heartbeat message and caches the data.
// Topic format: gorai.<robot>.<service>.heartbeat
func (m *Monitor) handleHeartbeatMessage(msg *nats.Msg) {
	topicName := extractServiceName(msg.Subject)
	if topicName == "" || topicName == "system" {
		return
	}

	var data map[string]any
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return
	}

	// Only cache messages that have a "service" field indicating they are service heartbeats
	serviceName, hasService := data["service"].(string)
	if !hasService || serviceName == "" {
		return
	}

	statusValue, _ := data["status_value"].(string)
	statusValueType, _ := data["status_value_type"].(string)

	sd := &ServiceData{
		StatusValue:     statusValue,
		StatusValueType: statusValueType,
		Data:            data,
		LastSeen:        time.Now(),
	}

	m.mu.Lock()
	prev := m.services[serviceName]
	changed := prev == nil || prev.StatusValue != sd.StatusValue
	m.services[serviceName] = sd
	cb := m.onStatusChange
	m.mu.Unlock()

	if cb != nil && changed {
		cb(serviceName, sd)
	}
}

// extractServiceName pulls the service name from a NATS topic.
// Topic format: gorai.<robot>.<service>.heartbeat
func extractServiceName(subject string) string {
	dotCount := 0
	serviceStart := 0
	serviceEnd := 0

	for i, c := range subject {
		if c == '.' {
			dotCount++
			if dotCount == 2 {
				serviceStart = i + 1
			}
			if dotCount == 3 {
				serviceEnd = i
				break
			}
		}
	}

	if dotCount < 3 || serviceStart >= serviceEnd {
		return ""
	}

	return subject[serviceStart:serviceEnd]
}

// GetServiceData returns the cached data for a service.
func (m *Monitor) GetServiceData(name string) (map[string]any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sd, exists := m.services[name]
	if !exists {
		return nil, false
	}
	return sd.Data, true
}

// FormatStatusValue returns the display string for a service's status value.
func (m *Monitor) FormatStatusValue(name string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sd, exists := m.services[name]
	if !exists {
		return ""
	}
	return sd.StatusValue
}

// GetStatusValue returns the cached status value fields for a service.
func (m *Monitor) GetStatusValue(name string) (statusValue string, statusValueType string, lastSeen time.Time, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sd, exists := m.services[name]
	if !exists {
		return "", "", time.Time{}, false
	}
	return sd.StatusValue, sd.StatusValueType, sd.LastSeen, true
}

// SetServiceData stores service data directly (for testing).
func (m *Monitor) SetServiceData(name string, data map[string]any) {
	statusValue, _ := data["status_value"].(string)
	statusValueType, _ := data["status_value_type"].(string)

	sd := &ServiceData{
		StatusValue:     statusValue,
		StatusValueType: statusValueType,
		Data:            data,
		LastSeen:        time.Now(),
	}

	m.mu.Lock()
	m.services[name] = sd
	cb := m.onStatusChange
	m.mu.Unlock()

	if cb != nil {
		cb(name, sd)
	}
}

// FormatServiceDetail returns formatted detail strings for rich service display.
// Returns nil if no special detail is available.
func (m *Monitor) FormatServiceDetail(name string) []ServiceDetailRow {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sd, exists := m.services[name]
	if !exists {
		return nil
	}

	var rows []ServiceDetailRow

	// Suntimes: show sunrise/sunset times
	if todaySunrise, ok := sd.Data["today_sunrise"].(string); ok {
		if t, err := time.Parse(time.RFC3339, todaySunrise); err == nil {
			rows = append(rows, ServiceDetailRow{
				Label: "Sunrise",
				Value: t.Format("3:04 PM"),
			})
		}
	}
	if todaySunset, ok := sd.Data["today_sunset"].(string); ok {
		if t, err := time.Parse(time.RFC3339, todaySunset); err == nil {
			rows = append(rows, ServiceDetailRow{
				Label: "Sunset",
				Value: t.Format("3:04 PM"),
			})
		}
	}

	// Light controller: show schedule details
	if schedulesRaw, ok := sd.Data["schedules"].([]any); ok {
		for _, schedRaw := range schedulesRaw {
			sched, ok := schedRaw.(map[string]any)
			if !ok {
				continue
			}
			schedName, _ := sched["name"].(string)
			onTimeStr, _ := sched["on_time"].(string)
			offTimeStr, _ := sched["off_time"].(string)
			active, _ := sched["active"].(bool)

			var timeRange string
			if onTime, err := time.Parse(time.RFC3339, onTimeStr); err == nil {
				if offTime, err := time.Parse(time.RFC3339, offTimeStr); err == nil {
					timeRange = onTime.Format("3:04 PM") + " - " + offTime.Format("3:04 PM")
				}
			}

			status := "pending"
			if active {
				status = "active"
			}

			rows = append(rows, ServiceDetailRow{
				Label:  schedName,
				Value:  timeRange,
				Status: status,
			})
		}
	}

	return rows
}

// ServiceDetailRow represents a single row of service detail for dashboard display.
type ServiceDetailRow struct {
	Label  string
	Value  string
	Status string
}
