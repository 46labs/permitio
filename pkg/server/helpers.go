package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]interface{}{
		"error":   http.StatusText(status),
		"message": msg,
		"status":  status,
	})
}

// extractPathSegments strips a prefix and splits the remaining path.
// e.g. "/v2/schema/proj/env/resources/doc" with prefix "/v2/schema" returns ["proj", "env", "resources", "doc"]
func extractPathSegments(path, prefix string) []string {
	trimmed := strings.TrimPrefix(path, prefix)
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

// Permit's list defaults: page 1, 30 per page, at most 100.
const (
	defaultPerPage = 30
	maxPerPage     = 100
)

// paginate returns one page of items, as Permit pages its lists (page and
// per_page, 1-based), and the number of pages.
func paginate[T any](r *http.Request, items []T) ([]T, int) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	perPage = min(perPage, maxPerPage)
	pages := (len(items) + perPage - 1) / perPage
	start := (page - 1) * perPage
	if start >= len(items) {
		return []T{}, pages
	}
	return items[start:min(start+perPage, len(items))], pages
}
