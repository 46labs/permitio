package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/46labs/permitio/pkg/store"
)

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request, segs []string) {
	// segs: [] = list/create, [key] = get/update/delete, [key, "roles"] = assign/unassign
	if len(segs) >= 2 && segs[1] == "roles" {
		s.handleUserRoles(w, r, segs[0])
		return
	}

	switch r.Method {
	case http.MethodPost:
		if len(segs) > 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Key        string                 `json:"key"`
			Email      *string                `json:"email,omitempty"`
			FirstName  *string                `json:"first_name,omitempty"`
			LastName   *string                `json:"last_name,omitempty"`
			Attributes map[string]interface{} `json:"attributes,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		u, err := s.store.CreateUser(body.Key, body.Email, body.FirstName, body.LastName, body.Attributes)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, u)

	case http.MethodGet:
		if len(segs) == 0 {
			// List - return paginated format
			users := s.store.ListUsers()
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"data":        users,
				"total_count": len(users),
				"page_count":  1,
			})
		} else {
			u, err := s.store.GetUser(segs[0])
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, u)
		}

	case http.MethodPatch:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Email      *string                `json:"email,omitempty"`
			FirstName  *string                `json:"first_name,omitempty"`
			LastName   *string                `json:"last_name,omitempty"`
			Attributes map[string]interface{} `json:"attributes,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		u, err := s.store.UpdateUser(segs[0], body.Email, body.FirstName, body.LastName, body.Attributes)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, u)

	case http.MethodPut:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Key        string                 `json:"key"`
			Email      *string                `json:"email,omitempty"`
			FirstName  *string                `json:"first_name,omitempty"`
			LastName   *string                `json:"last_name,omitempty"`
			Attributes map[string]interface{} `json:"attributes,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		u, err := s.store.UpsertUser(segs[0], body.Email, body.FirstName, body.LastName, body.Attributes)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, u)

	case http.MethodDelete:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if err := s.store.DeleteUser(segs[0]); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleUserRoles(w http.ResponseWriter, r *http.Request, userKey string) {
	switch r.Method {
	case http.MethodPost:
		var body struct {
			Role             string  `json:"role"`
			Tenant           string  `json:"tenant"`
			ResourceInstance *string `json:"resource_instance,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ra, err := s.store.CreateRoleAssignmentWithInstance(userKey, body.Role, body.Tenant, body.ResourceInstance)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		// Return RoleAssignmentRead format
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":              ra.ID,
			"user":            ra.User,
			"role":            ra.Role,
			"tenant":          ra.Tenant,
			"user_id":         ra.UserID,
			"role_id":         ra.RoleID,
			"tenant_id":       ra.TenantID,
			"organization_id": store.MockOrgID,
			"project_id":      store.MockProjID,
			"environment_id":  store.MockEnvID,
			"created_at":      ra.CreatedAt.Format(time.RFC3339Nano),
		})

	case http.MethodDelete:
		var body struct {
			Role             string  `json:"role"`
			Tenant           string  `json:"tenant"`
			ResourceInstance *string `json:"resource_instance,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.store.DeleteRoleAssignment(userKey, body.Role, body.Tenant); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		// UnassignRole returns UserRead
		u, err := s.store.GetUser(userKey)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, u)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
