package settings

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrInvalidDate = errors.New("today must use YYYY-MM-DD format")

func GetToday(ctx context.Context, db *sql.DB) (string, error) {
	var today string

	err := db.QueryRowContext(ctx, `
		SELECT value
		FROM app_settings
		WHERE key = 'today'
	`).Scan(&today)

	if err != nil {
		return "", err
	}

	return today, nil
}

func validateToday(value string) error {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return ErrInvalidDate
	}

	return nil
}

func SetToday(ctx context.Context, db *sql.DB, value string) error {
	err := validateToday(value)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value)
		VALUES ('today', $1)
		ON CONFLICT (key)
		DO UPDATE SET value = EXCLUDED.value
	`, value)

	return err
}
