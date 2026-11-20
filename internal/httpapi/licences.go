package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"licensed-goods/internal/licensing"
)

func (s *Server) submitLicence(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SellerID  string `json:"seller_id"`
		GroupID   string `json:"group_id"`
		Number    string `json:"number"`
		Issuer    string `json:"issuer"`
		ValidFrom string `json:"valid_from"`
		ValidTo   string `json:"valid_to"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	id, err := licensing.SubmitLicence(
		r.Context(),
		s.db,
		licensing.LicenceSubmission{
			SellerID:  input.SellerID,
			GroupID:   input.GroupID,
			Number:    input.Number,
			Issuer:    input.Issuer,
			ValidFrom: input.ValidFrom,
			ValidTo:   input.ValidTo,
		},
	)

	if err != nil {
		switch {
		case errors.Is(err, licensing.ErrInvalidSubmission),
			errors.Is(err, licensing.ErrUnknownSeller),
			errors.Is(err, licensing.ErrUnknownGroup):
			http.Error(w, err.Error(), http.StatusBadRequest)

		case errors.Is(err, licensing.ErrPendingExists):
			http.Error(w, err.Error(), http.StatusConflict)

		default:
			http.Error(w, "Failed to submit licence", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]string{
		"id":     id,
		"status": "pending",
	})
}
