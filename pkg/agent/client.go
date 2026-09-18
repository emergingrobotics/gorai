package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

// ErrStreamEnded is returned on the Prompt error channel when the
// stream terminates successfully (empty-payload terminator observed).
var ErrStreamEnded = errors.New("agent: stream ended")

// ErrNoAgents is returned by Discover when no agents answer.
var ErrNoAgents = errors.New("agent: no agents discovered")

// Instance is one discovered agent instance.
type Instance struct {
	// ID is the per-instance service id (framework-assigned).
	ID string
	// Agent, Owner, Session are the metadata fields (§3.2).
	Agent   string
	Owner   string
	Session string
	// ProtocolVersion is metadata.protocol_version (MAJOR.MINOR).
	ProtocolVersion string
	// Name is the instance name: last token of the prompt endpoint
	// subject, per §4.3.
	Name string
	// PromptSubject is the prompt endpoint subject as reported by
	// $SRV.INFO — the only subject callers may address (§4.3: MUST NOT
	// construct from identity alone).
	PromptSubject string
	// StatusSubject is the status endpoint subject, if reported.
	StatusSubject string
	// MaxPayload and AttachmentsOK are the prompt endpoint's declared
	// capabilities (§2.1).
	MaxPayload    string
	AttachmentsOK bool
}

// Discover runs $SRV.INFO against the shared "agents" service name and
// returns every compliant agent instance on the bus. Records that are
// not compliant (wrong service name, missing protocol_version, no
// prompt endpoint) are skipped.
func Discover(nc *nats.Conn, timeout time.Duration) ([]Instance, error) {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	subj, err := micro.ControlSubject(micro.InfoVerb, ServiceName, "")
	if err != nil {
		return nil, err
	}

	var (
		mu   sync.Mutex
		seen = map[string]bool{}
		out  []Instance
	)

	inbox := nc.NewInbox()
	sub, err := nc.Subscribe(inbox, func(m *nats.Msg) {
		inst, perr := parseInstance(m.Data)
		if perr != nil {
			return
		}
		mu.Lock()
		if !seen[inst.ID] {
			seen[inst.ID] = true
			out = append(out, inst)
		}
		mu.Unlock()
	})
	defer sub.Unsubscribe()

	if err := nc.PublishRequest(subj, inbox, nil); err != nil {
		return nil, fmt.Errorf("agent: $SRV.INFO %q: %w", ServiceName, err)
	}

	// Drain until the window closes. $SRV has no reply-count header, so
	// a time window is the standard approach (the nats CLI's
	// --replies=0 does the same).
	deadline := time.After(timeout)
	<-deadline

	mu.Lock()
	defer mu.Unlock()
	if len(out) == 0 {
		return nil, ErrNoAgents
	}
	return out, nil
}

