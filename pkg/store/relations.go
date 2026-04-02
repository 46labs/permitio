package store

import "fmt"

func (s *Store) CreateRelation(resourceKey, key, name, subjectResource string) (*Relation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, ok := s.resources[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceKey)
	}
	if s.relations[resourceKey] == nil {
		s.relations[resourceKey] = make(map[string]*Relation)
	}
	if _, exists := s.relations[resourceKey][key]; exists {
		return nil, fmt.Errorf("relation %q already exists on resource %q", key, resourceKey)
	}
	// Look up subject resource ID
	subjectResourceID := ""
	if sr, ok := s.resources[subjectResource]; ok {
		subjectResourceID = sr.ID
	}
	rel := &Relation{
		BaseFields:        newBase(),
		Key:               key,
		Name:              name,
		SubjectResource:   subjectResource,
		SubjectResourceID: subjectResourceID,
		ObjectResourceID:  res.ID,
		ObjectResource:    resourceKey,
	}
	s.relations[resourceKey][key] = rel
	return rel, nil
}

func (s *Store) GetRelation(resourceKey, key string) (*Relation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rels, ok := s.relations[resourceKey]
	if !ok {
		return nil, fmt.Errorf("relation %q not found on resource %q", key, resourceKey)
	}
	rel, ok := rels[key]
	if !ok {
		return nil, fmt.Errorf("relation %q not found on resource %q", key, resourceKey)
	}
	return rel, nil
}

func (s *Store) ListRelations(resourceKey string) []*Relation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rels := s.relations[resourceKey]
	result := make([]*Relation, 0, len(rels))
	for _, rel := range rels {
		result = append(result, rel)
	}
	return result
}

func (s *Store) DeleteRelation(resourceKey, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rels, ok := s.relations[resourceKey]
	if !ok {
		return fmt.Errorf("relation %q not found on resource %q", key, resourceKey)
	}
	if _, ok := rels[key]; !ok {
		return fmt.Errorf("relation %q not found on resource %q", key, resourceKey)
	}
	delete(rels, key)
	return nil
}
