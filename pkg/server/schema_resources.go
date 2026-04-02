package server

import (
	"encoding/json"
	"net/http"

	"github.com/46labs/permitio/pkg/store"
)

func (s *Server) handleResources(w http.ResponseWriter, r *http.Request, segs []string) {
	// segs: [] = list/create
	// [key] = get/update/delete
	// [key, "roles"] = list resource roles
	// [key, "roles", roleKey] = get/update/delete resource role
	// [key, "roles", roleKey, "permissions"] = assign/remove permissions
	// [key, "roles", roleKey, "implicit_grants"] = implicit grants
	// [key, "relations"] = list/create relations
	// [key, "relations", relKey] = get/delete relation

	if len(segs) >= 2 {
		resourceKey := segs[0]
		subResource := segs[1]
		switch subResource {
		case "roles":
			s.handleResourceRoles(w, r, resourceKey, segs[2:])
			return
		case "relations":
			s.handleRelations(w, r, resourceKey, segs[2:])
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
			Key         string                         `json:"key"`
			Name        string                         `json:"name"`
			Description *string                        `json:"description,omitempty"`
			Actions     map[string]actionBlockEditable `json:"actions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		actions := make(map[string]store.ActionBlockInput)
		for k, v := range body.Actions {
			a := store.ActionBlockInput{}
			if v.Name != nil {
				a.Name = *v.Name
			}
			a.Description = v.Description
			actions[k] = a
		}
		res, err := s.store.CreateResource(body.Key, body.Name, actions)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resourceToRead(res))

	case http.MethodGet:
		if len(segs) == 0 {
			resources := s.store.ListResources()
			result := make([]interface{}, len(resources))
			for i, r := range resources {
				result[i] = resourceToRead(r)
			}
			writeJSON(w, http.StatusOK, result)
		} else {
			res, err := s.store.GetResource(segs[0])
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, resourceToRead(res))
		}

	case http.MethodPatch:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Name    *string                         `json:"name,omitempty"`
			Actions *map[string]actionBlockEditable `json:"actions,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		var actions map[string]store.ActionBlockInput
		if body.Actions != nil {
			actions = make(map[string]store.ActionBlockInput)
			for k, v := range *body.Actions {
				a := store.ActionBlockInput{}
				if v.Name != nil {
					a.Name = *v.Name
				}
				a.Description = v.Description
				actions[k] = a
			}
		}
		res, err := s.store.UpdateResource(segs[0], body.Name, actions)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resourceToRead(res))

	case http.MethodDelete:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if err := s.store.DeleteResource(segs[0]); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type actionBlockEditable struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// resourceToRead converts our store Resource to the format the SDK expects (ActionBlockRead with id and key)
func resourceToRead(res *store.Resource) map[string]interface{} {
	actions := make(map[string]interface{})
	for k, v := range res.Actions {
		actions[k] = map[string]interface{}{
			"name":        v.Name,
			"description": v.Description,
			"id":          v.ID,
			"key":         k,
		}
	}
	result := map[string]interface{}{
		"key":             res.Key,
		"id":              res.ID,
		"organization_id": res.OrganizationID,
		"project_id":      res.ProjectID,
		"environment_id":  res.EnvironmentID,
		"created_at":      res.CreatedAt,
		"updated_at":      res.UpdatedAt,
		"name":            res.Name,
		"actions":         actions,
	}
	if res.Urn != nil {
		result["urn"] = res.Urn
	}
	if res.Description != nil {
		result["description"] = res.Description
	}
	return result
}

func (s *Server) handleResourceRoles(w http.ResponseWriter, r *http.Request, resourceKey string, segs []string) {
	// segs: [] = list/create, [roleKey] = get/update/delete
	// [roleKey, "permissions"] = assign/remove permissions
	// [roleKey, "implicit_grants"] = implicit grants
	if len(segs) >= 2 {
		switch segs[1] {
		case "permissions":
			s.handleResourceRolePermissions(w, r, resourceKey, segs[0])
			return
		case "implicit_grants":
			s.handleImplicitGrants(w, r, resourceKey, segs[0])
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
			Extends     []string `json:"extends,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		rr, err := s.store.CreateResourceRole(resourceKey, body.Key, body.Name, body.Permissions)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rr)

	case http.MethodGet:
		if len(segs) == 0 {
			roles := s.store.ListResourceRoles(resourceKey)
			writeJSON(w, http.StatusOK, roles)
		} else {
			rr, err := s.store.GetResourceRole(resourceKey, segs[0])
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, rr)
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
		rr, err := s.store.UpdateResourceRole(resourceKey, segs[0], body.Name, body.Description, body.Permissions)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rr)

	case http.MethodDelete:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if err := s.store.DeleteResourceRole(resourceKey, segs[0]); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleResourceRolePermissions(w http.ResponseWriter, r *http.Request, resourceKey, roleKey string) {
	switch r.Method {
	case http.MethodPost:
		var body struct {
			Permissions []string `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		rr, err := s.store.AssignPermissionsToResourceRole(resourceKey, roleKey, body.Permissions)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rr)

	case http.MethodDelete:
		var body struct {
			Permissions []string `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		rr, err := s.store.RemovePermissionsFromResourceRole(resourceKey, roleKey, body.Permissions)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rr)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleRelations(w http.ResponseWriter, r *http.Request, resourceKey string, segs []string) {
	switch r.Method {
	case http.MethodPost:
		if len(segs) > 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Key             string  `json:"key"`
			Name            string  `json:"name"`
			Description     *string `json:"description,omitempty"`
			SubjectResource string  `json:"subject_resource"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		rel, err := s.store.CreateRelation(resourceKey, body.Key, body.Name, body.SubjectResource)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rel)

	case http.MethodGet:
		if len(segs) == 0 {
			// List - returns PaginatedResultRelationRead
			rels := s.store.ListRelations(resourceKey)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"data":        rels,
				"total_count": len(rels),
				"page_count":  1,
			})
		} else {
			rel, err := s.store.GetRelation(resourceKey, segs[0])
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, rel)
		}

	case http.MethodDelete:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if err := s.store.DeleteRelation(resourceKey, segs[0]); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
