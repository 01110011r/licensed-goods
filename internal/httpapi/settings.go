package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"licensed-goods/internal/settings"
)

func (s *Server) getToday(w http.ResponseWriter, r *http.Request) {
	today, err := settings.GetToday(r.Context(), s.db)
	if err != nil {
		http.Error(w, "Failed to load Today setting", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"today": today,
	})
}

func (s *Server) updateToday(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Today string `json:"today"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	err := settings.SetToday(r.Context(), s.db, input.Today)
	if err != nil {
		if errors.Is(err, settings.ErrInvalidDate) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Error(w, "Failed to update Today setting", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"today": input.Today,
	})
}
