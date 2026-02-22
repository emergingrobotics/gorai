package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeBaseRDL() *RDL {
	return &RDL{
		Version: "2",
		Robot:   RobotConfig{Name: "test-robot", Namespace: "gorai"},
		NATS:    &NATSConfig{URL: "nats://localhost:4222"},
		Components: []ComponentConfig{
			{Name: "plug_a", Type: "switch", Model: "tasmota", Attributes: map[string]any{"address": "192.168.1.10", "poll_interval": "30s"}},
			{Name: "bulb_a", Type: "switch", Model: "kauf", Attributes: map[string]any{"address": "192.168.1.20", "default_brightness": float64(240)}},
		},
		Services: []ServiceConfig{
			{Name: "light-controller", Type: "automation", Model: "light-controller", Attributes: map[string]any{"retry_attempts": float64(3)}},
		},
	}
}

func cloneRDL(r *RDL) *RDL {
	c := *r
	c.Components = make([]ComponentConfig, len(r.Components))
	for i, comp := range r.Components {
		c.Components[i] = comp
		c.Components[i].Attributes = make(map[string]any)
		for k, v := range comp.Attributes {
			c.Components[i].Attributes[k] = v
		}
	}
	c.Services = make([]ServiceConfig, len(r.Services))
	for i, svc := range r.Services {
		c.Services[i] = svc
		c.Services[i].Attributes = make(map[string]any)
		for k, v := range svc.Attributes {
			c.Services[i].Attributes[k] = v
		}
	}
	if r.NATS != nil {
		n := *r.NATS
		c.NATS = &n
	}
	if r.Dashboard != nil {
		d := *r.Dashboard
		c.Dashboard = &d
	}
	return &c
}

func TestStructuralDiff_IdenticalConfigs(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	reason, changed := StructuralDiff(old, new)
	assert.False(t, changed)
	assert.Empty(t, reason)
}

func TestStructuralDiff_RobotNameChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Robot.Name = "other-robot"
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "robot name changed")
}

func TestStructuralDiff_RobotNamespaceChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Robot.Namespace = "other"
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "namespace changed")
}

func TestStructuralDiff_ComponentAdded(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Components = append(new.Components, ComponentConfig{Name: "plug_b", Type: "switch", Model: "tasmota"})
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "component count changed")
}

func TestStructuralDiff_ComponentRemoved(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Components = new.Components[:1]
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "component count changed")
}

func TestStructuralDiff_ComponentTypeChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Components[0].Type = "sensor"
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "type changed")
}

func TestStructuralDiff_ComponentModelChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Components[0].Model = "shelly"
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "model changed")
}

func TestStructuralDiff_ComponentDisabledChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Components[0].Disabled = true
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "disabled changed")
}

func TestStructuralDiff_ComponentNameSwapped(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Components[0].Name = "new_device"
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "added")
}

func TestStructuralDiff_ServiceAdded(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Services = append(new.Services, ServiceConfig{Name: "extra", Type: "automation", Model: "foo"})
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "service count changed")
}

func TestStructuralDiff_ServiceRemoved(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Services = nil
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "service count changed")
}

func TestStructuralDiff_ServiceTypeChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Services[0].Type = "other"
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "type changed")
}

func TestStructuralDiff_ServiceModelChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Services[0].Model = "other"
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "model changed")
}

func TestStructuralDiff_NATSURLChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.NATS.URL = "nats://other:4222"
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "NATS URL changed")
}

func TestStructuralDiff_NATSJetStreamChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.NATS.JetStream = true
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "JetStream changed")
}

func TestStructuralDiff_DashboardListenChanged(t *testing.T) {
	old := makeBaseRDL()
	old.Dashboard = &DashboardConfig{Listen: ":8080"}
	new := cloneRDL(old)
	new.Dashboard.Listen = ":9090"
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "dashboard listen changed")
}

func TestStructuralDiff_DashboardEnabledChanged(t *testing.T) {
	enabled := true
	disabled := false
	old := makeBaseRDL()
	old.Dashboard = &DashboardConfig{Enabled: &enabled}
	new := cloneRDL(old)
	new.Dashboard.Enabled = &disabled
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "dashboard enabled changed")
}

func TestStructuralDiff_LogLevelNotStructural(t *testing.T) {
	old := makeBaseRDL()
	old.Log = &LogConfig{Level: "info"}
	new := cloneRDL(old)
	new.Log = &LogConfig{Level: "debug"}
	reason, changed := StructuralDiff(old, new)
	assert.False(t, changed)
	assert.Empty(t, reason)
}

func TestStructuralDiff_DescriptionNotStructural(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Robot.Description = "updated description"
	reason, changed := StructuralDiff(old, new)
	assert.False(t, changed)
	assert.Empty(t, reason)
}

func TestStructuralDiff_ComponentOrderChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	// Reverse order
	new.Components[0], new.Components[1] = new.Components[1], new.Components[0]
	reason, changed := StructuralDiff(old, new)
	assert.False(t, changed)
	assert.Empty(t, reason)
}

func TestStructuralDiff_ServiceOrderChanged(t *testing.T) {
	old := makeBaseRDL()
	old.Services = append(old.Services, ServiceConfig{Name: "svc2", Type: "t2", Model: "m2"})
	new := cloneRDL(old)
	new.Services[0], new.Services[1] = new.Services[1], new.Services[0]
	reason, changed := StructuralDiff(old, new)
	assert.False(t, changed)
	assert.Empty(t, reason)
}

