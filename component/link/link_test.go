package link_test

import (
	"context"
	"testing"
	"time"

	"github.com/gorai/gorai/component"
	"github.com/gorai/gorai/component/link"
	"github.com/gorai/gorai/component/link/fake"
	"github.com/gorai/gorai/pkg/resource"
)

func TestLink_IsComponent(t *testing.T) {
	// Link must implement component.Component
	var _ component.Component = (link.Link)(nil)
}

func TestFakeLink_Type(t *testing.T) {
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)

	// Default is NATS
	if l.Type() != resource.LinkTypeNATS {
		t.Errorf("type = %v, want LinkTypeNATS", l.Type())
	}

	// Change type
	l.SetType(resource.LinkTypeSerial)
	if l.Type() != resource.LinkTypeSerial {
		t.Errorf("type = %v, want LinkTypeSerial", l.Type())
	}
}

func TestFakeLink_Direction(t *testing.T) {
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)

	// Default is broadcast
	if l.Direction() != resource.LinkBroadcast {
		t.Errorf("direction = %v, want LinkBroadcast", l.Direction())
	}

	// Change direction
	l.SetDirection(resource.LinkBidirectional)
	if l.Direction() != resource.LinkBidirectional {
		t.Errorf("direction = %v, want LinkBidirectional", l.Direction())
	}
}

func TestFakeLink_IsConnected(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)

	// Default is connected
	connected, err := l.IsConnected(ctx)
	if err != nil {
		t.Fatalf("IsConnected failed: %v", err)
	}
	if !connected {
		t.Error("expected connected initially")
	}

	// Disconnect
	l.SetConnected(false)
	connected, err = l.IsConnected(ctx)
	if err != nil {
		t.Fatalf("IsConnected failed: %v", err)
	}
	if connected {
		t.Error("expected disconnected after SetConnected(false)")
	}
}

func TestFakeLink_GetStats(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)

	// Initial stats should be zero
	stats, err := l.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.BytesSent != 0 || stats.MessagesSent != 0 {
		t.Error("expected zero initial stats")
	}

	// Send some data
	if err := l.Send(ctx, []byte("hello")); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if err := l.Send(ctx, []byte("world")); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	stats, err = l.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.BytesSent != 10 {
		t.Errorf("BytesSent = %d, want 10", stats.BytesSent)
	}
	if stats.MessagesSent != 2 {
		t.Errorf("MessagesSent = %d, want 2", stats.MessagesSent)
	}

	// Add receive data and receive it
	l.AddToReceiveBuffer([]byte("response"))
	_, err = l.Receive(ctx)
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	stats, err = l.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.BytesReceived != 8 {
		t.Errorf("BytesReceived = %d, want 8", stats.BytesReceived)
	}
	if stats.MessagesRecv != 1 {
		t.Errorf("MessagesRecv = %d, want 1", stats.MessagesRecv)
	}

	// Test latency
	l.SetLatency(50 * time.Millisecond)
	stats, err = l.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.Latency != 50*time.Millisecond {
		t.Errorf("Latency = %v, want 50ms", stats.Latency)
	}
}

func TestFakeLink_SendReceive(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)

	// Send data
	testData := []byte("test message")
	if err := l.Send(ctx, testData); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Check send buffer
	buffer := l.GetSendBuffer()
	if len(buffer) != 1 {
		t.Fatalf("send buffer length = %d, want 1", len(buffer))
	}
	if string(buffer[0]) != "test message" {
		t.Errorf("send buffer[0] = %q, want 'test message'", buffer[0])
	}

	// Add to receive buffer and receive
	l.AddToReceiveBuffer([]byte("response"))
	data, err := l.Receive(ctx)
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if string(data) != "response" {
		t.Errorf("received = %q, want 'response'", data)
	}

	// Receive when buffer is empty should fail
	_, err = l.Receive(ctx)
	if err == nil {
		t.Error("expected error when receiving from empty buffer")
	}
}

func TestFakeLink_SendWhenDisconnected(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)

	l.SetConnected(false)

	err := l.Send(ctx, []byte("test"))
	if err == nil {
		t.Error("expected error when sending on disconnected link")
	}
}

