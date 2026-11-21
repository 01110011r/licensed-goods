package config

import "os"

type Config struct {
	Port        string
	Environment string
	DatabaseURL string
	UploadDir   string
	// ReviewerToken authorises reviewer-only endpoints.
	ReviewerToken string
}

func Load() Config {
	return Config{
		Port:          getEnv("APP_PORT", "8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		ReviewerToken: os.Getenv("REVIEWER_TOKEN"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
