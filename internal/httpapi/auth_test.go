package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAdmin(t *testing.T) {
	ok := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}

	tests := []struct {
		name   string
		token  string
		header string
		want   int
	}{
		{name: "no token configured refuses everything", token: "", header: "Bearer anything", want: http.StatusForbidden},
		{name: "missing header", token: "secret", header: "", want: http.StatusUnauthorized},
		{name: "wrong scheme", token: "secret", header: "Basic secret", want: http.StatusUnauthorized},
		{name: "wrong token", token: "secret", header: "Bearer guess", want: http.StatusUnauthorized},
		{name: "token prefix only", token: "secret", header: "Bearer secre", want: http.StatusUnauthorized},
		{name: "correct token", token: "secret", header: "Bearer secret", want: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(nil, "", tt.token)
			h := s.requireAdmin(ok)

			req := httptest.NewRequest(http.MethodGet, "/api/reviews", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			h(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

// Reviewer routes must refuse requests without a token before touching the
// database, so a nil DB is safe here.
func TestAdminRoutesRequireToken(t *testing.T) {
	s := New(nil, "", "secret")
	handler := s.Handler()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/reviews"},
		{http.MethodPatch, "/api/reviews/lic-1"},
		{http.MethodPut, "/api/settings/today"},
		{http.MethodPost, "/api/licences"},
		{http.MethodPost, "/api/products"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
		})
	}
}