func TestFakeLink_GetProperties(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "link", "nats_link")
	l := fake.NewWithName(name)

	props, err := l.GetProperties(ctx)
	if err != nil {
		t.Fatalf("GetProperties failed: %v", err)
	}

	if props.Type != resource.LinkTypeNATS {
		t.Errorf("type = %v, want LinkTypeNATS", props.Type)
	}
	if props.Direction != resource.LinkBroadcast {
		t.Errorf("direction = %v, want LinkBroadcast", props.Direction)
	}
	if props.Name != "nats_link" {
		t.Errorf("name = %q, want 'nats_link'", props.Name)
	}
	if !props.SupportsQoS {
		t.Error("expected SupportsQoS to be true for NATS link")
	}
	if !props.SupportsPersistence {
		t.Error("expected SupportsPersistence to be true for NATS link")
	}
}

func TestFakeLink_SetTimeout(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)

	if err := l.SetTimeout(ctx, 1000); err != nil {
		t.Fatalf("SetTimeout failed: %v", err)
	}
}

func TestFakeLink_FlushAndReset(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)

	// Send some data
	l.Send(ctx, []byte("test"))
	l.AddToReceiveBuffer([]byte("data"))

	// Flush should succeed
	if err := l.Flush(ctx); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Reset should clear buffers
	if err := l.Reset(ctx); err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	buffer := l.GetSendBuffer()
	if len(buffer) != 0 {
		t.Error("expected empty send buffer after reset")
	}

	stats, _ := l.GetStats(ctx)
	if stats.BytesSent != 0 {
		t.Error("expected zero stats after reset")
	}
}

func TestFakeLink_Serial(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)
	l.SetType(resource.LinkTypeSerial)
	l.SetDirection(resource.LinkBidirectional)

	// Test baud rate
	baud, err := l.GetBaudRate(ctx)
	if err != nil {
		t.Fatalf("GetBaudRate failed: %v", err)
	}
	if baud != 115200 {
		t.Errorf("baud rate = %d, want 115200", baud)
	}

	if err := l.SetBaudRate(ctx, 9600); err != nil {
		t.Fatalf("SetBaudRate failed: %v", err)
	}
	baud, _ = l.GetBaudRate(ctx)
	if baud != 9600 {
		t.Errorf("baud rate = %d, want 9600", baud)
	}

	// Test data bits
	bits, err := l.GetDataBits(ctx)
	if err != nil {
		t.Fatalf("GetDataBits failed: %v", err)
	}
	if bits != 8 {
		t.Errorf("data bits = %d, want 8", bits)
	}

	if err := l.SetDataBits(ctx, 7); err != nil {
		t.Fatalf("SetDataBits failed: %v", err)
	}
	bits, _ = l.GetDataBits(ctx)
	if bits != 7 {
		t.Errorf("data bits = %d, want 7", bits)
	}

	// Test stop bits
	stopBits, err := l.GetStopBits(ctx)
	if err != nil {
		t.Fatalf("GetStopBits failed: %v", err)
	}
	if stopBits != 1 {
		t.Errorf("stop bits = %v, want 1", stopBits)
	}

	if err := l.SetStopBits(ctx, 2); err != nil {
		t.Fatalf("SetStopBits failed: %v", err)
	}
	stopBits, _ = l.GetStopBits(ctx)
	if stopBits != 2 {
		t.Errorf("stop bits = %v, want 2", stopBits)
	}

	// Test parity
	parity, err := l.GetParity(ctx)
	if err != nil {
		t.Fatalf("GetParity failed: %v", err)
	}
	if parity != "none" {
		t.Errorf("parity = %q, want 'none'", parity)
	}

	if err := l.SetParity(ctx, "even"); err != nil {
		t.Fatalf("SetParity failed: %v", err)
	}
	parity, _ = l.GetParity(ctx)
	if parity != "even" {
		t.Errorf("parity = %q, want 'even'", parity)
	}

	// Test available
	l.AddToReceiveBuffer([]byte("hello"))
	l.AddToReceiveBuffer([]byte("world"))

	avail, err := l.Available(ctx)
	if err != nil {
		t.Fatalf("Available failed: %v", err)
	}
	if avail != 10 {
		t.Errorf("available = %d, want 10", avail)
	}
}

