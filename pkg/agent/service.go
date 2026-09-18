package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

// PromptHandler processes one decoded request and streams its response
// chunks through Send on the provided Stream. The handler MUST call
// stream.Ack() first (the protocol requires an ack chunk before any
// latency-inducing work) and MUST call stream.Done() or
// stream.Error() exactly once to close the stream.
//
// Returning without calling Done/Error is treated as an internal error:
// the service responds with a 500 and the empty terminator.
type PromptHandler func(req *Request, stream *Stream)

// Request is a decoded prompt endpoint request handed to a
// PromptHandler.
type Request struct {
	// Envelope is the decoded request (§5.3 discrimination applied).
	Envelope *RequestEnvelope
	// Reply is the reply subject the handler must not use directly —
	// use Stream.Send instead. Exposed for correlation/debugging.
	Reply string
	// Subject is the subject the request arrived on.
	Subject string
}

// ServiceConfig configures an Agent service instance.
type ServiceConfig struct {
	// Agent is the canonical harness identifier (metadata.agent),
	// e.g. "claude-code" or "gorai-llm".
	Agent string
	// AgentAbbr is the (optional) short subject token for the harness
	// (metadata may use the full Agent; the subject uses this).
	// Defaults to Agent when empty.
	AgentAbbr string
	// Owner is the operator or account owning this instance.
	Owner string
	// Name is the instance name (the 5th subject token). For
	// session-aware harnesses this is the session label.
	Name string
	// Session optionally distinguishes session-aware harnesses in
	// metadata. Empty for session-less instances.
	Session string
	// Version is the semver of the harness implementation (NOT the
	// protocol version).
	Version string
	// Description is human-readable, surfaced by nats micro tools.
	Description string
	// MaxPayload is the endpoint's declared max single-request payload,
	// e.g. "1MB". Defaults to "1MB".
	MaxPayload string
	// AttachmentsOK declares whether the endpoint accepts attachments.
	AttachmentsOK bool
	// HeartbeatInterval is the heartbeat cadence. Defaults to
	// DefaultHeartbeatInterval.
	HeartbeatInterval time.Duration
	// Logger defaults to slog.Default().
	Logger *slog.Logger
}

