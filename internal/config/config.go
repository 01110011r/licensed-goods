package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	UploadDir   string
	// AdminToken authorises administrator-only endpoints.
	AdminToken string
}

func Load() Config {
	return Config{
		Port:        getEnv("APP_PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		UploadDir:   getEnv("UPLOAD_DIR", "./uploads"),
		AdminToken:  os.Getenv("ADMIN_TOKEN"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
