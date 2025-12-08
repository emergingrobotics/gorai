// Package fake provides a fake link implementation for testing.
package fake

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorai/gorai/component/link"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("link", "fake", New)
}

// Link is a fake link for testing.
type Link struct {
	name      resource.Name
	mu        sync.RWMutex
	linkType  resource.LinkType
	direction resource.LinkDirection
	connected bool
	stats     resource.LinkStats

	// For send/receive simulation
	sendBuffer    [][]byte
	receiveBuffer [][]byte
	timeoutMS     int

	// Serial-specific
	baudRate int
	dataBits int
	stopBits float64
	parity   string

	// IP-specific
	remoteAddr string
	localAddr  string
	port       int
	protocol   string
}

// New creates a new fake link.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "link", nameStr)

	linkType := resource.LinkTypeNATS
	if lt, ok := conf["link_type"].(float64); ok {
		linkType = resource.LinkType(int(lt))
	}

	direction := resource.LinkBroadcast
	if dir, ok := conf["direction"].(float64); ok {
		direction = resource.LinkDirection(int(dir))
	}

	l := &Link{
		name:          name,
		linkType:      linkType,
		direction:     direction,
		connected:     true,
		stats:         resource.LinkStats{},
		sendBuffer:    [][]byte{},
		receiveBuffer: [][]byte{},
		timeoutMS:     5000,
		baudRate:      115200,
		dataBits:      8,
		stopBits:      1,
		parity:        "none",
		remoteAddr:    "127.0.0.1",
		localAddr:     "127.0.0.1",
		port:          4222,
		protocol:      "tcp",
	}

	return l, nil
}

// NewWithName creates a fake link with a specific resource name.
func NewWithName(name resource.Name) *Link {
	return &Link{
		name:          name,
		linkType:      resource.LinkTypeNATS,
		direction:     resource.LinkBroadcast,
		connected:     true,
		stats:         resource.LinkStats{},
		sendBuffer:    [][]byte{},
		receiveBuffer: [][]byte{},
		timeoutMS:     5000,
		baudRate:      115200,
		dataBits:      8,
		stopBits:      1,
		parity:        "none",
		remoteAddr:    "127.0.0.1",
		localAddr:     "127.0.0.1",
		port:          4222,
		protocol:      "tcp",
	}
}

// Name returns the link's resource name.
func (l *Link) Name() resource.Name {
	return l.name
}

// Reconfigure updates the link configuration.
func (l *Link) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if lt, ok := conf.GetInt("link_type"); ok {
		l.linkType = resource.LinkType(lt)
	}
	if dir, ok := conf.GetInt("direction"); ok {
		l.direction = resource.LinkDirection(dir)
	}
	if baud, ok := conf.GetInt("baud_rate"); ok {
		l.baudRate = baud
	}
	return nil
}

// DoCommand executes arbitrary commands for extensibility.
func (l *Link) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "connect":
			l.mu.Lock()
			l.connected = true
			l.mu.Unlock()
			return map[string]any{"status": "ok"}, nil
		case "disconnect":
			l.mu.Lock()
			l.connected = false
			l.mu.Unlock()
			return map[string]any{"status": "ok"}, nil
		case "get_state":
			l.mu.RLock()
			defer l.mu.RUnlock()
			return map[string]any{
				"type":      l.linkType.String(),
				"direction": l.direction.String(),
				"connected": l.connected,
			}, nil
		case "simulate_error":
			l.mu.Lock()
			l.stats.ErrorCount++
			l.mu.Unlock()
			return map[string]any{"status": "ok"}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (l *Link) Close(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.connected = false
	return nil
}

// Type returns the link transport type.
func (l *Link) Type() resource.LinkType {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.linkType
}

// Direction returns whether the link is bidirectional or broadcast.
func (l *Link) Direction() resource.LinkDirection {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.direction
}

// IsConnected returns true if the link is active.
func (l *Link) IsConnected(ctx context.Context) (bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.connected, nil
}

// GetStats returns link statistics.
func (l *Link) GetStats(ctx context.Context) (*resource.LinkStats, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	stats := l.stats // Copy
	return &stats, nil
}

// Extended interface methods

// GetProperties returns the link properties.
func (l *Link) GetProperties(ctx context.Context) (link.Properties, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return link.Properties{
		Type:               l.linkType,
		Direction:          l.direction,
		Name:               l.name.Name,
		MaxBandwidth:       1000000, // 1MB/s
		MaxMessageSize:     65536,
		SupportsQoS:        l.linkType == resource.LinkTypeNATS,
		SupportsEncryption: true,
		SupportsPersistence: l.linkType == resource.LinkTypeNATS,
	}, nil
}

