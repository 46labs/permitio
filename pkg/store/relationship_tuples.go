package store

import "fmt"

func (s *Store) CreateRelationshipTuple(subject, relation, object string, tenant *string) (*RelationshipTuple, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rt := RelationshipTuple{
		BaseFields: newBase(),
		Subject:    subject,
		Relation:   relation,
		Object:     object,
	}
	if tenant != nil {
		rt.Tenant = *tenant
		if t, ok := s.tenants[*tenant]; ok {
			rt.TenantID = t.ID
		}
	}
	s.relationshipTuples = append(s.relationshipTuples, rt)
	return &rt, nil
}

func (s *Store) ListRelationshipTuples() []RelationshipTuple {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]RelationshipTuple, len(s.relationshipTuples))
	copy(result, s.relationshipTuples)
	return result
}

func (s *Store) DeleteRelationshipTuple(subject, relation, object string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, rt := range s.relationshipTuples {
		if rt.Subject == subject && rt.Relation == relation && rt.Object == object {
			s.relationshipTuples = append(s.relationshipTuples[:i], s.relationshipTuples[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("relationship tuple not found")
}
