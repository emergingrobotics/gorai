package agent

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/emergingrobotics/gorai/pkg/embeddednats"
)

// startTestServer boots an embedded NATS server on an ephemeral port.
func startTestServer(t *testing.T, jetStream bool) *nats.Conn {
	t.Helper()
	server, err := embeddednats.New(embeddednats.Config{
		Host:         "127.0.0.1",
		Port:         -1,
		JetStream:    jetStream,
		JetStreamDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("embeddednats.New: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("server.Start: %v", err)
	}
	t.Cleanup(server.Shutdown)

	nc, err := nats.Connect(server.ClientURL())
	if err != nil {
		t.Fatalf("nats.Connect: %v", err)
	}
	t.Cleanup(nc.Close)
	return nc
}

func TestDecodeRequest_PlainText(t *testing.T) {
	env, err := DecodeRequest([]byte("  hello world\n"))
	if err != nil {
		t.Fatalf("DecodeRequest: %v", err)
	}
	if env.Prompt != "hello world\n" {
		t.Fatalf("prompt = %q", env.Prompt)
	}
}

func TestDecodeRequest_Envelope(t *testing.T) {
	env, err := DecodeRequest([]byte(`{"prompt": "do it", "attachments": [{"filename": "a.txt", "content": "xx"}], "custom_field": 42}`))
	if err != nil {
		t.Fatalf("DecodeRequest: %v", err)
	}
	if env.Prompt != "do it" {
		t.Fatalf("prompt = %q", env.Prompt)
	}
	if len(env.Attachments) != 1 || env.Attachments[0].Filename != "a.txt" {
		t.Fatalf("attachments = %#v", env.Attachments)
	}
	if _, ok := env.Extra["custom_field"]; !ok {
		t.Fatalf("unknown field not preserved: %#v", env.Extra)
	}
}

func TestDecodeRequest_Errors(t *testing.T) {
	if _, err := DecodeRequest(nil); err != ErrEmptyPayload {
		t.Fatalf("nil payload: %v", err)
	}
	if _, err := DecodeRequest([]byte("   ")); err != ErrEmptyPayload {
		t.Fatalf("blank payload: %v", err)
	}
	if _, err := DecodeRequest([]byte(`{"no_prompt": 1}`)); err != ErrMalformedEnvelope {
		t.Fatalf("missing prompt: %v", err)
	}
	if _, err := DecodeRequest([]byte(`{not json`)); err == nil {
		t.Fatal("bad json should error")
	}
}

func TestEnvelopeRoundTripPreservesUnknownFields(t *testing.T) {
	in := []byte(`{"prompt": "x", "future_flag": true}`)
	var env RequestEnvelope
	if err := env.UnmarshalJSON(in); err != nil {
		t.Fatal(err)
	}
	out, err := env.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var env2 RequestEnvelope
	if err := env2.UnmarshalJSON(out); err != nil {
		t.Fatal(err)
	}
	if _, ok := env2.Extra["future_flag"]; !ok {
		t.Fatalf("unknown field lost on round-trip: %s", out)
	}
}

func TestSubjects(t *testing.T) {
	if got := Subject(VerbPrompt, "cc", "aconnolly", "synadia-com-2"); got != "agents.prompt.cc.aconnolly.synadia-com-2" {
		t.Fatalf("Subject = %q", got)
	}
	if got := HeartbeatSubject("cc", "o", "n"); got != "agents.hb.cc.o.n" {
		t.Fatalf("HeartbeatSubject = %q", got)
	}
}

func TestMaxPayloadBytes(t *testing.T) {
	cases := map[string]int{
		"512KB": 512 * 1024,
		"1MB":   1 << 20,
		"2GB":   2 << 30,
		"100B":  100,
		"42":    42,
		"bogus": 1 << 20,
		"":      1 << 20,
		"3.5MB": 3 << 20, // leading integer read; fractional part ignored
	}
	for in, want := range cases {
		cfg := &ServiceConfig{MaxPayload: in}
		if got := cfg.MaxPayloadBytes(); got != want {
			t.Errorf("MaxPayloadBytes(%q) = %d, want %d", in, got, want)
		}
	}
}

// startAgent brings up a full protocol agent with a canned handler.
func startAgent(t *testing.T, nc *nats.Conn, handler PromptHandler) *Agent {
	t.Helper()
	ag, err := NewAgent(nc, &ServiceConfig{
		Agent:             "test-bot",
		Owner:             "tester",
		Name:              "bench-1",
		Version:           "1.0.0",
		Description:       "test agent",
		HeartbeatInterval: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	if err := ag.Start(handler); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		if err := ag.Stop(); err != nil {
			t.Logf("Stop: %v", err)
		}
	})
	return ag
}

func TestDiscover(t *testing.T) {
	nc := startTestServer(t, false)

	insts, err := Discover(nc, time.Second)
	if err == nil {
		t.Fatalf("expected ErrNoAgents, got %v", insts)
	}
	if err != ErrNoAgents {
		t.Fatalf("err = %v, want ErrNoAgents", err)
	}

	startAgent(t, nc, func(req *Request, s *Stream) {
		_ = s.Ack()
		_ = s.Response("ok")
		_ = s.Done()
	})

	insts, err = Discover(nc, 2*time.Second)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(insts) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(insts), insts)
	}
	inst := insts[0]
	if inst.Agent != "test-bot" || inst.Owner != "tester" || inst.Name != "bench-1" {
		t.Fatalf("identity = %#v", inst)
	}
	if inst.ProtocolVersion != ProtocolVersion {
		t.Fatalf("protocol_version = %q", inst.ProtocolVersion)
	}
	if inst.PromptSubject != "agents.prompt.test-bot.tester.bench-1" {
		t.Fatalf("prompt subject = %q", inst.PromptSubject)
	}
	if inst.StatusSubject != "agents.status.test-bot.tester.bench-1" {
		t.Fatalf("status subject = %q", inst.StatusSubject)
	}
	if inst.MaxPayload != "1MB" || inst.AttachmentsOK {
		t.Fatalf("endpoint metadata = %#v", inst)
	}
	if inst.ID == "" {
		t.Fatal("missing instance id")
	}
}