func TestFakeLink_IP(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)
	l.SetType(resource.LinkTypeIP)

	// Test remote address
	addr, err := l.GetRemoteAddress(ctx)
	if err != nil {
		t.Fatalf("GetRemoteAddress failed: %v", err)
	}
	if addr != "127.0.0.1" {
		t.Errorf("remote address = %q, want '127.0.0.1'", addr)
	}

	l.SetRemoteAddress("192.168.1.100")
	addr, _ = l.GetRemoteAddress(ctx)
	if addr != "192.168.1.100" {
		t.Errorf("remote address = %q, want '192.168.1.100'", addr)
	}

	// Test local address
	local, err := l.GetLocalAddress(ctx)
	if err != nil {
		t.Fatalf("GetLocalAddress failed: %v", err)
	}
	if local != "127.0.0.1" {
		t.Errorf("local address = %q, want '127.0.0.1'", local)
	}

	// Test port
	port, err := l.GetPort(ctx)
	if err != nil {
		t.Fatalf("GetPort failed: %v", err)
	}
	if port != 4222 {
		t.Errorf("port = %d, want 4222", port)
	}

	l.SetPort(8080)
	port, _ = l.GetPort(ctx)
	if port != 8080 {
		t.Errorf("port = %d, want 8080", port)
	}

	// Test protocol
	protocol, err := l.GetProtocol(ctx)
	if err != nil {
		t.Fatalf("GetProtocol failed: %v", err)
	}
	if protocol != "tcp" {
		t.Errorf("protocol = %q, want 'tcp'", protocol)
	}
}

func TestFakeLink_Name(t *testing.T) {
	name := resource.NewComponentName("gorai", "link", "serial_link")
	l := fake.NewWithName(name)

	if l.Name().String() != "gorai:component:link/serial_link" {
		t.Errorf("Name() = %q, want 'gorai:component:link/serial_link'", l.Name().String())
	}
}

func TestFakeLink_DoCommand(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)

	// Test get_state command
	result, err := l.DoCommand(ctx, map[string]any{
		"command": "get_state",
	})
	if err != nil {
		t.Fatalf("DoCommand get_state failed: %v", err)
	}

	if result["type"].(string) != "nats" {
		t.Errorf("type = %v, want 'nats'", result["type"])
	}
	if result["connected"].(bool) != true {
		t.Errorf("connected = %v, want true", result["connected"])
	}

	// Test disconnect command
	_, err = l.DoCommand(ctx, map[string]any{
		"command": "disconnect",
	})
	if err != nil {
		t.Fatalf("DoCommand disconnect failed: %v", err)
	}

	connected, _ := l.IsConnected(ctx)
	if connected {
		t.Error("expected disconnected after disconnect command")
	}

	// Test connect command
	_, err = l.DoCommand(ctx, map[string]any{
		"command": "connect",
	})
	if err != nil {
		t.Fatalf("DoCommand connect failed: %v", err)
	}

	connected, _ = l.IsConnected(ctx)
	if !connected {
		t.Error("expected connected after connect command")
	}

	// Test simulate_error command
	_, err = l.DoCommand(ctx, map[string]any{
		"command": "simulate_error",
	})
	if err != nil {
		t.Fatalf("DoCommand simulate_error failed: %v", err)
	}

	stats, _ := l.GetStats(ctx)
	if stats.ErrorCount != 1 {
		t.Errorf("error count = %d, want 1", stats.ErrorCount)
	}

	// Test unknown command
	_, err = l.DoCommand(ctx, map[string]any{
		"command": "unknown",
	})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestFakeLink_Close(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "link", "test")
	l := fake.NewWithName(name)

	if err := l.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	connected, _ := l.IsConnected(ctx)
	if connected {
		t.Error("expected disconnected after Close")
	}
}
