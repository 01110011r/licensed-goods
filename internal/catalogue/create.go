package catalogue

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidProduct  = errors.New("invalid product")
	ErrUnknownSeller   = errors.New("seller does not exist")
	ErrUnknownCategory = errors.New("category does not exist")
)

const maxProductNameLength = 200

// NewProduct is the input for creating a product. Status defaults to draft.
type NewProduct struct {
	SellerID   string `json:"seller_id"`
	Name       string `json:"name"`
	CategoryID string `json:"category_id"`
	IKPU       string `json:"ikpu"`
	Status     string `json:"status"`
}

// Category is a node in the product category tree.
type Category struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"`
}

func validateNewProduct(input NewProduct) (NewProduct, error) {
	input.SellerID = strings.TrimSpace(input.SellerID)
	input.Name = strings.TrimSpace(input.Name)
	input.CategoryID = strings.TrimSpace(input.CategoryID)
	input.IKPU = strings.TrimSpace(input.IKPU)
	input.Status = strings.TrimSpace(input.Status)

	if input.SellerID == "" {
		return input, fmt.Errorf("%w: seller_id is required", ErrInvalidProduct)
	}
	if input.Name == "" {
		return input, fmt.Errorf("%w: name is required", ErrInvalidProduct)
	}
	if utf8.RuneCountInString(input.Name) > maxProductNameLength {
		return input, fmt.Errorf("%w: name must be at most %d characters", ErrInvalidProduct, maxProductNameLength)
	}
	if input.CategoryID == "" {
		return input, fmt.Errorf("%w: category_id is required", ErrInvalidProduct)
	}
	if !isSeventeenDigits(input.IKPU) {
		return input, fmt.Errorf("%w: ikpu must be exactly 17 digits", ErrInvalidProduct)
	}

	if input.Status == "" {
		input.Status = "draft"
	}
	switch input.Status {
	case "draft", "active", "archived":
	default:
		return input, fmt.Errorf("%w: status must be draft, active or archived", ErrInvalidProduct)
	}

	return input, nil
}

func isSeventeenDigits(value string) bool {
	if len(value) != 17 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// CreateProduct validates the input, checks that the seller and category
// exist, and stores the product. Hold status is not stored: it is calculated
// from licences on every read.
func CreateProduct(ctx context.Context, db *sql.DB, input NewProduct) (string, error) {
	input, err := validateNewProduct(input)
	if err != nil {
		return "", err
	}

	var exists bool

	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM sellers WHERE id = $1)`, input.SellerID,
	).Scan(&exists); err != nil {
		return "", fmt.Errorf("check seller: %w", err)
	}
	if !exists {
		return "", ErrUnknownSeller
	}

	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM categories WHERE id = $1)`, input.CategoryID,
	).Scan(&exists); err != nil {
		return "", fmt.Errorf("check category: %w", err)
	}
	if !exists {
		return "", ErrUnknownCategory
	}

	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate product ID: %w", err)
	}
	id := "prod-" + hex.EncodeToString(random)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO products (id, seller_id, name, category_id, ikpu, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, input.SellerID, input.Name, input.CategoryID, input.IKPU, input.Status); err != nil {
		return "", fmt.Errorf("insert product: %w", err)
	}

	return id, nil
}

// ListCategories returns every category, ordered by ID.
func ListCategories(ctx context.Context, db *sql.DB) ([]Category, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, parent_id FROM categories ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("load categories: %w", err)
	}
	defer rows.Close()

	categories := make([]Category, 0)
	for rows.Next() {
		var c Category
		var parent sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &parent); err != nil {
			return nil, err
		}
		if parent.Valid {
			p := parent.String
			c.ParentID = &p
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}