// normalize fills defaults and validates identity.
func (c *ServiceConfig) normalize() error {
	if c.Agent == "" {
		return errors.New("agent: ServiceConfig.Agent is required")
	}
	if c.Owner == "" {
		return errors.New("agent: ServiceConfig.Owner is required")
	}
	if c.Name == "" {
		return errors.New("agent: ServiceConfig.Name is required")
	}
	if c.AgentAbbr == "" {
		c.AgentAbbr = c.Agent
	}
	if c.Version == "" {
		c.Version = "0.1.0"
	}
	if c.MaxPayload == "" {
		c.MaxPayload = "1MB"
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	return nil
}

// Agent is a Synadia Agent Protocol v0.3 service instance.
//
// Start registers the "agents" micro service (prompt + status endpoints),
// and begins publishing the periodic heartbeat. Stop undoes all of it.
type Agent struct {
	cfg        *ServiceConfig
	nc         *nats.Conn
	svc        micro.Service
	instanceID string
	stopHB     chan struct{}
	hbDone     chan struct{}
}

// NewAgent creates an agent bound to the given NATS connection. It is
// inert until Start is called.
func NewAgent(nc *nats.Conn, cfg *ServiceConfig) (*Agent, error) {
	if nc == nil {
		return nil, errors.New("agent: nil NATS connection")
	}
	if cfg == nil {
		return nil, errors.New("agent: nil ServiceConfig")
	}
	if err := cfg.normalize(); err != nil {
		return nil, err
	}
	return &Agent{cfg: cfg, nc: nc}, nil
}

// Identity returns the subject identity tokens in use.
func (a *Agent) Identity() (agent, owner, name string) {
	return a.cfg.AgentAbbr, a.cfg.Owner, a.cfg.Name
}

// promptSubject is the default endpoint subject for this instance.
func (a *Agent) promptSubject() string {
	return Subject(VerbPrompt, a.cfg.AgentAbbr, a.cfg.Owner, a.cfg.Name)
}

// statusSubject is the default endpoint subject for this instance.
func (a *Agent) statusSubject() string {
	return Subject(VerbStatus, a.cfg.AgentAbbr, a.cfg.Owner, a.cfg.Name)
}

// Start registers the service and starts the heartbeat loop.
func (a *Agent) Start(handler PromptHandler) error {
	ident := micro.Config{
		Name:    ServiceName,
		Version: a.cfg.Version,
		Metadata: map[string]string{
			"agent":            a.cfg.Agent,
			"owner":            a.cfg.Owner,
			"protocol_version": ProtocolVersion,
		},
	}
	if a.cfg.Session != "" {
		ident.Metadata["session"] = a.cfg.Session
	}
	if a.cfg.Description != "" {
		ident.Description = a.cfg.Description
	}

	svc, err := micro.AddService(a.nc, ident)
	if err != nil {
		return fmt.Errorf("agent: registering %q service: %w", ServiceName, err)
	}
	a.svc = svc
	a.instanceID = svc.Info().ID

	// prompt endpoint: queue group "agents" (protocol-fixed, §3.3).
	if err := svc.AddEndpoint(VerbPrompt, promptHandlerFunc(a, handler),
		micro.WithEndpointSubject(a.promptSubject()),
		micro.WithEndpointQueueGroup(QueueGroup),
		micro.WithEndpointMetadata(map[string]string{
			"max_payload":    a.cfg.MaxPayload,
			"attachments_ok": fmt.Sprintf("%t", a.cfg.AttachmentsOK),
		}),
	); err != nil {
		svc.Stop()
		return fmt.Errorf("agent: adding prompt endpoint: %w", err)
	}

	// status endpoint (§8.7): same queue group; replies with the
	// heartbeat payload shape.
	if err := svc.AddEndpoint(VerbStatus, statusHandlerFunc(a),
		micro.WithEndpointSubject(a.statusSubject()),
		micro.WithEndpointQueueGroup(QueueGroup),
	); err != nil {
		svc.Stop()
		return fmt.Errorf("agent: adding status endpoint: %w", err)
	}

	a.startHeartbeat()
	a.cfg.Logger.Info("agent started",
		"agent", a.cfg.Agent, "owner", a.cfg.Owner, "name", a.cfg.Name,
		"prompt", a.promptSubject(), "status", a.statusSubject())
	return nil
}

// Stop undoes Start.
func (a *Agent) Stop() error {
	if a.stopHB != nil {
		close(a.stopHB)
		<-a.hbDone
		a.stopHB = nil
	}
	if a.svc != nil {
		return a.svc.Stop()
	}
	return nil
}

// startHeartbeat publishes the §8.3 payload on the fixed heartbeat
// subject every HeartbeatInterval, until Stop.
func (a *Agent) startHeartbeat() {
	a.stopHB = make(chan struct{})
	a.hbDone = make(chan struct{})
	go func() {
		defer close(a.hbDone)
		ticker := time.NewTicker(a.cfg.HeartbeatInterval)
		defer ticker.Stop()
		// Publish one immediately so callers that subscribe-before-
		// discover (§8.5) don't wait a full interval.
		a.publishHeartbeat()
		for {
			select {
			case <-a.stopHB:
				return
			case <-ticker.C:
				a.publishHeartbeat()
			}
		}
	}()
}

func (a *Agent) publishHeartbeat() {
	hb := a.heartbeatPayload()
	data, err := json.Marshal(hb)
	if err != nil {
		a.cfg.Logger.Error("agent: heartbeat marshal", "error", err)
		return
	}
	if err := a.nc.Publish(HeartbeatSubject(a.cfg.AgentAbbr, a.cfg.Owner, a.cfg.Name), data); err != nil {
		a.cfg.Logger.Warn("agent: heartbeat publish", "error", err)
	}
}

// heartbeatPayload builds the §8.3 payload. The status endpoint uses the
// same builder so the two stay in lockstep (§8.7.1).
func (a *Agent) heartbeatPayload() Heartbeat {
	return Heartbeat{
		Agent:      a.cfg.Agent,
		Owner:      a.cfg.Owner,
		Session:    a.cfg.Session,
		InstanceID: a.instanceID,
		TS:         time.Now().UTC(),
		IntervalS:  a.cfg.HeartbeatInterval.Seconds(),
	}
}

func statusHandlerFunc(a *Agent) micro.HandlerFunc {
	return func(req micro.Request) {
		data, err := json.Marshal(a.heartbeatPayload())
		if err != nil {
			_ = req.Error(ErrCodeInternalError, "building status payload", nil)
			return
		}
		if err := req.Respond(data); err != nil {
			a.cfg.Logger.Warn("agent: status respond", "error", err)
		}
	}
}

// promptHandlerFunc adapts a PromptHandler to a micro.HandlerFunc: it
// applies the §5.3/§5.4 checks, builds the Stream, and guarantees the
// stream is closed even if the handler misbehaves.
func promptHandlerFunc(a *Agent, handler PromptHandler) micro.HandlerFunc {
	return func(req micro.Request) {
		raw := req.Data()

		// §5.4 client-side capability checks, mirrored server-side:
		// oversize and attachments-when-not-allowed are 400s.
		if len(raw) > a.cfg.MaxPayloadBytes() {
			_ = req.Error(ErrCodeBadRequest, "request exceeds max_payload", nil)
			return
		}
		env, err := DecodeRequest(raw)
		if err != nil {
			_ = req.Error(ErrCodeBadRequest, err.Error(), nil)
			return
		}
		if len(env.Attachments) > 0 && !a.cfg.AttachmentsOK {
			_ = req.Error(ErrCodeBadRequest, "attachments not accepted by this endpoint",
				mustMarshalError(StatusPayloadError{Error: "attachments_not_allowed"}))
			return
		}

		stream := newStream(a, req)
		handler(&Request{Envelope: env, Reply: req.Reply(), Subject: req.Subject()}, stream)

		// Handler contract: Done/Error called exactly once. If it
		// returned without closing, close with 500 so the caller never
		// hangs on a lost terminator.
		if !stream.closed() {
			a.cfg.Logger.Warn("agent: prompt handler returned without closing stream",
				"subject", req.Subject())
			_ = req.Error(ErrCodeInternalError, "handler did not complete the stream",
				mustMarshalError(StatusPayloadError{Error: "handler_error"}))
		}
	}
}

func mustMarshalError(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// MaxPayloadBytes parses the declared "512KB"/"1MB"-style MaxPayload
// string. It reads the leading integer and an optional B/KB/MB/GB unit;
// unparseable values fall back to 1 MiB.
func (c *ServiceConfig) MaxPayloadBytes() int {
	const fallback = 1 << 20
	s := strings.TrimSpace(c.MaxPayload)
	var factor int64 = 1
	lower := strings.ToLower(s)
	switch {
	case strings.HasSuffix(lower, "kb"):
		factor = 1024
		s = s[:len(s)-2]
	case strings.HasSuffix(lower, "mb"):
		factor = 1024 * 1024
		s = s[:len(s)-2]
	case strings.HasSuffix(lower, "gb"):
		factor = 1024 * 1024 * 1024
		s = s[:len(s)-2]
	case strings.HasSuffix(lower, "b"):
		s = s[:len(s)-1]
	}
	s = strings.TrimSpace(s)
	// Read the leading integer digits only.
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return fallback
	}
	n, err := strconv.ParseInt(s[:i], 10, 63)
	if err != nil || n <= 0 {
		return fallback
	}
	return int(n * factor)
}
