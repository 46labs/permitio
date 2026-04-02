package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleRelationshipTuples(w http.ResponseWriter, r *http.Request, segs []string) {
	// segs could be [] for base or ["bulk"] for bulk operations
	isBulk := len(segs) > 0 && segs[0] == "bulk"

	switch r.Method {
	case http.MethodPost:
		if isBulk {
			s.handleRelationshipTuplesBulkCreate(w, r)
			return
		}
		var body struct {
			Subject  string  `json:"subject"`
			Relation string  `json:"relation"`
			Object   string  `json:"object"`
			Tenant   *string `json:"tenant,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		rt, err := s.store.CreateRelationshipTuple(body.Subject, body.Relation, body.Object, body.Tenant)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rt)

	case http.MethodGet:
		if isBulk {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		tuples := s.store.ListRelationshipTuples()
		writeJSON(w, http.StatusOK, tuples)

	case http.MethodDelete:
		if isBulk {
			s.handleRelationshipTuplesBulkDelete(w, r)
			return
		}
		// Delete with body
		var body struct {
			Subject  string `json:"subject"`
			Relation string `json:"relation"`
			Object   string `json:"object"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.store.DeleteRelationshipTuple(body.Subject, body.Relation, body.Object); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleRelationshipTuplesBulkCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Operations []struct {
			Subject  string  `json:"subject"`
			Relation string  `json:"relation"`
			Object   string  `json:"object"`
			Tenant   *string `json:"tenant,omitempty"`
		} `json:"operations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	for _, op := range body.Operations {
		s.store.CreateRelationshipTuple(op.Subject, op.Relation, op.Object, op.Tenant) //nolint:errcheck
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRelationshipTuplesBulkDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Idents []struct {
			Subject  string `json:"subject"`
			Relation string `json:"relation"`
			Object   string `json:"object"`
		} `json:"idents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	for _, ident := range body.Idents {
		s.store.DeleteRelationshipTuple(ident.Subject, ident.Relation, ident.Object) //nolint:errcheck
	}
	w.WriteHeader(http.StatusOK)
}
