package review

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"licensed-goods/internal/catalogue"
)

type QueueItem struct {
	ID               string                  `json:"id"`
	SellerID         string                  `json:"seller_id"`
	SellerName       string                  `json:"seller_name"`
	GroupID          string                  `json:"group_id"`
	GroupName        string                  `json:"group_name"`
	Number           string                  `json:"number"`
	Issuer           string                  `json:"issuer"`
	ValidFrom        string                  `json:"valid_from"`
	ValidTo          string                  `json:"valid_to,omitempty"`
	CreatedAt        time.Time               `json:"created_at"`
	AffectedProducts []catalogue.ProductView `json:"affected_products"`
}

func ListPending(ctx context.Context, db *sql.DB) ([]QueueItem, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			l.id,
			l.seller_id,
			COALESCE(s.name, ''),
			l.group_id,
			COALESCE(g.name, ''),
			l.number,
			l.issuer,
			l.valid_from::text,
			COALESCE(l.valid_to::text, ''),
			l.created_at
		FROM licences l
		LEFT JOIN sellers s ON s.id = l.seller_id
		LEFT JOIN licence_groups g ON g.id = l.group_id
		WHERE l.status = 'pending'
		ORDER BY l.created_at ASC, l.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("load pending licences: %w", err)
	}

	queue := make([]QueueItem, 0)

	for rows.Next() {
		var item QueueItem

		if err := rows.Scan(
			&item.ID,
			&item.SellerID,
			&item.SellerName,
			&item.GroupID,
			&item.GroupName,
			&item.Number,
			&item.Issuer,
			&item.ValidFrom,
			&item.ValidTo,
			&item.CreatedAt,
		); err != nil {
			rows.Close()
			return nil, err
		}

		queue = append(queue, item)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	products, err := catalogue.ListProducts(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load affected products: %w", err)
	}

	for i := range queue {
		queue[i].AffectedProducts = make(
			[]catalogue.ProductView, 0,
		)

		for _, product := range products {
			if product.SellerID != queue[i].SellerID {
				continue
			}

			for _, group := range product.RequiredGroups {
				if group.ID == queue[i].GroupID {
					queue[i].AffectedProducts = append(
						queue[i].AffectedProducts,
						product,
					)
					break
				}
			}
		}
	}

	return queue, nil
}
