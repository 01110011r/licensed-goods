package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"licensed-goods/internal/licensing"
)

const maxUploadSize = 10 << 20 // 10 MB

func (s *Server) submitLicence(w http.ResponseWriter, r *http.Request) {
	var input licensing.LicenceSubmission
	var savedFile string

	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+(1<<20))

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			http.Error(w, "Invalid multipart request or request too large", http.StatusBadRequest)
			return
		}

		input.SellerID = r.FormValue("seller_id")
		input.GroupID = r.FormValue("group_id")
		input.Number = r.FormValue("number")
		input.Issuer = r.FormValue("issuer")
		input.ValidFrom = r.FormValue("valid_from")
		input.ValidTo = r.FormValue("valid_to")

		file, header, err := r.FormFile("file")
		if err == nil {
			defer file.Close()

			savedFile, err = saveLicenceFile(file, header, s.uploadDir)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			input.FilePath = savedFile
		} else if !errors.Is(err, http.ErrMissingFile) {
			http.Error(w, "Invalid uploaded file", http.StatusBadRequest)
			return
		}
	} else if strings.HasPrefix(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Content-Type must be application/json or multipart/form-data", http.StatusUnsupportedMediaType)
		return
	}

	id, err := licensing.SubmitLicence(r.Context(), s.db, input)
	if err != nil {
		if savedFile != "" {
			os.Remove(savedFile)
		}

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

func saveLicenceFile(file multipart.File, header *multipart.FileHeader, uploadDir string) (string, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))

	allowed := map[string]bool{
		".pdf":  true,
		".png":  true,
		".jpg":  true,
		".jpeg": true,
	}

	if !allowed[ext] {
		return "", fmt.Errorf("unsupported file type")
	}

	buffer := make([]byte, 512)

	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read uploaded file: %w", err)
	}

	contentType := http.DetectContentType(buffer[:n])

	expectedTypes := map[string]string{
		".pdf":  "application/pdf",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
	}

	if contentType != expectedTypes[ext] {
		return "", fmt.Errorf("file content does not match its extension")
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("reset uploaded file: %w", err)
	}

	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return "", fmt.Errorf("generate filename: %w", err)
	}

	if err := os.MkdirAll(uploadDir, 0750); err != nil {
		return "", fmt.Errorf("create upload directory: %w", err)
	}

	path := filepath.Join(uploadDir, hex.EncodeToString(id)+ext)

	dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
	if err != nil {
		return "", fmt.Errorf("create upload: %w", err)
	}

	defer dst.Close()

	written, err := io.Copy(dst, io.LimitReader(file, maxUploadSize+1))
	if err != nil {
		os.Remove(path)
		return "", fmt.Errorf("save upload: %w", err)
	}

	if written > maxUploadSize {
		os.Remove(path)
		return "", fmt.Errorf("file exceeds 10 MB limit")
	}

	return path, nil
}
