package dashboard

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
)

// BasicAuthMiddleware returns middleware that enforces HTTP Basic Auth
// using constant-time comparison for both username and password.
func BasicAuthMiddleware(username, password string, logger *slog.Logger) func(http.Handler) http.Handler {
	usernameBytes := []byte(username)
	passwordBytes := []byte(password)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok ||
				subtle.ConstantTimeCompare([]byte(u), usernameBytes) != 1 ||
				subtle.ConstantTimeCompare([]byte(p), passwordBytes) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="gorai"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				if logger != nil {
					logger.Warn("Unauthorized dashboard access attempt", "remote", r.RemoteAddr)
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
