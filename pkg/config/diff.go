package config

import (
	"encoding/json"
	"fmt"
	"slices"
)

// AttributeChange represents a component or service whose attributes changed.
type AttributeChange struct {
	Name          string
	NewAttributes map[string]any
}

// StructuralDiff compares two RDL configs and returns a human-readable reason
// and true if a structural change is detected. Returns ("", false) if the configs
// are structurally identical.
//
// Structural fields include: robot name/namespace, NATS config, dashboard config,
// component/service count, and component/service identity (name, type, model, disabled).
// Components and services are matched by name, not by index position.
func StructuralDiff(old, new *RDL) (string, bool) {
	if old.Robot.Name != new.Robot.Name {
		return fmt.Sprintf("robot name changed from %q to %q", old.Robot.Name, new.Robot.Name), true
	}
	if old.Robot.Namespace != new.Robot.Namespace {
		return fmt.Sprintf("robot namespace changed from %q to %q", old.Robot.Namespace, new.Robot.Namespace), true
	}

	if reason, changed := natsStructuralDiff(old.NATS, new.NATS); changed {
		return reason, true
	}

	if reason, changed := dashboardStructuralDiff(old.Dashboard, new.Dashboard); changed {
		return reason, true
	}

	if reason, changed := componentStructuralDiff(old.Components, new.Components); changed {
		return reason, true
	}

	if reason, changed := serviceStructuralDiff(old.Services, new.Services); changed {
		return reason, true
	}

	return "", false
}

// natsStructuralDiff compares NATS configuration for structural changes.
func natsStructuralDiff(old, new *NATSConfig) (string, bool) {
	oldURL := "nats://localhost:4222"
	newURL := "nats://localhost:4222"
	if old != nil && old.URL != "" {
		oldURL = old.URL
	}
	if new != nil && new.URL != "" {
		newURL = new.URL
	}
	if oldURL != newURL {
		return fmt.Sprintf("NATS URL changed from %q to %q", oldURL, newURL), true
	}

	oldJS := old != nil && old.JetStream
	newJS := new != nil && new.JetStream
	if oldJS != newJS {
		return fmt.Sprintf("NATS JetStream changed from %v to %v", oldJS, newJS), true
	}

	oldCreds := ""
	newCreds := ""
	if old != nil {
		oldCreds = old.CredentialsFile
	}
	if new != nil {
		newCreds = new.CredentialsFile
	}
	if oldCreds != newCreds {
		return "NATS credentials file changed", true
	}

	// Compare URLs (multi-server cluster)
	var oldURLs, newURLs []string
	if old != nil {
		oldURLs = old.URLs
	}
	if new != nil {
		newURLs = new.URLs
	}
	if !slices.Equal(oldURLs, newURLs) {
		return "NATS URLs changed", true
	}

	// Compare TLS config
	var oldTLS, newTLS TLSConfig
	if old != nil && old.TLS != nil {
		oldTLS = *old.TLS
	}
	if new != nil && new.TLS != nil {
		newTLS = *new.TLS
	}
	if oldTLS != newTLS {
		return "NATS TLS configuration changed", true
	}

	// Compare connection parameters
	var oldConnTimeout, newConnTimeout string
	if old != nil {
		oldConnTimeout = old.ConnectTimeout
	}
	if new != nil {
		newConnTimeout = new.ConnectTimeout
	}
	if oldConnTimeout != newConnTimeout {
		return "NATS connect timeout changed", true
	}

	var oldReconnectWait, newReconnectWait string
	if old != nil {
		oldReconnectWait = old.ReconnectWait
	}
	if new != nil {
		newReconnectWait = new.ReconnectWait
	}
	if oldReconnectWait != newReconnectWait {
		return "NATS reconnect wait changed", true
	}

	var oldMaxReconnects, newMaxReconnects int
	if old != nil {
		oldMaxReconnects = old.MaxReconnects
	}
	if new != nil {
		newMaxReconnects = new.MaxReconnects
	}
	if oldMaxReconnects != newMaxReconnects {
		return "NATS max reconnects changed", true
	}

	return "", false
}

