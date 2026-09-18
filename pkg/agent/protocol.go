// Package agent implements the Synadia Agent Protocol for NATS (v0.3) — a
// NATS-native wire contract for identifying, discovering, and communicating
// with AI agents.
//
// Gorai adopts this protocol as the native on-bus dialect for AI agents
// (see VISION.md, "Agents on the Bus"). An AI agent on the gorai mesh is a
// NATS micro service named "agents" with prompt/status endpoints, a
// fixed-subject heartbeat, and chunked, typed response streams.
//
// Protocol reference:
//
//	https://github.com/synadia-ai/synadia-agent-sdk-docs/blob/main/core-protocol.md
//
// What we adopt:
//   - service registration under the shared "agents" service name, with
//     discovery via $SRV.PING / $SRV.INFO (callers learn endpoint subjects
//     from $SRV.INFO — never construct them from identity)
//   - the verb-first subject convention agents.{verb}.{agent}.{owner}.{name}
//   - request envelopes (plain-text shorthand or {prompt, attachments})
//   - typed response streams: {type, data} chunks, ack-first, empty-payload
//     terminator, error headers, inactivity timeout, cancel-by-unsubscribing
//   - mid-stream queries
//   - fixed-subject heartbeat + on-demand status endpoint, dead at 3x interval
//
// What we do NOT adopt:
//   - "security delegated to the broker" — on gorai the trust model is
//     enforced at the capability node (NCP), and on Gorai Loom by the
//     filamesh crypto ladder; the agent protocol rides whatever transport
//     already authenticates the bus
//   - KV/JetStream-backed agent registries for constrained nodes — $SRV
//     discovery is enough on the bus, and mesh KV stays for channel/schema
//     catalogs
package agent

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Protocol constants (Synadia Agent Protocol for NATS, v0.3).

const (
	// ProtocolVersion is the MAJOR.MINOR protocol version this package
	// implements. Declared in service metadata as "protocol_version".
	ProtocolVersion = "0.3"

	// ServiceName is the shared NATS micro service name. Every compliant
	// agent registers under this exact name; it is the discovery filter
	// that separates protocol agents from other services on the bus.
	ServiceName = "agents"

	// QueueGroup is the queue group all protocol instances use on their
	// request/reply endpoints. Fixed protocol-wide so that mixed-SDK
	// deployments load-balance identically.
	QueueGroup = "agents"

	// DefaultHeartbeatInterval is the recommended heartbeat cadence.
	DefaultHeartbeatInterval = 30 * time.Second

	// DeadThreshold is how many missed heartbeat intervals make an
	// instance look dead (3 x interval_s).
	DeadThreshold = 3

	// DefaultInactivityTimeout bounds a response stream when chunks stop
	// arriving (e.g. the terminator was lost).
	DefaultInactivityTimeout = 60 * time.Second
)

// SubjectRoot is the fixed first token of every protocol subject.
const SubjectRoot = "agents"

// Verb tokens. Each protocol endpoint owns its own positional slot so
// wildcard subscriptions partition cleanly.
const (
	VerbPrompt      = "prompt"
	VerbHeartbeat   = "hb" // abbreviated on purpose: heartbeat traffic dominates subject volume
	VerbStatus      = "status"
	VerbAttachments = "attachments" // reserved; not implemented in v0.3
)

// Chunk type discriminators for response streams (§6.2).
const (
	ChunkResponse = "response"
	ChunkStatus   = "status"
	ChunkQuery    = "query"
)

// StatusAck is the one v0.3 status chunk value. It MUST be the first
// chunk on the reply subject; the terminal done signal is the
// empty-payload terminator, not a chunk.
const StatusAck = "ack"

// Service error codes (§9.2). Use these verbatim in RespondError.
const (
	ErrCodeBadRequest    = "400" // malformed envelope, empty payload, oversize request
	ErrCodeUnauthorized  = "401"
	ErrCodeForbidden     = "403"
	ErrCodeNotFound      = "404"
	ErrCodeConflict      = "409"
	ErrCodeRateLimited   = "429"
	ErrCodeInternalError = "500"
)

// Errors returned by the helpers in this package.
var (
	ErrEmptyPayload      = errors.New("agent: empty request payload is invalid")
	ErrMalformedEnvelope = errors.New("agent: request is neither plain text nor a valid prompt envelope")
)

