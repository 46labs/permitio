package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleRoleAssignments(w http.ResponseWriter, r *http.Request, segs []string) {
	// segs could be [] for base or ["bulk"] for bulk operations
	isBulk := len(segs) > 0 && segs[0] == "bulk"

	switch r.Method {
	case http.MethodPost:
		if isBulk {
			// Bulk assign not needed for basic tests
			w.WriteHeader(http.StatusOK)
			return
		}
		var body struct {
			User   string `json:"user"`
			Role   string `json:"role"`
			Tenant string `json:"tenant"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ra, err := s.store.CreateRoleAssignment(body.User, body.Role, body.Tenant)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, ra)

	case http.MethodGet:
		if isBulk {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		assignments := s.store.ListRoleAssignments()
		// The SDK expects a JSON array of RoleAssignmentRead
		writeJSON(w, http.StatusOK, assignments)

	case http.MethodDelete:
		if isBulk {
			// Bulk unassign
			w.WriteHeader(http.StatusOK)
			return
		}
		var body struct {
			User   string `json:"user"`
			Role   string `json:"role"`
			Tenant string `json:"tenant"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.store.DeleteRoleAssignment(body.User, body.Role, body.Tenant); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
