package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"licensed-goods/internal/catalogue"
)

const maxProductBodySize = 64 << 10 // 64 KB

func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxProductBodySize)

	var input catalogue.NewProduct
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	id, err := catalogue.CreateProduct(r.Context(), s.db, input)
	if err != nil {
		switch {
		case errors.Is(err, catalogue.ErrInvalidProduct),
			errors.Is(err, catalogue.ErrUnknownSeller),
			errors.Is(err, catalogue.ErrUnknownCategory):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "Failed to create product", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func (s *Server) categories(w http.ResponseWriter, r *http.Request) {
	categories, err := catalogue.ListCategories(r.Context(), s.db)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}
