// Package components provides component status monitoring for the dashboard.
package components

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

// ComponentValue holds the cached status value for a component.
type ComponentValue struct {
	Value     any       `json:"value"`
	ValueType string    `json:"value_type"`
	Unit      string    `json:"unit,omitempty"`
	LastSeen  time.Time `json:"last_seen"`
}

// StatusChangeCallback is called when a component's status value changes.
type StatusChangeCallback func(componentName string, value *ComponentValue)

// Monitor subscribes to component data topics and caches the latest
// status_value for each component. The dashboard queries this cache
// when rendering component lists.
type Monitor struct {
	mu     sync.RWMutex
	values map[string]*ComponentValue

	onStatusChange StatusChangeCallback

	nats   *gorainats.Client
	topics *topics.Builder
	logger *slog.Logger

	sub    *nats.Subscription
	ctx    context.Context
	cancel context.CancelFunc
}

// NewMonitor creates a new component status monitor.
func NewMonitor(natsClient *gorainats.Client, topicsBuilder *topics.Builder, logger *slog.Logger) *Monitor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Monitor{
		values: make(map[string]*ComponentValue),
		nats:   natsClient,
		topics: topicsBuilder,
		logger: logger,
	}
}

// OnStatusChange registers a callback that fires when a component's status value
// is updated. Only one callback is supported; subsequent calls replace the previous.
func (m *Monitor) OnStatusChange(fn StatusChangeCallback) {
	m.mu.Lock()
	m.onStatusChange = fn
	m.mu.Unlock()
}

// Start begins monitoring component data topics via NATS wildcard subscription.
func (m *Monitor) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	if m.nats == nil || m.topics == nil {
		m.logger.Warn("Component monitor: no NATS or topics configured, running without subscriptions")
		return nil
	}

	topic := m.topics.AllComponents(topics.Data)
	sub, err := m.nats.Subscribe(topic, func(msg *nats.Msg) {
		m.handleDataMessage(msg)
	})
	if err != nil {
		return fmt.Errorf("subscribe to %s: %w", topic, err)
	}
	m.sub = sub

	m.logger.Info("Component monitor started", "topic", topic)
	return nil
}

// Stop stops the component monitor.
func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.sub != nil {
		m.sub.Unsubscribe()
		m.sub = nil
	}
	m.logger.Info("Component monitor stopped")
}

// handleDataMessage parses a NATS data message and extracts status_value fields.
// Topic format: gorai.<robot>.<component>.data
func (m *Monitor) handleDataMessage(msg *nats.Msg) {
	componentName := extractComponentName(msg.Subject)
	if componentName == "" {
		return
	}

	var data map[string]any
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return
	}

	statusValue, hasValue := data["status_value"]
	statusValueType, hasType := data["status_value_type"].(string)
	if !hasValue || !hasType {
		return
	}

	unit, _ := data["status_value_unit"].(string)

	cv := &ComponentValue{
		Value:     statusValue,
		ValueType: statusValueType,
		Unit:      unit,
		LastSeen:  time.Now(),
	}

	m.mu.Lock()
	prev := m.values[componentName]
	changed := prev == nil || fmt.Sprintf("%v", prev.Value) != fmt.Sprintf("%v", cv.Value) || prev.ValueType != cv.ValueType || prev.Unit != cv.Unit
	m.values[componentName] = cv
	cb := m.onStatusChange
	m.mu.Unlock()

	if cb != nil && changed {
		cb(componentName, cv)
	}
}

// extractComponentName pulls the component name from a NATS topic.
// Topic format: gorai.<robot>.<component>.data
func extractComponentName(subject string) string {
	// Split by dots: [gorai, robot, component, data]
	dotCount := 0
	componentStart := 0
	componentEnd := 0

	for i, c := range subject {
		if c == '.' {
			dotCount++
			if dotCount == 2 {
				componentStart = i + 1
			}
			if dotCount == 3 {
				componentEnd = i
				break
			}
		}
	}

	if dotCount < 3 || componentStart >= componentEnd {
		return ""
	}

	return subject[componentStart:componentEnd]
}

// SetStatusValue stores a status value for a component.
// Intended for testing and manual injection.
func (m *Monitor) SetStatusValue(componentName string, value any, valueType string, unit string) {
	cv := &ComponentValue{
		Value:     value,
		ValueType: valueType,
		Unit:      unit,
		LastSeen:  time.Now(),
	}

	m.mu.Lock()
	m.values[componentName] = cv
	cb := m.onStatusChange
	m.mu.Unlock()

	if cb != nil {
		cb(componentName, cv)
	}
}

// GetStatusValue returns the cached status value for a component.
func (m *Monitor) GetStatusValue(componentName string) (value any, valueType string, unit string, lastSeen time.Time, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cv, exists := m.values[componentName]
	if !exists {
		return nil, "", "", time.Time{}, false
	}
	return cv.Value, cv.ValueType, cv.Unit, cv.LastSeen, true
}

// FormatStatusValue returns a display string for a component's status value.
func (m *Monitor) FormatStatusValue(componentName string) string {
	value, valueType, unit, _, ok := m.GetStatusValue(componentName)
	if !ok {
		return ""
	}

	switch valueType {
	case "binary":
		s, _ := value.(string)
		return s
	case "number":
		switch v := value.(type) {
		case float64:
			if unit != "" {
				return fmt.Sprintf("%.1f %s", v, unit)
			}
			return fmt.Sprintf("%.1f", v)
		case json.Number:
			if unit != "" {
				return fmt.Sprintf("%s %s", v.String(), unit)
			}
			return v.String()
		default:
			return fmt.Sprintf("%v", v)
		}
	case "string":
		s, _ := value.(string)
		return s
	default:
		return fmt.Sprintf("%v", value)
	}
}