func TestPrompt_Streaming(t *testing.T) {
	nc := startTestServer(t, false)
	startAgent(t, nc, func(req *Request, s *Stream) {
		if req.Envelope.Prompt != "stream me" {
			t.Errorf("prompt = %q", req.Envelope.Prompt)
		}
		_ = s.Ack()
		_ = s.Response("part 1")
		_ = s.Response("part 2")
		_ = s.Done()
	})

	insts, err := Discover(nc, 2*time.Second)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	chunks, errs := Prompt(nc, insts[0], []byte("stream me"), &PromptOpts{InactivityTimeout: 2 * time.Second})
	var got []string
	for c := range chunks {
		switch c.Type {
		case ChunkStatus:
			got = append(got, "status:"+c.Data.(string))
		case ChunkResponse:
			got = append(got, c.Data.(string))
		default:
			got = append(got, c.Type)
		}
	}
	if err := <-errs; err != ErrStreamEnded {
		t.Fatalf("terminal error = %v", err)
	}
	want := []string{"status:ack", "part 1", "part 2"}
	if len(got) != len(want) {
		t.Fatalf("chunks = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chunks = %v, want %v", got, want)
		}
	}
}

func TestPrompt_ErrorResponse(t *testing.T) {
	nc := startTestServer(t, false)
	startAgent(t, nc, func(req *Request, s *Stream) {
		_ = s.Ack()
		_ = s.Error(ErrCodeBadRequest, "nope", &StatusPayloadError{Error: "invalid_input"})
	})

	insts, err := Discover(nc, 2*time.Second)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	_, errs := Prompt(nc, insts[0], []byte("hi"), nil)
	err = <-errs
	var se *ServiceError
	if ok := errorsAs(err, &se); !ok || se.Code != ErrCodeBadRequest || se.Msg != "nope" {
		t.Fatalf("error = %v, want ServiceError 400/nope", err)
	}
}

func TestPrompt_ClientSideChecks(t *testing.T) {
	nc := startTestServer(t, false)
	startAgent(t, nc, func(req *Request, s *Stream) {
		_ = s.Ack()
		_ = s.Response("ok")
		_ = s.Done()
	})
	insts, err := Discover(nc, 2*time.Second)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	// Oversize payload is rejected before publishing.
	big := make([]byte, (1<<20)+1)
	_, errs := Prompt(nc, insts[0], big, nil)
	if err := <-errs; err == nil {
		t.Fatal("oversize payload accepted")
	}

	// Attachments to an attachments_ok=false endpoint are rejected.
	_, errs = Prompt(nc, insts[0], []byte(`{"prompt":"x","attachments":[{"filename":"a","content":"YQ=="}]}`), nil)
	if err := <-errs; err == nil {
		t.Fatal("attachments accepted on attachments_ok=false endpoint")
	}
}

