package store

func (s *Store) CreateImplicitGrant(resourceKey, roleKey string, ig ImplicitGrant) (*ImplicitGrant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Fill in IDs if possible
	if res, ok := s.resources[ig.OnResource]; ok {
		ig.ResourceID = res.ID
	}
	if roles, ok := s.resourceRoles[ig.OnResource]; ok {
		if role, ok := roles[ig.Role]; ok {
			ig.RoleID = role.ID
		}
	}
	if rels, ok := s.relations[resourceKey]; ok {
		if rel, ok := rels[ig.LinkedByRelation]; ok {
			ig.RelationID = rel.ID
		}
	}
	s.implicitGrants = append(s.implicitGrants, ig)
	return &ig, nil
}

func (s *Store) DeleteImplicitGrant(resourceKey, roleKey string, ig ImplicitGrant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.implicitGrants {
		if existing.Role == ig.Role && existing.OnResource == ig.OnResource && existing.LinkedByRelation == ig.LinkedByRelation {
			s.implicitGrants = append(s.implicitGrants[:i], s.implicitGrants[i+1:]...)
			return nil
		}
	}
	return nil // idempotent
}

func (s *Store) ListImplicitGrants() []ImplicitGrant {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ImplicitGrant, len(s.implicitGrants))
	copy(result, s.implicitGrants)
	return result
}
