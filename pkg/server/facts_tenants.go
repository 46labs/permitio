package server

import (
	"encoding/json"
	"net/http"

	"github.com/46labs/permitio/pkg/store"
)

func (s *Server) handleTenants(w http.ResponseWriter, r *http.Request, segs []string) {
	// Route: tenants/{key}/users
	if len(segs) >= 2 && segs[1] == "users" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		users := s.store.ListTenantUsers(segs[0])
		if users == nil {
			users = []*store.User{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data":        users,
			"total_count": len(users),
			"page_count":  1,
		})
		return
	}

	switch r.Method {
	case http.MethodPost:
		if len(segs) > 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Key         string                 `json:"key"`
			Name        string                 `json:"name"`
			Description *string                `json:"description,omitempty"`
			Attributes  map[string]interface{} `json:"attributes,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		t, err := s.store.CreateTenant(body.Key, body.Name)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if body.Description != nil {
			t.Description = body.Description
		}
		if body.Attributes != nil {
			t.Attributes = body.Attributes
		}
		writeJSON(w, http.StatusCreated, t)

	case http.MethodGet:
		if len(segs) == 0 {
			// List
			tenants := s.store.ListTenants()
			writeJSON(w, http.StatusOK, tenants)
		} else {
			// Get by key
			t, err := s.store.GetTenant(segs[0])
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, t)
		}

	case http.MethodPatch:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Name        *string                `json:"name,omitempty"`
			Description *string                `json:"description,omitempty"`
			Attributes  map[string]interface{} `json:"attributes,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		t, err := s.store.UpdateTenant(segs[0], body.Name, body.Description, body.Attributes)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, t)

	case http.MethodDelete:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if err := s.store.DeleteTenant(segs[0]); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
