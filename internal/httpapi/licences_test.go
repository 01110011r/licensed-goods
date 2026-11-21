package httpapi

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Each case below is rejected before any database access, so a nil DB is safe.

func TestSubmitLicenceRejectsJSON(t *testing.T) {
	s := New(nil, t.TempDir(), "secret")

	req := httptest.NewRequest(http.MethodPost, "/api/licences",
		strings.NewReader(`{"seller_id":"s1","FilePath":"/etc/passwd"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.submitLicence(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415", rec.Code)
	}
}

func TestSubmitLicenceRequiresFile(t *testing.T) {
	s := New(nil, t.TempDir(), "secret")

	body, contentType := multipartBody(t, map[string]string{
		"seller_id":  "s1",
		"group_id":   "MED",
		"number":     "N1",
		"issuer":     "Ministry",
		"valid_from": "2026-11-21",
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/licences", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	s.submitLicence(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestSubmitLicenceRejectsContentMismatch(t *testing.T) {
	s := New(nil, t.TempDir(), "secret")

	// Plain text named as a PDF must be refused.
	body, contentType := multipartBody(t, map[string]string{
		"seller_id":  "s1",
		"group_id":   "MED",
		"number":     "N1",
		"issuer":     "Ministry",
		"valid_from": "2026-11-21",
	}, &upload{name: "licence.pdf", data: []byte("just some text")})

	req := httptest.NewRequest(http.MethodPost, "/api/licences", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	s.submitLicence(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "does not match") {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestSubmitLicenceRejectsUnsupportedExtension(t *testing.T) {
	s := New(nil, t.TempDir(), "secret")

	body, contentType := multipartBody(t, map[string]string{
		"seller_id":  "s1",
		"group_id":   "MED",
		"number":     "N1",
		"issuer":     "Ministry",
		"valid_from": "2026-11-21",
	}, &upload{name: "licence.exe", data: []byte("%PDF-1.4 fake")})

	req := httptest.NewRequest(http.MethodPost, "/api/licences", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	s.submitLicence(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

type upload struct {
	name string
	data []byte
}

func multipartBody(t *testing.T, fields map[string]string, file *upload) (*bytes.Buffer, string) {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}

	if file != nil {
		part, err := writer.CreateFormFile("file", file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(file.data); err != nil {
			t.Fatal(err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	return &buf, writer.FormDataContentType()
}