func TestAttributeDiff_NoChange(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	comps, svcs := AttributeDiff(old, new)
	assert.Empty(t, comps)
	assert.Empty(t, svcs)
}

func TestAttributeDiff_ComponentChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Components[0].Attributes["poll_interval"] = "60s"
	comps, svcs := AttributeDiff(old, new)
	require.Len(t, comps, 1)
	assert.Equal(t, "plug_a", comps[0].Name)
	assert.Equal(t, "60s", comps[0].NewAttributes["poll_interval"])
	assert.Empty(t, svcs)
}

func TestAttributeDiff_ServiceChanged(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Services[0].Attributes["retry_attempts"] = float64(5)
	comps, svcs := AttributeDiff(old, new)
	assert.Empty(t, comps)
	require.Len(t, svcs, 1)
	assert.Equal(t, "light-controller", svcs[0].Name)
	assert.Equal(t, float64(5), svcs[0].NewAttributes["retry_attempts"])
}

func TestAttributeDiff_NestedChange(t *testing.T) {
	old := makeBaseRDL()
	old.Components[1].Attributes["default_color"] = map[string]any{"r": float64(255), "g": float64(180), "b": float64(0)}
	new := cloneRDL(old)
	new.Components[1].Attributes["default_color"] = map[string]any{"r": float64(0), "g": float64(255), "b": float64(0)}
	comps, _ := AttributeDiff(old, new)
	require.Len(t, comps, 1)
	assert.Equal(t, "bulb_a", comps[0].Name)
}

func TestAttributeDiff_MultipleChanges(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Components[0].Attributes["poll_interval"] = "10s"
	new.Components[1].Attributes["default_brightness"] = float64(100)
	new.Services[0].Attributes["retry_attempts"] = float64(1)
	comps, svcs := AttributeDiff(old, new)
	assert.Len(t, comps, 2)
	assert.Len(t, svcs, 1)
}

func TestAttributeDiff_NewAttributeAdded(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	new.Components[0].Attributes["new_param"] = "value"
	comps, _ := AttributeDiff(old, new)
	require.Len(t, comps, 1)
	assert.Equal(t, "plug_a", comps[0].Name)
}

func TestAttributeDiff_AttributeRemoved(t *testing.T) {
	old := makeBaseRDL()
	new := cloneRDL(old)
	delete(new.Components[0].Attributes, "poll_interval")
	comps, _ := AttributeDiff(old, new)
	require.Len(t, comps, 1)
	assert.Equal(t, "plug_a", comps[0].Name)
}

func TestAttributeDiff_EmptyAttributes(t *testing.T) {
	old := &RDL{
		Robot:      RobotConfig{Name: "test"},
		Components: []ComponentConfig{{Name: "c1", Type: "t", Model: "m"}},
	}
	new := &RDL{
		Robot:      RobotConfig{Name: "test"},
		Components: []ComponentConfig{{Name: "c1", Type: "t", Model: "m"}},
	}
	comps, _ := AttributeDiff(old, new)
	assert.Empty(t, comps)
}

func TestStructuralDiff_NATSURLsChanged(t *testing.T) {
	old := makeBaseRDL()
	old.NATS.URLs = []string{"nats://a:4222"}
	new := cloneRDL(old)
	new.NATS.URLs = []string{"nats://a:4222", "nats://b:4222"}
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "NATS URLs changed")
}

func TestStructuralDiff_NATSTLSChanged(t *testing.T) {
	old := makeBaseRDL()
	old.NATS.TLS = &TLSConfig{CAFile: "/ca.pem"}
	new := cloneRDL(old)
	new.NATS.TLS = &TLSConfig{CAFile: "/new-ca.pem"}
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "NATS TLS")
}

func TestStructuralDiff_NATSConnectTimeoutChanged(t *testing.T) {
	old := makeBaseRDL()
	old.NATS.ConnectTimeout = "5s"
	new := cloneRDL(old)
	new.NATS.ConnectTimeout = "10s"
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "connect timeout")
}

func TestStructuralDiff_NATSReconnectWaitChanged(t *testing.T) {
	old := makeBaseRDL()
	old.NATS.ReconnectWait = "1s"
	new := cloneRDL(old)
	new.NATS.ReconnectWait = "5s"
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "reconnect wait")
}

func TestStructuralDiff_NATSMaxReconnectsChanged(t *testing.T) {
	old := makeBaseRDL()
	old.NATS.MaxReconnects = 10
	new := cloneRDL(old)
	new.NATS.MaxReconnects = 20
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "max reconnects")
}

func TestStructuralDiff_DashboardWebSocketChanged(t *testing.T) {
	enabled := true
	old := makeBaseRDL()
	old.Dashboard = &DashboardConfig{Enabled: &enabled, WebSocket: &WebSocketConfig{BufferSize: 1024}}
	new := cloneRDL(old)
	new.Dashboard.WebSocket = &WebSocketConfig{BufferSize: 2048}
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "websocket")
}

func TestStructuralDiff_DashboardVideoChanged(t *testing.T) {
	enabled := true
	old := makeBaseRDL()
	old.Dashboard = &DashboardConfig{Enabled: &enabled, Video: &VideoConfig{MaxFPS: 30}}
	new := cloneRDL(old)
	new.Dashboard.Video = &VideoConfig{MaxFPS: 15}
	reason, changed := StructuralDiff(old, new)
	assert.True(t, changed)
	assert.Contains(t, reason, "video")
}
