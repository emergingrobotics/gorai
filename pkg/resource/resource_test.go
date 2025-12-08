package resource_test

import (
	"context"
	"testing"

	"github.com/gorai/gorai/pkg/resource"
)

func TestName_String(t *testing.T) {
	tests := []struct {
		name     resource.Name
		expected string
	}{
		{
			name: resource.Name{
				Namespace: "gorai",
				Type:      "component",
				Subtype:   "motor",
				Name:      "left_wheel",
			},
			expected: "gorai:component:motor/left_wheel",
		},
		{
			name: resource.Name{
				Namespace: "myco",
				Type:      "service",
				Subtype:   "vision",
				Name:      "detector",
			},
			expected: "myco:service:vision/detector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.name.String()
			if got != tt.expected {
				t.Errorf("Name.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestName_Short(t *testing.T) {
	n := resource.Name{
		Namespace: "gorai",
		Type:      "component",
		Subtype:   "camera",
		Name:      "front",
	}

	got := n.Short()
	expected := "camera/front"
	if got != expected {
		t.Errorf("Name.Short() = %q, want %q", got, expected)
	}
}

func TestName_Topic(t *testing.T) {
	n := resource.Name{
		Namespace: "gorai",
		Type:      "component",
		Subtype:   "sensor",
		Name:      "cpu_temp",
	}

	got := n.Topic()
	expected := "gorai.component.sensor.cpu_temp"
	if got != expected {
		t.Errorf("Name.Topic() = %q, want %q", got, expected)
	}
}

func TestName_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rname   resource.Name
		wantErr bool
	}{
		{
			name: "valid component",
			rname: resource.Name{
				Namespace: "gorai",
				Type:      "component",
				Subtype:   "motor",
				Name:      "test",
			},
			wantErr: false,
		},
		{
			name: "valid service",
			rname: resource.Name{
				Namespace: "gorai",
				Type:      "service",
				Subtype:   "vision",
				Name:      "detector",
			},
			wantErr: false,
		},
		{
			name: "missing namespace",
			rname: resource.Name{
				Type:    "component",
				Subtype: "motor",
				Name:    "test",
			},
			wantErr: true,
		},
		{
			name: "missing type",
			rname: resource.Name{
				Namespace: "gorai",
				Subtype:   "motor",
				Name:      "test",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			rname: resource.Name{
				Namespace: "gorai",
				Type:      "invalid",
				Subtype:   "motor",
				Name:      "test",
			},
			wantErr: true,
		},
		{
			name: "missing subtype",
			rname: resource.Name{
				Namespace: "gorai",
				Type:      "component",
				Name:      "test",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			rname: resource.Name{
				Namespace: "gorai",
				Type:      "component",
				Subtype:   "motor",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rname.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestName_IsComponent(t *testing.T) {
	component := resource.NewComponentName("gorai", "motor", "left")
	if !component.IsComponent() {
		t.Error("expected IsComponent() to return true")
	}
	if component.IsService() {
		t.Error("expected IsService() to return false")
	}

	service := resource.NewServiceName("gorai", "vision", "detector")
	if service.IsComponent() {
		t.Error("expected IsComponent() to return false")
	}
	if !service.IsService() {
		t.Error("expected IsService() to return true")
	}
}

func TestName_Equal(t *testing.T) {
	n1 := resource.NewComponentName("gorai", "motor", "left")
	n2 := resource.NewComponentName("gorai", "motor", "left")
	n3 := resource.NewComponentName("gorai", "motor", "right")

	if !n1.Equal(n2) {
		t.Error("expected n1.Equal(n2) to be true")
	}
	if n1.Equal(n3) {
		t.Error("expected n1.Equal(n3) to be false")
	}
}

func TestParseName(t *testing.T) {
	tests := []struct {
		input   string
		want    resource.Name
		wantErr bool
	}{
		{
			input: "gorai:component:motor/left_wheel",
			want: resource.Name{
				Namespace: "gorai",
				Type:      "component",
				Subtype:   "motor",
				Name:      "left_wheel",
			},
			wantErr: false,
		},
		{
			input: "myco:service:vision/detector",
			want: resource.Name{
				Namespace: "myco",
				Type:      "service",
				Subtype:   "vision",
				Name:      "detector",
			},
			wantErr: false,
		},
		{
			input:   "invalid",
			wantErr: true,
		},
		{
			input:   "a:b",
			wantErr: true,
		},
		{
			input:   "a:b:c",
			wantErr: true,
		},
		{
			input:   "a:invalid:c/d",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := resource.ParseName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("ParseName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_Unmarshal(t *testing.T) {
	cfg := resource.Config{
		Raw: []byte(`{"interval_ms": 1000, "zone": "cpu", "enabled": true}`),
	}

	var v struct {
		IntervalMS int    `json:"interval_ms"`
		Zone       string `json:"zone"`
		Enabled    bool   `json:"enabled"`
	}

	if err := cfg.Unmarshal(&v); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if v.IntervalMS != 1000 {
		t.Errorf("IntervalMS = %d, want 1000", v.IntervalMS)
	}
	if v.Zone != "cpu" {
		t.Errorf("Zone = %q, want 'cpu'", v.Zone)
	}
	if !v.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestConfig_Get(t *testing.T) {
	cfg := resource.NewConfig(map[string]any{
		"name":    "test",
		"count":   float64(42),
		"rate":    3.14,
		"enabled": true,
	})

	// GetString
	if s, ok := cfg.GetString("name"); !ok || s != "test" {
		t.Errorf("GetString('name') = %q, %v, want 'test', true", s, ok)
	}

	// GetInt
	if i, ok := cfg.GetInt("count"); !ok || i != 42 {
		t.Errorf("GetInt('count') = %d, %v, want 42, true", i, ok)
	}

	// GetFloat
	if f, ok := cfg.GetFloat("rate"); !ok || f != 3.14 {
		t.Errorf("GetFloat('rate') = %v, %v, want 3.14, true", f, ok)
	}

	// GetBool
	if b, ok := cfg.GetBool("enabled"); !ok || !b {
		t.Errorf("GetBool('enabled') = %v, %v, want true, true", b, ok)
	}

	// Get missing key
	if _, ok := cfg.Get("missing"); ok {
		t.Error("Get('missing') should return ok=false")
	}
}

func TestNewConfigFromJSON(t *testing.T) {
	data := []byte(`{"foo": "bar", "num": 123}`)
	cfg, err := resource.NewConfigFromJSON(data)
	if err != nil {
		t.Fatalf("NewConfigFromJSON failed: %v", err)
	}

	if s, ok := cfg.GetString("foo"); !ok || s != "bar" {
		t.Errorf("GetString('foo') = %q, %v, want 'bar', true", s, ok)
	}
}

func TestNewConfigFromJSON_Invalid(t *testing.T) {
	_, err := resource.NewConfigFromJSON([]byte(`invalid json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// Mock resource for testing Dependencies
type mockResource struct {
	name resource.Name
}

func (m *mockResource) Name() resource.Name {
	return m.name
}

func (m *mockResource) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

func (m *mockResource) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, nil
}

func (m *mockResource) Close(ctx context.Context) error {
	return nil
}

func TestSimpleDependencies(t *testing.T) {
	motor := &mockResource{name: resource.NewComponentName("gorai", "motor", "left")}
	camera := &mockResource{name: resource.NewComponentName("gorai", "camera", "front")}

	deps := resource.NewSimpleDependencies(map[string]resource.Resource{
		motor.Name().String():  motor,
		camera.Name().String(): camera,
	})

	// Get by name
	r, err := deps.Get(motor.Name())
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if r != motor {
		t.Error("Get returned wrong resource")
	}

	// Get missing
	_, err = deps.Get(resource.NewComponentName("gorai", "motor", "right"))
	if err == nil {
		t.Error("expected error for missing resource")
	}

	// GetByType
	motors, err := deps.GetByType("motor")
	if err != nil {
		t.Fatalf("GetByType failed: %v", err)
	}
	if len(motors) != 1 {
		t.Errorf("GetByType returned %d resources, want 1", len(motors))
	}

	// All
	all := deps.All()
	if len(all) != 2 {
		t.Errorf("All returned %d resources, want 2", len(all))
	}
}

func TestEmptyDependencies(t *testing.T) {
	deps := resource.EmptyDependencies()

	_, err := deps.Get(resource.NewComponentName("gorai", "motor", "left"))
	if err == nil {
		t.Error("expected error for empty dependencies")
	}

	all := deps.All()
	if len(all) != 0 {
		t.Errorf("All returned %d resources, want 0", len(all))
	}
}

func TestLinkType_String(t *testing.T) {
	tests := []struct {
		lt   resource.LinkType
		want string
	}{
		{resource.LinkTypeSerial, "serial"},
		{resource.LinkTypeIP, "ip"},
		{resource.LinkTypeNATS, "nats"},
		{resource.LinkTypeCAN, "can"},
		{resource.LinkTypeI2C, "i2c"},
		{resource.LinkTypeSPI, "spi"},
		{resource.LinkType(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.lt.String()
		if got != tt.want {
			t.Errorf("LinkType(%d).String() = %q, want %q", tt.lt, got, tt.want)
		}
	}
}

func TestLinkDirection_String(t *testing.T) {
	tests := []struct {
		ld   resource.LinkDirection
		want string
	}{
		{resource.LinkBidirectional, "bidirectional"},
		{resource.LinkBroadcast, "broadcast"},
		{resource.LinkDirection(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.ld.String()
		if got != tt.want {
			t.Errorf("LinkDirection(%d).String() = %q, want %q", tt.ld, got, tt.want)
		}
	}
}

func TestBounds(t *testing.T) {
	bounds := resource.Bounds{
		MinX: 0, MinY: 0, MinZ: 0,
		MaxX: 1, MaxY: 2, MaxZ: 3,
	}

	if bounds.MaxX != 1 || bounds.MaxY != 2 || bounds.MaxZ != 3 {
		t.Error("Bounds fields not set correctly")
	}
}

func TestLinkStats(t *testing.T) {
	stats := resource.LinkStats{
		BytesSent:     1000,
		BytesReceived: 2000,
		MessagesSent:  10,
		MessagesRecv:  20,
		ErrorCount:    1,
	}

	if stats.BytesSent != 1000 || stats.MessagesRecv != 20 {
		t.Error("LinkStats fields not set correctly")
	}
}