// parseInstance decodes one $SRV.INFO response and locates the prompt
// endpoint per §4.3.
func parseInstance(data []byte) (Instance, error) {
	var raw struct {
		ID        string               `json:"id"`
		Name      string               `json:"name"`
		Version   string               `json:"version"`
		Metadata  map[string]string    `json:"metadata"`
		Endpoints []micro.EndpointInfo `json:"endpoints"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Instance{}, err
	}
	if raw.Name != ServiceName {
		return Instance{}, fmt.Errorf("service %q is not the %q service", raw.Name, ServiceName)
	}
	if raw.Metadata["protocol_version"] == "" {
		return Instance{}, fmt.Errorf("service %q has no protocol_version metadata", raw.ID)
	}
	inst := Instance{
		ID:              raw.ID,
		Agent:           raw.Metadata["agent"],
		Owner:           raw.Metadata["owner"],
		Session:         raw.Metadata["session"],
		ProtocolVersion: raw.Metadata["protocol_version"],
	}
	for _, e := range raw.Endpoints {
		switch e.Name {
		case VerbPrompt:
			inst.PromptSubject = e.Subject
			if e.Metadata != nil {
				inst.MaxPayload = e.Metadata["max_payload"]
				inst.AttachmentsOK = e.Metadata["attachments_ok"] == "true"
			}
			inst.Name = instanceNameFromSubject(e.Subject, inst.Session)
		case VerbStatus:
			inst.StatusSubject = e.Subject
		}
	}
	if inst.PromptSubject == "" {
		return Instance{}, fmt.Errorf("service %q has no %q endpoint", raw.ID, VerbPrompt)
	}
	return inst, nil
}

// instanceNameFromSubject extracts the last subject token (the instance
// name). Falls back to the session for non-default subjects (§4.3).
func instanceNameFromSubject(subject, session string) string {
	tokens := strings.Split(subject, ".")
	if len(tokens) >= 5 {
		return tokens[len(tokens)-1]
	}
	return session
}

// PromptOpts configures a streaming Prompt call.
type PromptOpts struct {
	// InactivityTimeout bounds how long to wait between chunks; it also
	// protects against a lost terminator. Defaults to
	// DefaultInactivityTimeout.
	InactivityTimeout time.Duration
	// OnQuery, if non-nil, answers mid-stream queries (§7) instead of
	// letting them time out. Return the answer bytes (plain text or an
	// envelope, §7.2) and a nil error to publish them; a nil reply or
	// error is treated as "no answer".
	OnQuery func(q *QueryData) ([]byte, error)
}

// Prompt sends a prompt to the agent and streams its response chunks.
//
// payload may be plain UTF-8 text or a JSON envelope (§5.1). The chunk
// channel yields typed chunks in publication order (ack chunks included);
// it is closed when the stream terminates. The error channel yields
// exactly one value: ErrStreamEnded (success), a *ServiceError
// (protocol error, §9), or a transport error (inactivity timeout,
// connection loss).
//
// Cancellation: drop the channels; interest-based delivery discards
// remaining chunks at the server at no cost (§6.7). The agent is not
// notified of cancellation.
//
// The request window (first-chunk wait) is 10s — long enough for a
// cold model call to emit its mandatory ack; from the ack onward the
// inactivity timer governs.
func Prompt(nc *nats.Conn, inst Instance, payload []byte, opts *PromptOpts) (<-chan Chunk, <-chan error) {
	chunks := make(chan Chunk, 16)
	errs := make(chan error, 1)

	// §5.4 client-side capability checks before publishing (spare the
	// agent a round trip).
	maxBytes := 1 << 20
	if inst.MaxPayload != "" {
		maxBytes = parseMaxPayload(inst.MaxPayload)
	}
	if len(payload) > maxBytes {
		close(chunks)
		errs <- fmt.Errorf("agent: payload %d bytes exceeds endpoint max_payload %s",
			len(payload), inst.MaxPayload)
		close(errs)
		return chunks, errs
	}
	var probe RequestEnvelope
	if err := json.Unmarshal(payload, &probe); err == nil {
		if len(probe.Attachments) > 0 && !inst.AttachmentsOK {
			close(chunks)
			errs <- fmt.Errorf("agent: endpoint %q does not accept attachments", inst.PromptSubject)
			close(errs)
			return chunks, errs
		}
	}

	go func() {
		defer close(chunks)
		defer close(errs)

		inactivity := DefaultInactivityTimeout
		var onQuery func(*QueryData) ([]byte, error)
		if opts != nil {
			if opts.InactivityTimeout > 0 {
				inactivity = opts.InactivityTimeout
			}
			if opts.OnQuery != nil {
				onQuery = opts.OnQuery
			}
		}

		// All chunks — the mandatory ack first (§6.4), content, and the
		// empty terminator (§6.5) — are published to the request's reply
		// subject, which we choose as a fresh inbox. nc.Request would
		// only deliver the first reply; a classic inbox subscription
		// receives the whole stream.
		inbox := nc.NewInbox()
		ch := make(chan *nats.Msg, 16)
		sub, err := nc.ChanSubscribe(inbox, ch)
		if err != nil {
			errs <- err
			return
		}
		defer sub.Unsubscribe()

		if err := nc.PublishRequest(inst.PromptSubject, inbox, payload); err != nil {
			errs <- fmt.Errorf("agent: prompt request: %w", err)
			return
		}

		handleChunk := func(m *nats.Msg) bool {
			if code := m.Header.Get(micro.ErrorCodeHeader); code != "" {
				errs <- &ServiceError{Code: code, Msg: m.Header.Get(micro.ErrorHeader), Body: m.Data}
				return false
			}
			if len(m.Data) == 0 && m.Header == nil {
				errs <- ErrStreamEnded // successful terminator (§6.5)
				return false
			}
			var c Chunk
			if err := json.Unmarshal(m.Data, &c); err != nil {
				return true // not a typed chunk; keep reading
			}
			if c.Type == ChunkQuery {
				var q QueryData
				if json.Unmarshal(mustJSON(c.Data), &q) == nil && onQuery != nil {
					if reply, qerr := onQuery(&q); qerr == nil && reply != nil {
						_ = nc.Publish(q.ReplySubject, reply)
					}
				}
			}
			chunks <- c
			return true
		}

		// The first chunk (ack) may be delayed by a cold model call —
		// allow a long initial window. From the first observed chunk
		// onward the timer bounds each wait, which is exactly the §6.6
		// inactivity rule ("since the last observed chunk") and also
		// guards against a lost terminator.
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		sawFirst := false

		for {
			select {
			case <-timer.C:
				if sawFirst {
					errs <- fmt.Errorf("agent: inactivity timeout after %s", inactivity)
				} else {
					errs <- fmt.Errorf("agent: no first chunk within 10s")
				}
				return
			case m, ok := <-ch:
				if !ok {
					// Reply subscription gone before the terminator.
					errs <- fmt.Errorf("agent: reply subscription closed before terminator")
					return
				}
				sawFirst = true
				if !handleChunk(m) {
					return
				}
				timer.Reset(inactivity)
			}
		}
	}()

	return chunks, errs
}

// ServiceError is a protocol error response (§9.2).
type ServiceError struct {
	Code string
	Msg  string
	Body []byte
}

func (e *ServiceError) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("agent: service error %s: %s", e.Code, e.Msg)
	}
	return fmt.Sprintf("agent: service error %s", e.Code)
}

// parseMaxPayload parses "512KB"/"1MB"-style strings; unparseable -> 1 MiB.
func parseMaxPayload(s string) int {
	cfg := &ServiceConfig{MaxPayload: s}
	return cfg.MaxPayloadBytes()
}

// mustJSON best-effort marshals; used to re-encode chunk data.
func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// Liveness tracks heartbeat observations per instance_id (§8) and
// decides death at 3 x interval_s since the last beat (§8.2).
//
// Subscribe-before-discover (§8.5): start the tracker before running
// Discover so the first beat isn't missed.
type Liveness struct {
	nc  *nats.Conn
	sub *nats.Subscription

	mu     sync.Mutex
	last   map[string]time.Time
	interv map[string]time.Duration
}

// NewLiveness subscribes to all heartbeats (agents.hb.*.*.*) and begins
// tracking them.
func NewLiveness(nc *nats.Conn) (*Liveness, error) {
	ln := &Liveness{
		nc:     nc,
		last:   make(map[string]time.Time),
		interv: make(map[string]time.Duration),
	}
	sub, err := nc.Subscribe(SubjectRoot+"."+VerbHeartbeat+".*.*.*", func(m *nats.Msg) {
		var hb Heartbeat
		if err := json.Unmarshal(m.Data, &hb); err != nil || hb.InstanceID == "" {
			return
		}
		ln.Observe(hb)
	})
	if err != nil {
		return nil, err
	}
	ln.sub = sub
	return ln, nil
}

// Observe feeds a heartbeat-shaped payload into the tracker. A status
// endpoint reply (§8.7) may be fed the same way.
func (ln *Liveness) Observe(hb Heartbeat) {
	if hb.InstanceID == "" {
		return
	}
	interval := time.Duration(hb.IntervalS * float64(time.Second))
	if interval <= 0 {
		interval = DefaultHeartbeatInterval
	}
	ln.mu.Lock()
	ln.last[hb.InstanceID] = time.Now()
	ln.interv[hb.InstanceID] = interval
	ln.mu.Unlock()
}

// Alive reports whether the instance beat within 3 x its heartbeat
// interval. An instance that never beat is not alive.
func (ln *Liveness) Alive(instanceID string) bool {
	ln.mu.Lock()
	defer ln.mu.Unlock()
	last, ok := ln.last[instanceID]
	if !ok {
		return false
	}
	interval := ln.interv[instanceID]
	if interval <= 0 {
		interval = DefaultHeartbeatInterval
	}
	return time.Since(last) < time.Duration(DeadThreshold*int64(interval))
}

// Known returns the instance ids ever observed.
func (ln *Liveness) Known() []string {
	ln.mu.Lock()
	defer ln.mu.Unlock()
	out := make([]string, 0, len(ln.last))
	for k := range ln.last {
		out = append(out, k)
	}
	return out
}

// Close stops the heartbeat subscription.
func (ln *Liveness) Close() error {
	if ln.sub != nil {
		return ln.sub.Unsubscribe()
	}
	return nil
}