// dashboardStructuralDiff compares dashboard configuration for structural changes.
func dashboardStructuralDiff(old, new *DashboardConfig) (string, bool) {
	oldEnabled := true
	newEnabled := true
	if old != nil && old.Enabled != nil {
		oldEnabled = *old.Enabled
	}
	if new != nil && new.Enabled != nil {
		newEnabled = *new.Enabled
	}
	if oldEnabled != newEnabled {
		return fmt.Sprintf("dashboard enabled changed from %v to %v", oldEnabled, newEnabled), true
	}

	oldListen := "127.0.0.1:8080"
	newListen := "127.0.0.1:8080"
	if old != nil && old.Listen != "" {
		oldListen = old.Listen
	}
	if new != nil && new.Listen != "" {
		newListen = new.Listen
	}
	if oldListen != newListen {
		return fmt.Sprintf("dashboard listen changed from %q to %q", oldListen, newListen), true
	}

	// Compare WebSocket config
	var oldWS, newWS WebSocketConfig
	if old != nil && old.WebSocket != nil {
		oldWS = *old.WebSocket
	}
	if new != nil && new.WebSocket != nil {
		newWS = *new.WebSocket
	}
	if oldWS != newWS {
		return "dashboard websocket configuration changed", true
	}

	// Compare Video config
	var oldVideo, newVideo VideoConfig
	if old != nil && old.Video != nil {
		oldVideo = *old.Video
	}
	if new != nil && new.Video != nil {
		newVideo = *new.Video
	}
	if oldVideo != newVideo {
		return "dashboard video configuration changed", true
	}

	return "", false
}

// componentStructuralDiff compares component lists for structural changes.
// Components are matched by name.
func componentStructuralDiff(old, new []ComponentConfig) (string, bool) {
	if len(old) != len(new) {
		return fmt.Sprintf("component count changed from %d to %d", len(old), len(new)), true
	}

	oldByName := make(map[string]ComponentConfig, len(old))
	for _, c := range old {
		oldByName[c.Name] = c
	}

	for _, nc := range new {
		oc, exists := oldByName[nc.Name]
		if !exists {
			return fmt.Sprintf("component %q added (replacing another component)", nc.Name), true
		}
		if oc.Type != nc.Type {
			return fmt.Sprintf("component %q type changed from %q to %q", nc.Name, oc.Type, nc.Type), true
		}
		if oc.Model != nc.Model {
			return fmt.Sprintf("component %q model changed from %q to %q", nc.Name, oc.Model, nc.Model), true
		}
		if oc.Disabled != nc.Disabled {
			return fmt.Sprintf("component %q disabled changed from %v to %v", nc.Name, oc.Disabled, nc.Disabled), true
		}
	}

	return "", false
}

// serviceStructuralDiff compares service lists for structural changes.
// Services are matched by name.
func serviceStructuralDiff(old, new []ServiceConfig) (string, bool) {
	if len(old) != len(new) {
		return fmt.Sprintf("service count changed from %d to %d", len(old), len(new)), true
	}

	oldByName := make(map[string]ServiceConfig, len(old))
	for _, s := range old {
		oldByName[s.Name] = s
	}

	for _, ns := range new {
		os, exists := oldByName[ns.Name]
		if !exists {
			return fmt.Sprintf("service %q added (replacing another service)", ns.Name), true
		}
		if os.Type != ns.Type {
			return fmt.Sprintf("service %q type changed from %q to %q", ns.Name, os.Type, ns.Type), true
		}
		if os.Model != ns.Model {
			return fmt.Sprintf("service %q model changed from %q to %q", ns.Name, os.Model, ns.Model), true
		}
		if os.Disabled != ns.Disabled {
			return fmt.Sprintf("service %q disabled changed from %v to %v", ns.Name, os.Disabled, ns.Disabled), true
		}
	}

	return "", false
}

// AttributeDiff returns lists of component and service names whose attributes
// maps differ between old and new configs. Only call after StructuralDiff
// confirms no structural changes.
func AttributeDiff(old, new *RDL) (components []AttributeChange, services []AttributeChange) {
	oldCompByName := make(map[string]ComponentConfig, len(old.Components))
	for _, c := range old.Components {
		oldCompByName[c.Name] = c
	}
	for _, nc := range new.Components {
		oc := oldCompByName[nc.Name]
		if !attributesEqual(oc.Attributes, nc.Attributes) {
			components = append(components, AttributeChange{
				Name:          nc.Name,
				NewAttributes: nc.Attributes,
			})
		}
	}

	oldSvcByName := make(map[string]ServiceConfig, len(old.Services))
	for _, s := range old.Services {
		oldSvcByName[s.Name] = s
	}
	for _, ns := range new.Services {
		os := oldSvcByName[ns.Name]
		if !attributesEqual(os.Attributes, ns.Attributes) {
			services = append(services, AttributeChange{
				Name:          ns.Name,
				NewAttributes: ns.Attributes,
			})
		}
	}

	return components, services
}

// attributesEqual compares two attribute maps by JSON-marshaling both and
// comparing the bytes. This handles map ordering and nested objects correctly.
func attributesEqual(a, b map[string]any) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	aJSON, errA := json.Marshal(a)
	bJSON, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}
