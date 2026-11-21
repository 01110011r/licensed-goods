package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// requireReviewer allows a request only when it carries the configured
// reviewer token as a bearer credential. If no token is configured, every
// request is refused (fail closed).
func (s *Server) requireReviewer(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.reviewerToken == "" {
			http.Error(w, "reviewer access is not configured", http.StatusForbidden)
			return
		}

		header := r.Header.Get("Authorization")
		token, found := strings.CutPrefix(header, "Bearer ")

		if !found || subtle.ConstantTimeCompare([]byte(token), []byte(s.reviewerToken)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="reviewer"`)
			http.Error(w, "reviewer token is missing or incorrect", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
