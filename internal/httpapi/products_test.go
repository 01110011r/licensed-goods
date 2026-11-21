package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Each case is rejected before any database access, so a nil DB is safe.
func TestCreateProductValidationBeforeDatabase(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "malformed JSON", body: `{"seller_id":`, want: http.StatusBadRequest},
		{name: "missing name", body: `{"seller_id":"s1","category_id":"c8","ikpu":"22042100999999999"}`, want: http.StatusBadRequest},
		{name: "short IKPU", body: `{"seller_id":"s1","name":"x","category_id":"c8","ikpu":"123"}`, want: http.StatusBadRequest},
		{name: "bad status", body: `{"seller_id":"s1","name":"x","category_id":"c8","ikpu":"22042100999999999","status":"live"}`, want: http.StatusBadRequest},
	}

	s := New(nil, t.TempDir(), "secret")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/products", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			s.createProduct(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}
