package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
)

type seedData struct {
	Today      string         `json:"today"`
	Categories []seedCategory `json:"categories"`
	Sellers    []seedSeller   `json:"sellers"`
	Groups     []seedGroup    `json:"groups"`
	Products   []seedProduct  `json:"products"`
	Licences   []seedLicence  `json:"licences"`
}

type seedCategory struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"`
}

type seedSeller struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type seedGroup struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Rules []seedRule `json:"rules"`
}

type seedRule struct {
	Kind       string `json:"kind"`
	CategoryID string `json:"category_id"`
	Prefix     string `json:"prefix"`
	Exclude    bool   `json:"exclude"`
}

type seedProduct struct {
	ID         string `json:"id"`
	SellerID   string `json:"seller_id"`
	Name       string `json:"name"`
	CategoryID string `json:"category_id"`
	IKPU       string `json:"ikpu"`
	Status     string `json:"status"`
}

type seedLicence struct {
	ID        string  `json:"id"`
	SellerID  string  `json:"seller_id"`
	GroupID   string  `json:"group_id"`
	Number    string  `json:"number"`
	Issuer    string  `json:"issuer"`
	ValidFrom string  `json:"valid_from"`
	ValidTo   *string `json:"valid_to"`
	Status    string  `json:"status"`
}

func Seed(ctx context.Context, db *sql.DB, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read seed file: %w", err)
	}

	var data seedData
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("parse seed file: %w", err)
	}

	if data.Today == "" {
		return fmt.Errorf("seed today is required")
	}

	var initialized bool

	checkErr := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM categories)`,
	).Scan(&initialized)

	if checkErr != nil {
		return fmt.Errorf("check database initialization: %w", checkErr)
	}

	if initialized {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback()

	// Categories
	for _, c := range data.Categories {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO categories (id, name, parent_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				parent_id = EXCLUDED.parent_id
		`, c.ID, c.Name, c.ParentID)
		if err != nil {
			return fmt.Errorf("insert category %s: %w", c.ID, err)
		}
	}

	// Sellers
	for _, s := range data.Sellers {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO sellers (id, name)
			VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name
		`, s.ID, s.Name)
		if err != nil {
			return fmt.Errorf("insert seller %s: %w", s.ID, err)
		}
	}

	// Licence groups and their rules
	for _, g := range data.Groups {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO licence_groups (id, name)
			VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name
		`, g.ID, g.Name)
		if err != nil {
			return fmt.Errorf("insert group %s: %w", g.ID, err)
		}

		// Replace rules so repeated imports don't duplicate them.
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM licence_group_rules WHERE group_id = $1`, g.ID,
		); err != nil {
			return fmt.Errorf("delete rules for group %s: %w", g.ID, err)
		}

		for _, r := range g.Rules {
			var dbKind string
			var categoryID, prefix any

			switch r.Kind {
			case "category":
				dbKind = "category"
				categoryID = r.CategoryID
			case "code":
				dbKind = "ikpu"
				prefix = r.Prefix
			default:
				return fmt.Errorf("unknown rule kind %q in group %s", r.Kind, g.ID)
			}

			_, err := tx.ExecContext(ctx, `
				INSERT INTO licence_group_rules
					(group_id, kind, category_id, prefix, excluded)
				VALUES ($1, $2, $3, $4, $5)
			`, g.ID, dbKind, categoryID, prefix, r.Exclude)
			if err != nil {
				return fmt.Errorf("insert rule for group %s: %w", g.ID, err)
			}
		}
	}

	// Products
	for _, p := range data.Products {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO products
				(id, seller_id, name, category_id, ikpu, status)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				seller_id = EXCLUDED.seller_id,
				name = EXCLUDED.name,
				category_id = EXCLUDED.category_id,
				ikpu = EXCLUDED.ikpu,
				status = EXCLUDED.status
		`, p.ID, p.SellerID, p.Name, p.CategoryID, p.IKPU, p.Status)
		if err != nil {
			return fmt.Errorf("insert product %s: %w", p.ID, err)
		}
	}

	// Licences
	for _, l := range data.Licences {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO licences
				(id, seller_id, group_id, number, issuer,
				 valid_from, valid_to, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				seller_id = EXCLUDED.seller_id,
				group_id = EXCLUDED.group_id,
				number = EXCLUDED.number,
				issuer = EXCLUDED.issuer,
				valid_from = EXCLUDED.valid_from,
				valid_to = EXCLUDED.valid_to,
				status = EXCLUDED.status
		`, l.ID, l.SellerID, l.GroupID, l.Number, l.Issuer,
			l.ValidFrom, l.ValidTo, l.Status)
		if err != nil {
			return fmt.Errorf("insert licence %s: %w", l.ID, err)
		}
	}

	// Initial assessment date
	_, err = tx.ExecContext(ctx, `
		INSERT INTO app_settings (key, value)
		VALUES ('today', $1)
		ON CONFLICT (key) DO NOTHING
	`, data.Today)
	if err != nil {
		return fmt.Errorf("save assessment date: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}

	return nil
}
