package gateway

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gorai/gorai/pkg/gsp"
	"github.com/gorai/gorai/pkg/nats"
	gonats "github.com/nats-io/nats.go"
	"go.bug.st/serial"
)

// PortHandler manages a single serial port's GSP communication.
type PortHandler struct {
	config    PortConfig
	gwConfig  *Config
	nc        *nats.Client
	logger    *slog.Logger

	mu        sync.RWMutex
	port      serial.Port
	parser    *gsp.Parser
	subs      map[string]*gonats.Subscription // subID -> NATS subscription
	running   bool
	stats     PortStats
	pongCh    chan struct{}
}

// PortStats holds statistics for a port.
type PortStats struct {
	BytesSent      uint64    `json:"bytes_sent"`
	BytesReceived  uint64    `json:"bytes_received"`
	FramesSent     uint64    `json:"frames_sent"`
	FramesReceived uint64    `json:"frames_received"`
	CRCErrors      uint64    `json:"crc_errors"`
	ParseErrors    uint64    `json:"parse_errors"`
	LastActivity   time.Time `json:"last_activity"`
	Connected      bool      `json:"connected"`
}

// NewPortHandler creates a new port handler.
func NewPortHandler(cfg PortConfig, nc *nats.Client, logger *slog.Logger, gwConfig *Config) *PortHandler {
	return &PortHandler{
		config:   cfg,
		gwConfig: gwConfig,
		nc:       nc,
		logger:   logger.With("device", cfg.Device, "device_id", cfg.DeviceID),
		parser:   gsp.NewParser(gsp.MaxPayloadSize + 64),
		subs:     make(map[string]*gonats.Subscription),
		pongCh:   make(chan struct{}, 1),
	}
}

// Run starts the port handler loop.
func (h *PortHandler) Run(ctx context.Context) error {
	h.mu.Lock()
	h.running = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.running = false
		h.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Connect to serial port
		if err := h.connect(); err != nil {
			h.logger.Warn("connection failed", "error", err)
			time.Sleep(h.gwConfig.ReconnectDelay)
			continue
		}

		// Run communication loop
		err := h.runLoop(ctx)
		h.disconnect()

		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			h.logger.Warn("connection lost", "error", err)
			time.Sleep(h.gwConfig.ReconnectDelay)
		}
	}
}

// Stop halts the port handler.
func (h *PortHandler) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Unsubscribe all NATS subscriptions
	for id, sub := range h.subs {
		sub.Unsubscribe()
		delete(h.subs, id)
	}

	h.disconnect()
}

// GetStats returns current port statistics.
func (h *PortHandler) GetStats() *PortStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	stats := h.stats
	stats.Connected = h.port != nil
	return &stats
}

func (h *PortHandler) connect() error {
	// Configure serial port
	mode := &serial.Mode{
		BaudRate: h.config.BaudRate,
		DataBits: h.config.DataBits,
		StopBits: serial.StopBits(h.config.StopBits),
		Parity:   parityFromString(h.config.Parity),
	}

	port, err := serial.Open(h.config.Device, mode)
	if err != nil {
		return err
	}

	// Set read timeout for non-blocking reads
	port.SetReadTimeout(100 * time.Millisecond)

	h.mu.Lock()
	h.port = port
	h.parser.Reset()
	h.mu.Unlock()

	h.logger.Info("connected")
	return nil
}

func (h *PortHandler) disconnect() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.port != nil {
		h.port.Close()
		h.port = nil
	}
}

func (h *PortHandler) runLoop(ctx context.Context) error {
	// Start ping goroutine
	pingCtx, pingCancel := context.WithCancel(ctx)
	defer pingCancel()
	go h.pingLoop(pingCtx)

	// Read loop
	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		h.mu.RLock()
		port := h.port
		h.mu.RUnlock()

		if port == nil {
			return ErrPortClosed
		}

		n, err := port.Read(buf)
		if err != nil {
			// Timeout is expected, continue
			if strings.Contains(err.Error(), "timeout") {
				continue
			}
			return err
		}

		if n == 0 {
			continue
		}

		h.mu.Lock()
		h.stats.BytesReceived += uint64(n)
		h.stats.LastActivity = time.Now()
		h.mu.Unlock()

		// Feed bytes to parser
		for i := 0; i < n; i++ {
			frame, err := h.parser.Feed(buf[i : i+1])
			if err != nil {
				h.mu.Lock()
				if err == gsp.ErrInvalidCRC {
					h.stats.CRCErrors++
				} else {
					h.stats.ParseErrors++
				}
				h.mu.Unlock()
				h.logger.Debug("parse error", "error", err)
				continue
			}

			if frame != nil {
				h.mu.Lock()
				h.stats.FramesReceived++
				h.mu.Unlock()

				if err := h.handleFrame(frame); err != nil {
					h.logger.Debug("frame handling error", "error", err)
				}
			}
		}
	}
}

