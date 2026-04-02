package server

import (
	"net/http"

	"github.com/46labs/permitio/pkg/store"
)

func (s *Server) handleAPIKeyScope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": store.MockOrgID,
		"project_id":      store.MockProjID,
		"environment_id":  store.MockEnvID,
		"access_level":    "environment",
	})
}
