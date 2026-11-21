package catalogue

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

// Runs against PostgreSQL when TEST_DATABASE_URL is set. Use a dedicated test
// database: the test inserts its own rows and removes them afterwards.
func openCatalogueTestDB(t *testing.T) *sql.DB {
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

	// ListProducts needs the shared 'today' setting; a fresh database has none.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value) VALUES ('today', '2026-11-21')
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`); err != nil {
		t.Fatalf("set today: %v", err)
	}
	return db
}

func uniqueID(t *testing.T, prefix string) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return prefix + "-" + hex.EncodeToString(b)
}

func TestCreateProductIntegration(t *testing.T) {
	db := openCatalogueTestDB(t)
	ctx := context.Background()

	// Fixture: a seller, a category, and a licence group whose category rule
	// covers that category. The seller has no licence for the group.
	seller := uniqueID(t, "test-seller")
	category := uniqueID(t, "test-cat")
	group := uniqueID(t, "test-group")

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}

	mustExec(`INSERT INTO sellers (id, name) VALUES ($1, 'Test seller')`, seller)
	mustExec(`INSERT INTO categories (id, name) VALUES ($1, 'Test category')`, category)
	mustExec(`INSERT INTO licence_groups (id, name) VALUES ($1, 'Test group')`, group)
	mustExec(`INSERT INTO licence_group_rules (group_id, kind, category_id, excluded)
	          VALUES ($1, 'category', $2, false)`, group, category)

	t.Cleanup(func() {
		db.Exec(`DELETE FROM products WHERE seller_id = $1`, seller)
		db.Exec(`DELETE FROM licence_group_rules WHERE group_id = $1`, group)
		db.Exec(`DELETE FROM licence_groups WHERE id = $1`, group)
		db.Exec(`DELETE FROM categories WHERE id = $1`, category)
		db.Exec(`DELETE FROM sellers WHERE id = $1`, seller)
	})

	id, err := CreateProduct(ctx, db, NewProduct{
		SellerID:   seller,
		Name:       "Test product",
		CategoryID: category,
		// Matches no seeded code rule, so only the test group applies.
		IKPU:   "99999999999999999",
		Status: "active",
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	products, err := ListProducts(ctx, db)
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}

	var found *ProductView
	for i := range products {
		if products[i].ID == id {
			found = &products[i]
		}
	}
	if found == nil {
		t.Fatalf("created product %s not listed", id)
	}
	if !found.OnHold {
		t.Fatal("product in a licensed category should be on hold for a seller without a licence")
	}
	if len(found.HoldReasons) != 1 {
		t.Fatalf("hold reasons = %v, want one reason", found.HoldReasons)
	}

	// The buyer catalogue must not show a held product.
	catalogue, err := ListBuyerCatalogue(ctx, db)
	if err != nil {
		t.Fatalf("ListBuyerCatalogue: %v", err)
	}
	for _, p := range catalogue {
		if p.ID == id {
			t.Fatal("held product appeared in the buyer catalogue")
		}
	}

	// Unknown references are refused.
	_, err = CreateProduct(ctx, db, NewProduct{
		SellerID: uniqueID(t, "missing-seller"), Name: "x", CategoryID: category, IKPU: "22042100999999999",
	})
	if !errors.Is(err, ErrUnknownSeller) {
		t.Fatalf("err = %v, want ErrUnknownSeller", err)
	}

	_, err = CreateProduct(ctx, db, NewProduct{
		SellerID: seller, Name: "x", CategoryID: uniqueID(t, "missing-cat"), IKPU: "22042100999999999",
	})
	if !errors.Is(err, ErrUnknownCategory) {
		t.Fatalf("err = %v, want ErrUnknownCategory", err)
	}
}
