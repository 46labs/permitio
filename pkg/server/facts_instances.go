package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

func (s *Server) handleResourceInstances(w http.ResponseWriter, r *http.Request, segs []string) {
	switch r.Method {
	case http.MethodPost:
		if len(segs) > 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Key        string                 `json:"key"`
			Resource   string                 `json:"resource"`
			Tenant     *string                `json:"tenant,omitempty"`
			Attributes map[string]interface{} `json:"attributes,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ri, err := s.store.CreateResourceInstance(body.Key, body.Resource, body.Tenant, body.Attributes)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, ri)

	case http.MethodGet:
		if len(segs) == 0 {
			// List
			instances := s.store.ListResourceInstances()
			writeJSON(w, http.StatusOK, instances)
		} else {
			// Get by "type:key" - may be URL-encoded
			instKey := decodeInstanceKey(segs)
			ri, err := s.store.GetResourceInstance(instKey)
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, ri)
		}

	case http.MethodPatch:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		instKey := decodeInstanceKey(segs)
		var body struct {
			Attributes map[string]interface{} `json:"attributes,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ri, err := s.store.UpdateResourceInstance(instKey, body.Attributes)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, ri)

	case http.MethodDelete:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		instKey := decodeInstanceKey(segs)
		if err := s.store.DeleteResourceInstance(instKey); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// decodeInstanceKey handles the "type:key" or URL-encoded "type%3Akey" segments.
// The SDK may URL-encode the colon. The segments are already split by "/" so
// "folder:budget" could come as a single segment or "folder%3Abudget".
func decodeInstanceKey(segs []string) string {
	// Join all remaining segments back together
	raw := strings.Join(segs, "/")
	// URL-decode
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return decoded
}
