package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleRoleAssignments(w http.ResponseWriter, r *http.Request, segs []string) {
	isBulk := len(segs) > 0 && segs[0] == "bulk"

	switch r.Method {
	case http.MethodPost:
		if isBulk {
			s.handleBulkRoleAssignmentCreate(w, r)
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
		writeJSON(w, http.StatusOK, assignments)

	case http.MethodDelete:
		if isBulk {
			s.handleBulkRoleAssignmentDelete(w, r)
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

func (s *Server) handleBulkRoleAssignmentCreate(w http.ResponseWriter, r *http.Request) {
	var assignments []struct {
		User   string `json:"user"`
		Role   string `json:"role"`
		Tenant string `json:"tenant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&assignments); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	created := 0
	for _, a := range assignments {
		if _, err := s.store.CreateRoleAssignment(a.User, a.Role, a.Tenant); err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"assignments_created": created,
	})
}

func (s *Server) handleBulkRoleAssignmentDelete(w http.ResponseWriter, r *http.Request) {
	var unassignments []struct {
		User   string `json:"user"`
		Role   string `json:"role"`
		Tenant string `json:"tenant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&unassignments); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	removed := 0
	for _, u := range unassignments {
		if err := s.store.DeleteRoleAssignment(u.User, u.Role, u.Tenant); err == nil {
			removed++
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"assignments_removed": removed,
	})
}
