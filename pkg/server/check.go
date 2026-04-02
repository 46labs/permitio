package server

import (
	"encoding/json"
	"net/http"
)

// checkRequest matches the enforcement.CheckRequest JSON shape.
type checkRequest struct {
	User     checkUser     `json:"user"`
	Action   string        `json:"action"`
	Resource checkResource `json:"resource"`
}

type checkUser struct {
	Key string `json:"key"`
}

type checkResource struct {
	Type   string `json:"type"`
	Key    string `json:"key,omitempty"`
	ID     string `json:"id,omitempty"`
	Tenant string `json:"tenant,omitempty"`
}

func (cr *checkResource) instanceKey() string {
	if cr.Key != "" {
		return cr.Key
	}
	return cr.ID
}

// POST /allowed
func (s *Server) handleCheckImpl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req checkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	allowed := s.store.CheckPermission(
		req.User.Key,
		req.Action,
		req.Resource.Type,
		req.Resource.instanceKey(),
		req.Resource.Tenant,
	)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"allow":  allowed,
		"result": allowed,
	})
}

// POST /allowed/bulk
func (s *Server) handleBulkCheckImpl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var requests []checkRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	results := make([]map[string]interface{}, len(requests))
	for i, req := range requests {
		allowed := s.store.CheckPermission(
			req.User.Key,
			req.Action,
			req.Resource.Type,
			req.Resource.instanceKey(),
			req.Resource.Tenant,
		)
		results[i] = map[string]interface{}{
			"allow":  allowed,
			"result": allowed,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"allow": results,
	})
}

// POST /allowed/all-tenants
func (s *Server) handleAllTenantsCheckImpl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req checkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	results := s.store.GetAllowedTenants(
		req.User.Key,
		req.Action,
		req.Resource.Type,
		req.Resource.instanceKey(),
	)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"allowed_tenants": results,
	})
}

// POST /user-permissions
func (s *Server) handleUserPermissionsImpl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		User    checkUser `json:"user"`
		Tenants []string  `json:"tenants,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	perms := s.store.GetUserPermissions(req.User.Key, req.Tenants)

	// The SDK expects the response to directly be a map of tenant keys to permission sets.
	writeJSON(w, http.StatusOK, perms)
}