// Subject builds a protocol subject for the given identity:
//
//	agents.prompt.cc.aconnolly.synadia-com-2
//
// agent/owner/name are the identity tokens; verb is one of the protocol
// verbs. Callers must still satisfy NATS subject naming rules (lowercase
// a-z, 0-9, '-', '_') for the identity tokens.
func Subject(verb, agent, owner, name string) string {
	return strings.Join([]string{SubjectRoot, verb, agent, owner, name}, ".")
}

// HeartbeatSubject builds the protocol-fixed heartbeat subject
// (agents.hb.{agent}.{owner}.{name}). This is the one subject callers MAY
// construct without a $SRV lookup.
func HeartbeatSubject(agent, owner, name string) string {
	return Subject(VerbHeartbeat, agent, owner, name)
}

// RequestEnvelope is the JSON form of a prompt request (§5.1). The plain
// UTF-8 form is just an envelope with only Prompt set; DecodeRequest
// handles both. Unknown top-level fields are preserved in Extra so a
// relay can forward them verbatim (§5.6 forward-compat).
type RequestEnvelope struct {
	Prompt      string
	Attachments []Attachment
	Extra       map[string]json.RawMessage
}

// UnmarshalJSON records unknown top-level fields in Extra.
func (r *RequestEnvelope) UnmarshalJSON(b []byte) error {
	var p struct {
		Prompt      string       `json:"prompt"`
		Attachments []Attachment `json:"attachments"`
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	r.Prompt = p.Prompt
	r.Attachments = p.Attachments

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		if k == "prompt" || k == "attachments" {
			continue
		}
		if r.Extra == nil {
			r.Extra = make(map[string]json.RawMessage)
		}
		r.Extra[k] = v
	}
	return nil
}

// MarshalJSON re-emits preserved unknown fields (§5.6: MUST preserve when
// relaying).
func (r *RequestEnvelope) MarshalJSON() ([]byte, error) {
	wire := struct {
		Prompt      string       `json:"prompt"`
		Attachments []Attachment `json:"attachments,omitempty"`
	}{Prompt: r.Prompt, Attachments: r.Attachments}
	b, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	if len(r.Extra) == 0 {
		return b, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, err
	}
	for k, v := range r.Extra {
		if _, exists := obj[k]; !exists {
			obj[k] = v
		}
	}
	return json.Marshal(obj)
}

// Attachment is a base64-encoded file carried in a request or
// response (§5.2). No MIME type is carried; receivers sniff content.
type Attachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// DecodeRequest applies the §5.3 discrimination rule to a raw payload:
// leading UTF-8 whitespace is skipped; a payload starting with '{' must
// parse as an envelope with a non-empty prompt string; anything else is
// plain UTF-8 text promoted to an envelope. A zero-byte payload is
// invalid.
func DecodeRequest(payload []byte) (*RequestEnvelope, error) {
	start := 0
	for start < len(payload) && (payload[start] == 0x09 || payload[start] == 0x0A ||
		payload[start] == 0x0D || payload[start] == 0x20) {
		start++
	}
	if start == len(payload) {
		return nil, ErrEmptyPayload
	}

	if payload[start] == '{' {
		var env RequestEnvelope
		if err := json.Unmarshal(payload[start:], &env); err != nil {
			return nil, errors.Join(ErrMalformedEnvelope, err)
		}
		if env.Prompt == "" {
			return nil, ErrMalformedEnvelope
		}
		return &env, nil
	}

	return &RequestEnvelope{Prompt: string(payload[start:])}, nil
}

// Chunk is a typed response-stream message (§6.2). Every non-terminating
// chunk on the reply subject MUST have this shape; the terminator is a
// zero-byte body with no headers.
type Chunk struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// QueryData is the payload of a mid-stream query chunk (§7.1).
type QueryData struct {
	ID           string       `json:"id"`
	ReplySubject string       `json:"reply_subject"`
	Prompt       string       `json:"prompt"`
	Attachments  []Attachment `json:"attachments,omitempty"`
}

// Heartbeat is the payload published on the fixed heartbeat subject
// (§8.3) and replied by the status endpoint (§8.7).
type Heartbeat struct {
	Agent      string    `json:"agent"`
	Owner      string    `json:"owner"`
	Session    string    `json:"session,omitempty"`
	InstanceID string    `json:"instance_id"`
	TS         time.Time `json:"ts"`
	IntervalS  float64   `json:"interval_s"`
}

// StatusPayloadError is the optional JSON body of an error response
// (§9.1).
type StatusPayloadError struct {
	Error       string `json:"error"`
	Message     string `json:"message,omitempty"`
	RetryAfterS int    `json:"retry_after_s,omitempty"`
}
