package sensorpub

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pressurefake "github.com/emergingrobotics/gorai/components/sensor/pressure/fake"
	"github.com/emergingrobotics/gorai/pkg/embeddednats"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// pubTestDeps resolves nats and the fake sensor by name.
type pubTestDeps struct {
	nc     *nats.Conn
	sensor any
}

func (d pubTestDeps) Get(name string) (any, error) {
	switch name {
	case "nats":
		return d.nc, nil
	case "bmp":
		return d.sensor, nil
	}
	return nil, nil
}
func (d pubTestDeps) GetByType(subtype string) ([]any, error) { return nil, nil }

func startNATS(t *testing.T) *nats.Conn {
	t.Helper()
	srv, err := embeddednats.New(embeddednats.Config{Host: "127.0.0.1", Port: -1})
	if err != nil {
		t.Fatalf("embedded nats: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("start nats: %v", err)
	}
	t.Cleanup(srv.Shutdown)
	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(nc.Close)
	return nc
}

func TestSensorPublisherPublishes(t *testing.T) {
	nc := startNATS(t)
	ctx := context.Background()

	sensorAny, err := pressurefake.New(ctx, nil, registry.Config{"name": "bmp"})
	if err != nil {
		t.Fatalf("fake sensor: %v", err)
	}
	sensor := sensorAny.(*pressurefake.Sensor)
	sensor.Set(95000, 18.5, 480.0)

	// Subscribe before starting the publisher.
	sub, err := nc.SubscribeSync(subjects.NewBuilder("test").ComponentData("bmp"))
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	conf := registry.Config{
		"name":      "sensorpub",
		"namespace": "test",
		"sources": []any{
			map[string]any{"name": "bmp", "kind": "sensor", "rate_hz": 50.0},
		},
	}
	pubAny, err := New(ctx, pubTestDeps{nc: nc, sensor: sensor}, conf)
	if err != nil {
		t.Fatalf("New publisher: %v", err)
	}
	pub := pubAny.(*Publisher)
	if err := pub.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = pub.Close(context.Background()) })

	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("no telemetry received: %v", err)
	}

	var readings map[string]any
	if err := json.Unmarshal(msg.Data, &readings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p, _ := readings["pressure_pa"].(float64); p != 95000 {
		t.Errorf("pressure_pa = %v, want 95000", readings["pressure_pa"])
	}
	if kind, _ := readings["kind"].(string); kind != "sensor" {
		t.Errorf("kind = %v, want sensor", readings["kind"])
	}
}
