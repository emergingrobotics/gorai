package health

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestServer() *Server {
	return New(Config{
		Listen: "127.0.0.1:0",
		Logger: slog.Default(),
	})
}

func parseResponse(t *testing.T, response *http.Response) statusResponse {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	response.Body.Close()
	var result statusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parsing JSON response %q: %v", string(body), err)
	}
	return result
}

func TestLivezReturns200Immediately(t *testing.T) {
	server := newTestServer()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/livez", nil)

	server.handler().ServeHTTP(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	result := parseResponse(t, response)
	if result.Status != "alive" {
		t.Fatalf("expected status %q, got %q", "alive", result.Status)
	}
}

func TestHealthzReturns503BeforeSetReady(t *testing.T) {
	server := newTestServer()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	server.handler().ServeHTTP(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", response.StatusCode)
	}
	result := parseResponse(t, response)
	if result.Status != "not_ready" {
		t.Fatalf("expected status %q, got %q", "not_ready", result.Status)
	}
}

func TestHealthzReturns200AfterSetReady(t *testing.T) {
	server := newTestServer()
	server.SetReady()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	server.handler().ServeHTTP(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	result := parseResponse(t, response)
	if result.Status != "ready" {
		t.Fatalf("expected status %q, got %q", "ready", result.Status)
	}
}

func TestHealthzReturns503AfterSetNotReady(t *testing.T) {
	server := newTestServer()
	server.SetReady()
	server.SetNotReady()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	server.handler().ServeHTTP(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", response.StatusCode)
	}
	result := parseResponse(t, response)
	if result.Status != "not_ready" {
		t.Fatalf("expected status %q, got %q", "not_ready", result.Status)
	}
}

func TestIsReadyReturnsCorrectState(t *testing.T) {
	server := newTestServer()

	if server.IsReady() {
		t.Fatal("expected IsReady() to be false before SetReady()")
	}

	server.SetReady()
	if !server.IsReady() {
		t.Fatal("expected IsReady() to be true after SetReady()")
	}

	server.SetNotReady()
	if server.IsReady() {
		t.Fatal("expected IsReady() to be false after SetNotReady()")
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	server := newTestServer()

	if err := server.Start(); err != nil {
		t.Fatalf("starting server: %v", err)
	}

	// Verify the server is listening by making a real HTTP request.
	address := server.Address()
	if address == "" {
		t.Fatal("expected non-empty listener address after Start()")
	}

	response, err := http.Get("http://" + address + "/livez")
	if err != nil {
		t.Fatalf("GET /livez: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	response.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf("shutting down server: %v", err)
	}
}

func TestServerStartAndShutdownWithHealthz(t *testing.T) {
	server := newTestServer()

	if err := server.Start(); err != nil {
		t.Fatalf("starting server: %v", err)
	}

	address := server.Address()

	// Before SetReady: /healthz should return 503.
	response, err := http.Get("http://" + address + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 before SetReady(), got %d", response.StatusCode)
	}
	response.Body.Close()

	// After SetReady: /healthz should return 200.
	server.SetReady()
	response, err = http.Get("http://" + address + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz after SetReady(): %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 after SetReady(), got %d", response.StatusCode)
	}
	response.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf("shutting down server: %v", err)
	}
}

func TestResponseContentType(t *testing.T) {
	server := newTestServer()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/livez", nil)

	server.handler().ServeHTTP(recorder, request)

	response := recorder.Result()
	contentType := response.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected Content-Type %q, got %q", "application/json", contentType)
	}
	response.Body.Close()
}

func TestDefaultListenAddress(t *testing.T) {
	server := New(Config{
		Logger: slog.Default(),
	})
	if server.listenAddress != "127.0.0.1:4180" {
		t.Fatalf("expected default listen address %q, got %q", "127.0.0.1:4180", server.listenAddress)
	}
}
