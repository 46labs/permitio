package server

import (
	"encoding/json"
	"net/http"

	"github.com/46labs/permitio/pkg/store"
)

func (s *Server) handleImplicitGrants(w http.ResponseWriter, r *http.Request, resourceKey, roleKey string) {
	switch r.Method {
	case http.MethodPost:
		var body struct {
			Role             string `json:"role"`
			OnResource       string `json:"on_resource"`
			LinkedByRelation string `json:"linked_by_relation"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ig := store.ImplicitGrant{
			Role:             body.Role,
			OnResource:       body.OnResource,
			LinkedByRelation: body.LinkedByRelation,
		}
		result, err := s.store.CreateImplicitGrant(resourceKey, roleKey, ig)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		// Return DerivedRoleRuleRead format
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"role_id":            result.RoleID,
			"resource_id":        result.ResourceID,
			"relation_id":        result.RelationID,
			"role":               result.Role,
			"on_resource":        result.OnResource,
			"linked_by_relation": result.LinkedByRelation,
		})

	case http.MethodDelete:
		var body struct {
			Role             string `json:"role"`
			OnResource       string `json:"on_resource"`
			LinkedByRelation string `json:"linked_by_relation"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ig := store.ImplicitGrant{
			Role:             body.Role,
			OnResource:       body.OnResource,
			LinkedByRelation: body.LinkedByRelation,
		}
		if err := s.store.DeleteImplicitGrant(resourceKey, roleKey, ig); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
