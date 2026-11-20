package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"licensed-goods/internal/catalogue"
	"licensed-goods/internal/review"
)

type Server struct {
	db *sql.DB
}

func New(db *sql.DB) *Server {
	return &Server{db: db}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.health)

	// api registration
	mux.HandleFunc("GET /api/products", s.products)
	mux.HandleFunc("GET /api/catalogue", s.buyerCatalogue)
	mux.HandleFunc("GET /api/reviews", s.reviews)
	mux.HandleFunc("PATCH /api/reviews/{id}", s.decideReview)
	mux.HandleFunc("POST /api/licences", s.submitLicence)
	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.db.PingContext(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "unhealthy",
			"database": "disconnected",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"database": "connected",
	})
}

func (s *Server) products(w http.ResponseWriter, r *http.Request) {
	products, err := catalogue.ListProducts(r.Context(), s.db)
	if err != nil {
		http.Error(w, "Failed to load products", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func (s *Server) buyerCatalogue(w http.ResponseWriter, r *http.Request) {
	products, err := catalogue.ListBuyerCatalogue(r.Context(), s.db)
	if err != nil {
		http.Error(w, "Failed to load catalogue", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(products); err != nil {
		return
	}
}

func (s *Server) reviews(w http.ResponseWriter, r *http.Request) {
	items, err := review.ListPending(r.Context(), s.db)
	if err != nil {
		http.Error(w, "Failed to load review queue", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(items); err != nil {
		return
	}
}

func (s *Server) decideReview(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status          string `json:"status"`
		RejectionReason string `json:"rejection_reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	err := review.Decide(
		r.Context(),
		s.db,
		r.PathValue("id"),
		input.Status,
		input.RejectionReason,
	)

	if err != nil {
		switch {
		case errors.Is(err, review.ErrNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, review.ErrNotPending):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, review.ErrInvalidDecision),
			errors.Is(err, review.ErrReasonRequired),
			errors.Is(err, review.ErrPastExpiry),
			errors.Is(err, review.ErrInvalidDates),
			errors.Is(err, review.ErrUnknownReference):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "Review operation failed", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"id":%q,"status":%q}`, r.PathValue("id"), input.Status)
}