func (h *PortHandler) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(h.gwConfig.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := h.sendPing(); err != nil {
				h.logger.Debug("ping send failed", "error", err)
				continue
			}

			// Wait for PONG
			select {
			case <-h.pongCh:
				// OK
			case <-time.After(h.gwConfig.PingTimeout):
				h.logger.Warn("ping timeout")
			case <-ctx.Done():
				return
			}
		}
	}
}

func (h *PortHandler) handleFrame(frame *gsp.Frame) error {
	msg, err := gsp.ParseMessage(frame.Payload)
	if err != nil {
		return err
	}

	switch msg.Command {
	case gsp.CmdPONG:
		select {
		case h.pongCh <- struct{}{}:
		default:
		}

	case gsp.CmdPUB:
		return h.handlePub(msg)

	case gsp.CmdSUB:
		return h.handleSub(msg)

	case gsp.CmdUNSUB:
		return h.handleUnsub(msg)

	case gsp.CmdPING:
		return h.sendPong()
	}

	return nil
}

func (h *PortHandler) handlePub(msg *gsp.Message) error {
	// Translate device subject to full NATS subject
	// Device sends: "pwm.command"
	// NATS gets:    "gorai.robot1.mcu.pwm.command"
	natsSubject := h.config.SubjectPrefix + "." + h.config.DeviceID + "." + msg.Subject

	h.logger.Debug("publish", "subject", natsSubject, "len", len(msg.Payload))
	return h.nc.Publish(natsSubject, msg.Payload)
}

func (h *PortHandler) handleSub(msg *gsp.Message) error {
	natsSubject := h.config.SubjectPrefix + "." + h.config.DeviceID + "." + msg.Subject

	sub, err := h.nc.Subscribe(natsSubject, func(m *gonats.Msg) {
		// Strip prefix to get device-local subject
		prefix := h.config.SubjectPrefix + "." + h.config.DeviceID + "."
		deviceSubject := strings.TrimPrefix(m.Subject, prefix)
		h.sendToDevice(msg.SubID, deviceSubject, m.Data)
	})
	if err != nil {
		h.sendErr("SUB_FAILED", err.Error())
		return err
	}

	h.mu.Lock()
	h.subs[msg.SubID] = sub
	h.mu.Unlock()

	h.logger.Debug("subscribe", "subject", natsSubject, "id", msg.SubID)
	return h.sendOK()
}

func (h *PortHandler) handleUnsub(msg *gsp.Message) error {
	h.mu.Lock()
	sub, ok := h.subs[msg.SubID]
	if ok {
		sub.Unsubscribe()
		delete(h.subs, msg.SubID)
	}
	h.mu.Unlock()

	h.logger.Debug("unsubscribe", "id", msg.SubID)
	return h.sendOK()
}

func (h *PortHandler) sendToDevice(subID, subject string, payload []byte) error {
	msgPayload := gsp.FormatMsg(subject, subID, payload)
	return h.sendFrame(msgPayload)
}

func (h *PortHandler) sendPing() error {
	return h.sendFrame(gsp.FormatPing())
}

func (h *PortHandler) sendPong() error {
	return h.sendFrame(gsp.FormatPong())
}

func (h *PortHandler) sendOK() error {
	return h.sendFrame(gsp.FormatOK())
}

func (h *PortHandler) sendErr(code, message string) error {
	return h.sendFrame(gsp.FormatErr(code, message))
}

func (h *PortHandler) sendFrame(payload []byte) error {
	frame, err := gsp.BuildFrame(payload)
	if err != nil {
		return err
	}

	h.mu.Lock()
	port := h.port
	h.mu.Unlock()

	if port == nil {
		return ErrPortClosed
	}

	n, err := port.Write(frame)
	if err != nil {
		return err
	}

	h.mu.Lock()
	h.stats.BytesSent += uint64(n)
	h.stats.FramesSent++
	h.mu.Unlock()

	return nil
}

func parityFromString(s string) serial.Parity {
	switch strings.ToLower(s) {
	case "odd":
		return serial.OddParity
	case "even":
		return serial.EvenParity
	case "mark":
		return serial.MarkParity
	case "space":
		return serial.SpaceParity
	default:
		return serial.NoParity
	}
}
