package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_PORT", "")
	t.Setenv("UPLOAD_DIR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("ADMIN_TOKEN", "")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.UploadDir != "./uploads" {
		t.Errorf("UploadDir = %q, want ./uploads", cfg.UploadDir)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
	if cfg.AdminToken != "" {
		t.Errorf("AdminToken = %q, want empty", cfg.AdminToken)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("APP_PORT", "9090")
	t.Setenv("UPLOAD_DIR", "/data/uploads")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("ADMIN_TOKEN", "token-123")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.UploadDir != "/data/uploads" {
		t.Errorf("UploadDir = %q, want /data/uploads", cfg.UploadDir)
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.AdminToken != "token-123" {
		t.Errorf("AdminToken = %q", cfg.AdminToken)
	}
}
