package catalogue

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"licensed-goods/internal/licensing"
)

type RequiredGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProductView struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	SellerID       string            `json:"seller_id"`
	SellerName     string            `json:"seller_name"`
	CategoryID     string            `json:"category_id"`
	IKPU           string            `json:"ikpu"`
	Status         string            `json:"status"`
	RequiredGroups []RequiredGroup   `json:"required_groups"`
	OnHold         bool              `json:"on_hold"`
	HoldReasons    []string          `json:"hold_reasons"`
	DataIssues     []licensing.Issue `json:"data_issues"`
}

func loadMatchingData(
	ctx context.Context,
	db *sql.DB,
) (map[string]licensing.Category, []licensing.Group, error) {
	categories := make(map[string]licensing.Category)

	rows, err := db.QueryContext(ctx,
		`SELECT id, COALESCE(parent_id, '') FROM categories`,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("load categories: %w", err)
	}

	for rows.Next() {
		var c licensing.Category
		if err := rows.Scan(&c.ID, &c.ParentID); err != nil {
			rows.Close()
			return nil, nil, err
		}
		categories[c.ID] = c
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()

	groups := make([]licensing.Group, 0)
	groupIndex := make(map[string]int)

	rows, err = db.QueryContext(ctx,
		`SELECT id FROM licence_groups ORDER BY id`,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("load groups: %w", err)
	}

	for rows.Next() {
		var g licensing.Group
		if err := rows.Scan(&g.ID); err != nil {
			rows.Close()
			return nil, nil, err
		}

		groupIndex[g.ID] = len(groups)
		groups = append(groups, g)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()

	rows, err = db.QueryContext(ctx, `
        SELECT group_id, kind,
               COALESCE(category_id, ''),
               COALESCE(prefix, ''),
               excluded
        FROM licence_group_rules
        ORDER BY id
    `)
	if err != nil {
		return nil, nil, fmt.Errorf("load rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var groupID, kind, categoryID, prefix string
		var excluded bool

		if err := rows.Scan(
			&groupID, &kind, &categoryID, &prefix, &excluded,
		); err != nil {
			return nil, nil, err
		}

		index, exists := groupIndex[groupID]
		if !exists {
			continue
		}

		rule := licensing.Rule{
			CategoryID: categoryID,
			IKPUPrefix: prefix,
			Excluded:   excluded,
		}

		switch kind {
		case "category":
			rule.Kind = licensing.RuleCategory
		case "ikpu":
			rule.Kind = licensing.RuleIKPU
		default:
			continue
		}

		groups[index].Rules = append(groups[index].Rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return categories, groups, nil
}

func loadToday(ctx context.Context, db *sql.DB) (time.Time, error) {
	var value string

	err := db.QueryRowContext(ctx,
		`SELECT value FROM app_settings WHERE key = 'today'`,
	).Scan(&value)
	if err != nil {
		return time.Time{}, fmt.Errorf("load today setting: %w", err)
	}

	today, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid today setting %q: %w", value, err)
	}

	return today, nil
}

func loadValidLicences(
	ctx context.Context,
	db *sql.DB,
	today time.Time,
) (map[string]bool, error) {
	valid := make(map[string]bool)

	rows, err := db.QueryContext(ctx, `
        SELECT seller_id, group_id
        FROM licences
        WHERE status = 'approved'
          AND valid_from <= $1
          AND (valid_to IS NULL OR valid_to >= $1)
    `, today.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("load valid licences: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sellerID, groupID string
		if err := rows.Scan(&sellerID, &groupID); err != nil {
			return nil, err
		}

		valid[sellerID+"\x00"+groupID] = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return valid, nil
}

func applyLicenceRequirements(
	view *ProductView,
	groupIDs []string,
	groupNames map[string]string,
	validLicences map[string]bool,
) {
	view.RequiredGroups = make([]RequiredGroup, 0, len(groupIDs))
	view.HoldReasons = make([]string, 0)
	view.OnHold = false

	for _, groupID := range groupIDs {
		view.RequiredGroups = append(view.RequiredGroups, RequiredGroup{
			ID:   groupID,
			Name: groupNames[groupID],
		})

		key := view.SellerID + "\x00" + groupID
		if !validLicences[key] {
			view.OnHold = true
			view.HoldReasons = append(
				view.HoldReasons,
				"Missing valid approved licence for group "+groupID,
			)
		}
	}
}

func ListProducts(ctx context.Context, db *sql.DB) ([]ProductView, error) {
	categories, groups, err := loadMatchingData(ctx, db)
	if err != nil {
		return nil, err
	}

	today, err := loadToday(ctx, db)
	if err != nil {
		return nil, err
	}

	validLicences, err := loadValidLicences(ctx, db, today)
	if err != nil {
		return nil, err
	}

	groupNames := make(map[string]string)
	for _, group := range groups {
		var name string
		err := db.QueryRowContext(ctx,
			`SELECT name FROM licence_groups WHERE id = $1`,
			group.ID,
		).Scan(&name)
		if err != nil {
			return nil, err
		}
		groupNames[group.ID] = name
	}

	rows, err := db.QueryContext(ctx, `
        SELECT p.id, p.name, p.seller_id,
               COALESCE(s.name, ''),
               p.category_id, p.ikpu, p.status
        FROM products p
        LEFT JOIN sellers s ON s.id = p.seller_id
        ORDER BY p.id
    `)
	if err != nil {
		return nil, fmt.Errorf("load products: %w", err)
	}
	defer rows.Close()

	products := make([]ProductView, 0)

	for rows.Next() {
		var p licensing.Product
		var view ProductView

		if err := rows.Scan(
			&p.ID,
			&view.Name,
			&view.SellerID,
			&view.SellerName,
			&p.CategoryID,
			&p.IKPU,
			&view.Status,
		); err != nil {
			return nil, err
		}

		view.ID = p.ID
		view.CategoryID = p.CategoryID
		view.IKPU = p.IKPU

		result := licensing.Match(p, categories, groups)
		view.DataIssues = result.Issues

		applyLicenceRequirements(
			&view,
			result.GroupIDs,
			groupNames,
			validLicences,
		)

		products = append(products, view)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func filterBuyerCatalogue(products []ProductView) []ProductView {
	visible := make([]ProductView, 0)

	for _, product := range products {
		if product.Status == "active" && !product.OnHold {
			visible = append(visible, product)
		}
	}

	return visible
}

func ListBuyerCatalogue(ctx context.Context, db *sql.DB) ([]ProductView, error) {
	products, err := ListProducts(ctx, db)
	if err != nil {
		return nil, err
	}

	return filterBuyerCatalogue(products), nil
}
