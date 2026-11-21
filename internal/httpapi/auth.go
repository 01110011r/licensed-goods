package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// requireAdmin allows a request only when it carries the configured
// administrator token as a bearer credential. If no token is configured, every
// request is refused (fail closed).
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.adminToken == "" {
			http.Error(w, "administrator access is not configured", http.StatusForbidden)
			return
		}

		header := r.Header.Get("Authorization")
		token, found := strings.CutPrefix(header, "Bearer ")

		if !found || subtle.ConstantTimeCompare([]byte(token), []byte(s.adminToken)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="administrator"`)
			http.Error(w, "administrator token is missing or incorrect", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
