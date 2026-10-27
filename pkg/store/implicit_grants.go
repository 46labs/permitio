package store

func (s *Store) CreateImplicitGrant(resourceKey, roleKey string, ig ImplicitGrant) (*ImplicitGrant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Store target resource and role from the URL path.
	ig.TargetResource = resourceKey
	ig.TargetRole = roleKey
	// The same derivation twice is one grant: answer with the existing one.
	for _, existing := range s.implicitGrants {
		if sameGrant(existing, ig) {
			found := existing
			return &found, nil
		}
	}
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
	s.materializeUnlocked()
	return &ig, nil
}

func (s *Store) DeleteImplicitGrant(resourceKey, roleKey string, ig ImplicitGrant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ig.TargetResource = resourceKey
	ig.TargetRole = roleKey
	for i, existing := range s.implicitGrants {
		if sameGrant(existing, ig) {
			s.implicitGrants = append(s.implicitGrants[:i], s.implicitGrants[i+1:]...)
			s.materializeUnlocked()
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

// sameGrant: one derivation is the granted role (target) and the rule that
// derives it (source role, its resource, and the relation followed).
func sameGrant(a, b ImplicitGrant) bool {
	return a.TargetResource == b.TargetResource && a.TargetRole == b.TargetRole &&
		a.Role == b.Role && a.OnResource == b.OnResource && a.LinkedByRelation == b.LinkedByRelation
}

// grantedToUnlocked is a role's derivation rules, as Permit returns them on
// the role (granted_to.users_with_role); nil when none derive it.
func (s *Store) grantedToUnlocked(resourceKey, roleKey string) *GrantedTo {
	var rules []ImplicitGrant
	for _, ig := range s.implicitGrants {
		if ig.TargetResource == resourceKey && ig.TargetRole == roleKey {
			rules = append(rules, ig)
		}
	}
	if len(rules) == 0 {
		return nil
	}
	return &GrantedTo{ID: resourceKey + "#" + roleKey, UsersWithRole: rules}
}