// Send sends data over the link.
func (l *Link) Send(ctx context.Context, data []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.connected {
		return fmt.Errorf("link not connected")
	}

	l.sendBuffer = append(l.sendBuffer, data)
	l.stats.BytesSent += uint64(len(data))
	l.stats.MessagesSent++
	return nil
}

// Receive receives data from the link (blocking).
func (l *Link) Receive(ctx context.Context) ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.connected {
		return nil, fmt.Errorf("link not connected")
	}

	if len(l.receiveBuffer) == 0 {
		return nil, fmt.Errorf("no data available")
	}

	data := l.receiveBuffer[0]
	l.receiveBuffer = l.receiveBuffer[1:]
	l.stats.BytesReceived += uint64(len(data))
	l.stats.MessagesRecv++
	return data, nil
}

// SetTimeout sets the read/write timeout.
func (l *Link) SetTimeout(ctx context.Context, timeoutMS int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeoutMS = timeoutMS
	return nil
}

// Flush ensures all buffered data is sent.
func (l *Link) Flush(ctx context.Context) error {
	// Fake implementation - nothing to flush
	return nil
}

// Reset resets the link connection.
func (l *Link) Reset(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sendBuffer = [][]byte{}
	l.receiveBuffer = [][]byte{}
	l.stats = resource.LinkStats{}
	return nil
}

// Serial-specific methods

// GetBaudRate returns the current baud rate.
func (l *Link) GetBaudRate(ctx context.Context) (int, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.baudRate, nil
}

// SetBaudRate sets the baud rate.
func (l *Link) SetBaudRate(ctx context.Context, baud int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.baudRate = baud
	return nil
}

// GetDataBits returns the number of data bits.
func (l *Link) GetDataBits(ctx context.Context) (int, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.dataBits, nil
}

// SetDataBits sets the number of data bits.
func (l *Link) SetDataBits(ctx context.Context, bits int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.dataBits = bits
	return nil
}

// GetStopBits returns the number of stop bits.
func (l *Link) GetStopBits(ctx context.Context) (float64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.stopBits, nil
}

// SetStopBits sets the number of stop bits.
func (l *Link) SetStopBits(ctx context.Context, bits float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stopBits = bits
	return nil
}

// GetParity returns the parity mode.
func (l *Link) GetParity(ctx context.Context) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.parity, nil
}

// SetParity sets the parity mode.
func (l *Link) SetParity(ctx context.Context, parity string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.parity = parity
	return nil
}

// Available returns the number of bytes available to read.
func (l *Link) Available(ctx context.Context) (int, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	total := 0
	for _, data := range l.receiveBuffer {
		total += len(data)
	}
	return total, nil
}

// IP-specific methods

// GetRemoteAddress returns the remote address.
func (l *Link) GetRemoteAddress(ctx context.Context) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.remoteAddr, nil
}

// GetLocalAddress returns the local address.
func (l *Link) GetLocalAddress(ctx context.Context) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.localAddr, nil
}

// GetPort returns the port number.
func (l *Link) GetPort(ctx context.Context) (int, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.port, nil
}

// GetProtocol returns the protocol.
func (l *Link) GetProtocol(ctx context.Context) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.protocol, nil
}

// Test helper methods

// SetConnected sets the connection state (for testing).
func (l *Link) SetConnected(connected bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.connected = connected
}

// SetType sets the link type (for testing).
func (l *Link) SetType(linkType resource.LinkType) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.linkType = linkType
}

// SetDirection sets the link direction (for testing).
func (l *Link) SetDirection(direction resource.LinkDirection) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.direction = direction
}

// AddToReceiveBuffer adds data to the receive buffer (for testing).
func (l *Link) AddToReceiveBuffer(data []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.receiveBuffer = append(l.receiveBuffer, data)
}

// GetSendBuffer returns the send buffer contents (for testing).
func (l *Link) GetSendBuffer() [][]byte {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([][]byte, len(l.sendBuffer))
	copy(result, l.sendBuffer)
	return result
}

// SetRemoteAddress sets the remote address (for testing).
func (l *Link) SetRemoteAddress(addr string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.remoteAddr = addr
}

// SetPort sets the port (for testing).
func (l *Link) SetPort(port int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.port = port
}

// SetLatency sets the latency in stats (for testing).
func (l *Link) SetLatency(latency time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stats.Latency = latency
}

// Verify interface compliance.
var _ link.Link = (*Link)(nil)
var _ link.Extended = (*Link)(nil)
var _ link.SerialLink = (*Link)(nil)
var _ link.IPLink = (*Link)(nil)
