package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleResourceActions(w http.ResponseWriter, r *http.Request, resourceKey string, segs []string) {
	// segs: [] = list/create, [actionKey] = get/update/delete
	switch r.Method {
	case http.MethodPost:
		if len(segs) > 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Key         string  `json:"key"`
			Name        string  `json:"name"`
			Description *string `json:"description,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		action, err := s.store.CreateResourceAction(resourceKey, body.Key, body.Name, body.Description)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, action)

	case http.MethodGet:
		if len(segs) == 0 {
			actions, err := s.store.ListResourceActions(resourceKey)
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, actions)
		} else {
			action, err := s.store.GetResourceAction(resourceKey, segs[0])
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, action)
		}

	case http.MethodPatch:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Name        *string `json:"name,omitempty"`
			Description *string `json:"description,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		action, err := s.store.UpdateResourceAction(resourceKey, segs[0], body.Name, body.Description)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, action)

	case http.MethodDelete:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if err := s.store.DeleteResourceAction(resourceKey, segs[0]); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
