package agent

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

// Stream is the writer side of a response stream (§6). A handler
// receives one per prompt request and must:
//
//  1. call Ack() first — the protocol requires the {type:"status",
//     data:"ack"} chunk before any latency-inducing work;
//  2. emit content via Response (and Status/Query as needed);
//  3. close exactly once via Done() (success) or Error() (failure).
//
// Sending after a close is ignored (the protocol forbids messages after
// the terminator).
type Stream struct {
	nc    *nats.Conn
	reply string

	mu   sync.Mutex
	done bool
}

func newStream(a *Agent, req micro.Request) *Stream {
	return &Stream{nc: a.nc, reply: req.Reply()}
}

// Ack sends the mandatory first chunk: {type:"status", data:"ack"}.
func (s *Stream) Ack() error {
	return s.send(ChunkStatus, StatusAck)
}

// Response sends a content chunk. Data may be a plain string (response
// text) or a struct/object (e.g. {text, attachments}).
func (s *Stream) Response(data any) error {
	return s.send(ChunkResponse, data)
}

// Status sends a lifecycle-signal chunk. Only StatusAck is defined in
// v0.3; callers MUST ignore unknown values.
func (s *Stream) Status(token string) error {
	return s.send(ChunkStatus, token)
}

// Query pauses the stream to ask the caller a question (§7). It returns
// after publishing the query chunk; the handler then waits for the
// single answer on q.ReplySubject via QueryReply.
func (s *Stream) Query(q QueryData) error {
	return s.send(ChunkQuery, q)
}

// QueryReply waits for the single answer to a mid-stream query (§7.2).
// The answer is either plain UTF-8 text or an envelope (§5.1); it is
// returned raw. A nil payload is returned on timeout (the caller never
// answered).
func (s *Stream) QueryReply(subject string, timeout time.Duration) ([]byte, error) {
	if timeout <= 0 {
		timeout = DefaultInactivityTimeout
	}
	msg, err := s.nc.Request(subject, nil, timeout)
	if err != nil {
		// No answer within the timeout: the caller didn't reply (§7.3).
		if errors.Is(err, nats.ErrTimeout) {
			return nil, nil
		}
		return nil, err
	}
	return msg.Data, nil
}

// Done closes the stream successfully: it publishes the empty-payload,
// headerless terminator.
func (s *Stream) Done() error {
	return s.close(nil)
}

// Error closes the stream with a protocol error (§9): an
// Nats-Service-Error-Code/-Error headered message, then the terminator.
// body may carry a StatusPayloadError JSON object.
func (s *Stream) Error(code, description string, body any) error {
	hdr := nats.Header{
		micro.ErrorCodeHeader: []string{code},
		micro.ErrorHeader:     []string{description},
	}
	var data []byte
	if body != nil {
		if b, err := json.Marshal(body); err == nil {
			data = b
		}
	}
	return s.close(&nats.Msg{Header: hdr, Data: data})
}

// closed reports whether the stream has been terminated (used by the
// service's handler-contract check).
func (s *Stream) closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

func (s *Stream) send(typ string, data any) error {
	if s.closed() {
		return nil
	}
	payload, err := json.Marshal(Chunk{Type: typ, Data: data})
	if err != nil {
		return err
	}
	return s.nc.Publish(s.reply, payload)
}

// close publishes an optional error message then the empty terminator,
// and marks the stream closed.
func (s *Stream) close(errMsg *nats.Msg) error {
	s.mu.Lock()
	if s.done {
		s.mu.Unlock()
		return nil
	}
	s.done = true
	s.mu.Unlock()

	if errMsg != nil {
		errMsg.Subject = s.reply
		if err := s.nc.PublishMsg(errMsg); err != nil {
			return err
		}
	}
	return s.nc.Publish(s.reply, nil)
}
