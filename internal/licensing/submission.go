package licensing

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidSubmission = errors.New("invalid licence submission")
	ErrUnknownSeller     = errors.New("seller does not exist")
	ErrUnknownGroup      = errors.New("licence group does not exist")
	ErrPendingExists     = errors.New("a pending application already exists")
)

type LicenceSubmission struct {
	SellerID  string
	GroupID   string
	Number    string
	Issuer    string
	ValidFrom string
	ValidTo   string
	// FilePath is set by the server after an upload is saved. It is never
	// read from client input.
	FilePath string `json:"-"`
}

func validateLicenceSubmission(
	input LicenceSubmission,
) (LicenceSubmission, time.Time, any, error) {
	input.SellerID = strings.TrimSpace(input.SellerID)
	input.GroupID = strings.TrimSpace(input.GroupID)
	input.Number = strings.TrimSpace(input.Number)
	input.Issuer = strings.TrimSpace(input.Issuer)

	if input.SellerID == "" ||
		input.GroupID == "" ||
		input.Number == "" ||
		input.Issuer == "" ||
		input.ValidFrom == "" ||
		input.FilePath == "" {
		return input, time.Time{}, nil, ErrInvalidSubmission
	}

	validFrom, err := time.Parse("2006-01-02", input.ValidFrom)
	if err != nil {
		return input, time.Time{}, nil, ErrInvalidSubmission
	}

	var validTo any
	if input.ValidTo != "" {
		parsed, err := time.Parse("2006-01-02", input.ValidTo)
		if err != nil || parsed.Before(validFrom) {
			return input, time.Time{}, nil, ErrInvalidSubmission
		}
		validTo = parsed
	}

	return input, validFrom, validTo, nil
}

func SubmitLicence(
	ctx context.Context,
	db *sql.DB,
	input LicenceSubmission,
) (string, error) {

	input, validFrom, validTo, err := validateLicenceSubmission(input)
	if err != nil {
		return "", err
	}

	var exists bool

	err = db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM sellers WHERE id = $1)`,
		input.SellerID,
	).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("check seller: %w", err)
	}
	if !exists {
		return "", ErrUnknownSeller
	}

	err = db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM licence_groups WHERE id = $1)`,
		input.GroupID,
	).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("check licence group: %w", err)
	}
	if !exists {
		return "", ErrUnknownGroup
	}

	randomID := make([]byte, 16)
	if _, err := rand.Read(randomID); err != nil {
		return "", fmt.Errorf("generate licence ID: %w", err)
	}
	id := "lic-" + hex.EncodeToString(randomID)

	_, err = db.ExecContext(ctx, `
		INSERT INTO licences (
			id, seller_id, group_id, number, issuer,
			valid_from, valid_to, status, file_path
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8)
	`,
		id,
		input.SellerID,
		input.GroupID,
		input.Number,
		input.Issuer,
		validFrom,
		validTo,
		input.FilePath,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" {
			return "", ErrPendingExists
		}
		return "", fmt.Errorf("insert licence: %w", err)
	}

	return id, nil
}
