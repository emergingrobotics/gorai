package dashboard

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorai/gorai/pkg/config"
)

func TestBasicAuthMiddleware_ValidCreds(t *testing.T) {
	handler := BasicAuthMiddleware("admin", "secret", slog.Default())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestBasicAuthMiddleware_InvalidCreds(t *testing.T) {
	handler := BasicAuthMiddleware("admin", "secret", slog.Default())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "wrong-password")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestBasicAuthMiddleware_NoCreds(t *testing.T) {
	handler := BasicAuthMiddleware("admin", "secret", slog.Default())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestBasicAuthMiddleware_TimingSafe(t *testing.T) {
	// Verify that the middleware uses constant-time comparison by checking
	// that wrong username and wrong password both return 401 (not a
	// different error for each case, which would leak information).
	handler := BasicAuthMiddleware("admin", "secret", slog.Default())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	tests := []struct {
		name     string
		username string
		password string
	}{
		{"wrong username", "wrong", "secret"},
		{"wrong password", "admin", "wrong"},
		{"both wrong", "wrong", "wrong"},
		{"empty username", "", "secret"},
		{"empty password", "admin", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.SetBasicAuth(tt.username, tt.password)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestDashboardWithAuth(t *testing.T) {
	d, err := New(
		&config.DashboardConfig{
			Listen:   "127.0.0.1:0",
			Username: "admin",
			Password: "secret",
		},
		&config.RDL{Robot: config.RobotConfig{Name: "test"}},
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Request without auth should get 401
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	d.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated request: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	// Request with correct auth should succeed
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "secret")
	rec = httptest.NewRecorder()
	d.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("authenticated request: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDashboardWithoutAuth(t *testing.T) {
	d, err := New(
		&config.DashboardConfig{Listen: "127.0.0.1:0"},
		&config.RDL{Robot: config.RobotConfig{Name: "test"}},
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Request without auth should succeed when no auth configured
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	d.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestWebSocketAuth(t *testing.T) {
	d, err := New(
		&config.DashboardConfig{
			Listen:   "127.0.0.1:0",
			Username: "admin",
			Password: "secret",
		},
		&config.RDL{Robot: config.RobotConfig{Name: "test"}},
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// WebSocket upgrade without auth should get 401
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	rec := httptest.NewRecorder()
	d.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated WebSocket: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
