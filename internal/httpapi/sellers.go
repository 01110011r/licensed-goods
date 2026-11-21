package httpapi

import (
	"encoding/json"
	"net/http"

	"licensed-goods/internal/catalogue"
)

func (s *Server) listSellerProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sellerID := r.PathValue("sellerID")

	// Verify that the seller exists.
	var exists bool
	err := s.db.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM sellers WHERE id = $1)`,
		sellerID,
	).Scan(&exists)

	if err != nil {
		http.Error(w, "failed to check seller", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "seller not found", http.StatusNotFound)
		return
	}

	products, err := catalogue.ListProducts(ctx, s.db)
	if err != nil {
		http.Error(w, "failed to list products", http.StatusInternalServerError)
		return
	}

	result := make([]catalogue.ProductView, 0)

	for _, product := range products {
		if product.SellerID == sellerID {
			result = append(result, product)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		return
	}
}