func TestPrompt_MidStreamQuery(t *testing.T) {
	nc := startTestServer(t, false)
	startAgent(t, nc, func(req *Request, s *Stream) {
		_ = s.Ack()
		_ = s.Response("before")
		_ = s.Query(QueryData{ID: "q1", ReplySubject: nc.NewInbox(), Prompt: "continue?"})
		_ = s.Response("after")
		_ = s.Done()
	})
	insts, err := Discover(nc, 2*time.Second)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	var gotPrompt string
	chunks, errs := Prompt(nc, insts[0], []byte("run"), &PromptOpts{
		OnQuery: func(q *QueryData) ([]byte, error) {
			gotPrompt = q.Prompt
			return []byte("yes"), nil
		},
	})
	for range chunks {
	}
	if err := <-errs; err != ErrStreamEnded {
		t.Fatalf("terminal error = %v", err)
	}
	if gotPrompt != "continue?" {
		t.Fatalf("query prompt = %q", gotPrompt)
	}
}

func TestStatusEndpoint(t *testing.T) {
	nc := startTestServer(t, false)
	startAgent(t, nc, func(req *Request, s *Stream) {
		_ = s.Ack()
		_ = s.Response("ok")
		_ = s.Done()
	})
	insts, err := Discover(nc, 2*time.Second)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	msg, err := nc.Request(insts[0].StatusSubject, nil, 2*time.Second)
	if err != nil {
		t.Fatalf("status request: %v", err)
	}
	var hb Heartbeat
	if err := json.Unmarshal(msg.Data, &hb); err != nil {
		t.Fatalf("status payload not heartbeat-shaped: %v (%s)", err, msg.Data)
	}
	if hb.InstanceID != insts[0].ID || hb.Agent != "test-bot" || hb.IntervalS <= 0 {
		t.Fatalf("status payload = %#v", hb)
	}
}

func TestHeartbeat_Liveness(t *testing.T) {
	nc := startTestServer(t, false)
	ln, err := NewLiveness(nc)
	if err != nil {
		t.Fatalf("NewLiveness: %v", err)
	}
	defer ln.Close()

	startAgent(t, nc, func(req *Request, s *Stream) {
		_ = s.Ack()
		_ = s.Response("ok")
		_ = s.Done()
	})
	insts, err := Discover(nc, 2*time.Second)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	// Wait for at least one heartbeat beat (50ms cadence in test agent).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !ln.Alive(insts[0].ID) {
		time.Sleep(20 * time.Millisecond)
	}
	if !ln.Alive(insts[0].ID) {
		t.Fatalf("agent not alive after heartbeats (known=%v)", ln.Known())
	}
}

func TestServiceConfigValidation(t *testing.T) {
	nc := startTestServer(t, false)
	_, err := NewAgent(nc, &ServiceConfig{Agent: "", Owner: "o", Name: "n"})
	if err == nil {
		t.Fatal("empty agent accepted")
	}
	_, err = NewAgent(nc, nil)
	if err == nil {
		t.Fatal("nil config accepted")
	}
}

func TestStream_DoesNotSendAfterClose(t *testing.T) {
	nc := startTestServer(t, false)
	ag, err := NewAgent(nc, &ServiceConfig{Agent: "a", Owner: "o", Name: "n"})
	if err != nil {
		t.Fatal(err)
	}
	if err := ag.Start(func(req *Request, s *Stream) {
		_ = s.Ack()
		_ = s.Done()
		_ = s.Response("should not appear")
		_ = s.Done()
		_ = s.Error(ErrCodeInternalError, "also should not appear", nil)
	}); err != nil {
		t.Fatal(err)
	}
	defer ag.Stop()

	// Drive the endpoint directly to observe wire behavior: publish with
	// our own inbox as the reply and collect every message on it.
	inbox := nc.NewInbox()
	ch := make(chan *nats.Msg, 16)
	sub, err := nc.ChanSubscribe(inbox, ch)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()
	if err := nc.PublishRequest(Subject(VerbPrompt, "a", "o", "n"), inbox, []byte("hi")); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// The whole stream should be: ack chunk, then the empty terminator —
	// nothing after close (the post-close Response/Error are no-ops).
	var got [][]byte
	for i := 0; i < 2; i++ {
		select {
		case m, ok := <-ch:
			if !ok {
				break
			}
			got = append(got, m.Data)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for message %d; got %#v", i, got)
		}
	}
	if len(got) != 2 {
		t.Fatalf("stream messages = %d, want 2 (ack + terminator); data=%q", len(got), got)
	}
	if len(got[0]) == 0 || !strings.Contains(string(got[0]), `"ack"`) {
		t.Fatalf("first message should be the ack chunk, got %q", got[0])
	}
	if len(got[1]) != 0 {
		t.Fatalf("terminator should be empty, got %q", got[1])
	}
}

// errorsAs is a local alias so the test doesn't import just for this.
func errorsAs(err error, target **ServiceError) bool {
	se, ok := err.(*ServiceError)
	if !ok {
		return false
	}
	*target = se
	return true
}
