package review

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// These tests run against PostgreSQL when TEST_DATABASE_URL is set, and are
// skipped otherwise. Point it at a dedicated test database: the test inserts
// and deletes its own rows, but also sets the shared 'today' setting.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "001_init.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	return db
}

func randomID(t *testing.T, prefix string) string {
	t.Helper()

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return prefix + "-" + hex.EncodeToString(b)
}

type fixture struct {
	sellerID  string
	groupID   string
	licenceID string
}

func setToday(t *testing.T, db *sql.DB, today string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('today', $1)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, today)
	if err != nil {
		t.Fatal(err)
	}
}

// newPendingLicence inserts a seller, group and pending licence, and removes
// them when the test finishes.
func newPendingLicence(t *testing.T, db *sql.DB, validTo *string) fixture {
	t.Helper()

	f := fixture{
		sellerID:  randomID(t, "test-seller"),
		groupID:   randomID(t, "test-group"),
		licenceID: randomID(t, "test-licence"),
	}

	t.Cleanup(func() {
		db.Exec(`DELETE FROM licences WHERE id = $1`, f.licenceID)
		db.Exec(`DELETE FROM licence_groups WHERE id = $1`, f.groupID)
		db.Exec(`DELETE FROM sellers WHERE id = $1`, f.sellerID)
	})

	if _, err := db.Exec(`INSERT INTO sellers (id, name) VALUES ($1, 'Test seller')`, f.sellerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO licence_groups (id, name) VALUES ($1, 'Test group')`, f.groupID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO licences (id, seller_id, group_id, number, issuer, valid_from, valid_to, status)
		VALUES ($1, $2, $3, 'N1', 'Ministry', '2026-01-01', $4, 'pending')
	`, f.licenceID, f.sellerID, f.groupID, validTo); err != nil {
		t.Fatal(err)
	}

	return f
}

func statusOf(t *testing.T, db *sql.DB, id string) (string, string) {
	t.Helper()

	var status string
	var reason sql.NullString
	if err := db.QueryRow(`SELECT status, rejection_reason FROM licences WHERE id = $1`, id).Scan(&status, &reason); err != nil {
		t.Fatal(err)
	}
	return status, reason.String
}

func TestDecideIntegration(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	setToday(t, db, "2026-11-21")

	t.Run("approve valid licence", func(t *testing.T) {
		future := "2027-11-21"
		f := newPendingLicence(t, db, &future)

		if err := Decide(ctx, db, f.licenceID, "approved", ""); err != nil {
			t.Fatalf("Decide: %v", err)
		}
		if status, _ := statusOf(t, db, f.licenceID); status != "approved" {
			t.Fatalf("status = %q, want approved", status)
		}
	})

	t.Run("approve licence without expiry", func(t *testing.T) {
		f := newPendingLicence(t, db, nil)

		if err := Decide(ctx, db, f.licenceID, "approved", ""); err != nil {
			t.Fatalf("Decide: %v", err)
		}
	})

	t.Run("reject stores reason", func(t *testing.T) {
		f := newPendingLicence(t, db, nil)

		if err := Decide(ctx, db, f.licenceID, "rejected", "  Illegible scan  "); err != nil {
			t.Fatalf("Decide: %v", err)
		}
		status, reason := statusOf(t, db, f.licenceID)
		if status != "rejected" || reason != "Illegible scan" {
			t.Fatalf("status=%q reason=%q, want rejected / trimmed reason", status, reason)
		}
	})

	t.Run("approving an expired licence is refused and leaves it pending", func(t *testing.T) {
		expired := "2026-11-20"
		f := newPendingLicence(t, db, &expired)

		err := Decide(ctx, db, f.licenceID, "approved", "")
		if !errors.Is(err, ErrPastExpiry) {
			t.Fatalf("err = %v, want ErrPastExpiry", err)
		}
		if status, _ := statusOf(t, db, f.licenceID); status != "pending" {
			t.Fatalf("status = %q, want pending", status)
		}
	})

	t.Run("deciding twice is refused", func(t *testing.T) {
		f := newPendingLicence(t, db, nil)

		if err := Decide(ctx, db, f.licenceID, "approved", ""); err != nil {
			t.Fatalf("first Decide: %v", err)
		}
		err := Decide(ctx, db, f.licenceID, "rejected", "changed my mind")
		if !errors.Is(err, ErrNotPending) {
			t.Fatalf("err = %v, want ErrNotPending", err)
		}
		if status, _ := statusOf(t, db, f.licenceID); status != "approved" {
			t.Fatalf("status = %q, want approved", status)
		}
	})

	t.Run("unknown licence", func(t *testing.T) {
		err := Decide(ctx, db, "does-not-exist", "approved", "")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
