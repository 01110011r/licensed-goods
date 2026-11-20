package review

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound         = errors.New("licence not found")
	ErrNotPending       = errors.New("licence is not pending")
	ErrInvalidDecision  = errors.New("decision must be approved or rejected")
	ErrReasonRequired   = errors.New("rejection reason is required")
	ErrPastExpiry       = errors.New("cannot approve an expired licence")
	ErrInvalidDates     = errors.New("valid_to cannot be before valid_from")
	ErrUnknownReference = errors.New("seller or licence group does not exist")
)

func Decide(
	ctx context.Context,
	db *sql.DB,
	id string,
	decision string,
	reason string,
) error {
	if decision != "approved" && decision != "rejected" {
		return ErrInvalidDecision
	}

	reason = strings.TrimSpace(reason)
	if decision == "rejected" && reason == "" {
		return ErrReasonRequired
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var sellerID, groupID string
	var validFrom time.Time
	var validTo sql.NullTime
	var status string

	err = tx.QueryRowContext(ctx, `
		SELECT seller_id, group_id, valid_from, valid_to, status
		FROM licences
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(
		&sellerID,
		&groupID,
		&validFrom,
		&validTo,
		&status,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("load licence: %w", err)
	}

	if status != "pending" {
		return ErrNotPending
	}

	if decision == "approved" {
		var todayValue string

		err = tx.QueryRowContext(ctx, `
			SELECT value FROM app_settings WHERE key = 'today'
		`).Scan(&todayValue)
		if err != nil {
			return fmt.Errorf("load today setting: %w", err)
		}

		today, err := time.Parse("2006-01-02", todayValue)
		if err != nil {
			return fmt.Errorf("parse today setting: %w", err)
		}

		if validTo.Valid {
			if validTo.Time.Before(today) {
				return ErrPastExpiry
			}
			if validTo.Time.Before(validFrom) {
				return ErrInvalidDates
			}
		}

		var referencesExist bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM sellers WHERE id = $1
			) AND EXISTS (
				SELECT 1 FROM licence_groups WHERE id = $2
			)
		`, sellerID, groupID).Scan(&referencesExist)

		if err != nil {
			return err
		}
		if !referencesExist {
			return ErrUnknownReference
		}
	}

	var rejectionReason any
	if decision == "rejected" {
		rejectionReason = reason
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE licences
		SET status = $1, rejection_reason = $2
		WHERE id = $3
	`, decision, rejectionReason, id)

	if err != nil {
		return fmt.Errorf("update licence: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit review decision: %w", err)
	}

	return nil
}
