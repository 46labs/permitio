package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleRoles(w http.ResponseWriter, r *http.Request, segs []string) {
	// segs: [] = list/create, [key] = get/update/delete
	// [key, "permissions"] = assign/remove permissions
	if len(segs) >= 2 {
		switch segs[1] {
		case "permissions":
			s.handleRolePermissions(w, r, segs[0])
			return
		case "parents":
			if len(segs) < 3 {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			s.handleRoleParents(w, r, segs[0], segs[2])
			return
		}
	}

	switch r.Method {
	case http.MethodPost:
		if len(segs) > 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Key         string   `json:"key"`
			Name        string   `json:"name"`
			Description *string  `json:"description,omitempty"`
			Permissions []string `json:"permissions,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		role, err := s.store.CreateRole(body.Key, body.Name, body.Permissions)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, role)

	case http.MethodGet:
		if len(segs) == 0 {
			roles := s.store.ListRoles()
			writeJSON(w, http.StatusOK, roles)
		} else {
			role, err := s.store.GetRole(segs[0])
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, role)
		}

	case http.MethodPatch:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Name        *string  `json:"name,omitempty"`
			Description *string  `json:"description,omitempty"`
			Permissions []string `json:"permissions,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		role, err := s.store.UpdateRole(segs[0], body.Name, body.Description, body.Permissions)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, role)

	case http.MethodDelete:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if err := s.store.DeleteRole(segs[0]); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleRolePermissions(w http.ResponseWriter, r *http.Request, roleKey string) {
	switch r.Method {
	case http.MethodPost:
		var body struct {
			Permissions []string `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		role, err := s.store.AssignPermissionsToRole(roleKey, body.Permissions)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, role)

	case http.MethodDelete:
		var body struct {
			Permissions []string `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		role, err := s.store.RemovePermissionsFromRole(roleKey, body.Permissions)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, role)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleRoleParents(w http.ResponseWriter, r *http.Request, roleKey, parentKey string) {
	switch r.Method {
	case http.MethodPut:
		role, err := s.store.AddParentRole(roleKey, parentKey)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, role)

	case http.MethodDelete:
		_, err := s.store.RemoveParentRole(roleKey, parentKey)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
