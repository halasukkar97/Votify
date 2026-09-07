package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const maxUploadBytes = 5 * 1024 * 1024

type uploadImageResponse struct {
	ImageURL string `json:"imageUrl"`
}

// UploadCoverHandler stores a local image and returns a persistent URL for option.imageUrl.
func (server *Server) UploadCoverHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "invalid upload request", http.StatusBadRequest)
		return
	}

	part, err := reader.NextPart()
	if err != nil {
		http.Error(w, "missing image file", http.StatusBadRequest)
		return
	}
	defer part.Close()

	if part.FormName() != "image" {
		http.Error(w, "image field is required", http.StatusBadRequest)
		return
	}

	fileBytes, err := io.ReadAll(io.LimitReader(part, maxUploadBytes+1))
	if err != nil {
		http.Error(w, "failed to read image", http.StatusBadRequest)
		return
	}

	if len(fileBytes) > maxUploadBytes {
		http.Error(w, "image must be 5MB or smaller", http.StatusBadRequest)
		return
	}

	contentType := http.DetectContentType(fileBytes)
	extension, ok := extensionForImageType(contentType)
	if !ok {
		http.Error(w, "image must be a jpg, png, or webp file", http.StatusBadRequest)
		return
	}

	uploadDir := server.UploadDir
	if uploadDir == "" {
		uploadDir = "uploads"
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "failed to prepare uploads", http.StatusInternalServerError)
		return
	}

	filename := uuid.New().String() + extension
	path := filepath.Join(uploadDir, filename)
	if err := os.WriteFile(path, fileBytes, 0644); err != nil {
		http.Error(w, "failed to save image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uploadImageResponse{ImageURL: publicUploadURL(r, filename)})
}

func extensionForImageType(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func publicUploadURL(r *http.Request, filename string) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
		if r.TLS != nil {
			proto = "https"
		}
	}

	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	return fmt.Sprintf("%s://%s/uploads/%s", strings.TrimSpace(proto), host, filename)
}
