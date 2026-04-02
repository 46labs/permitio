package store

import (
	"fmt"
	"time"
)

func (s *Store) CreateResourceRole(resourceKey, key, name string, permissions []string) (*ResourceRole, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, ok := s.resources[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceKey)
	}
	if s.resourceRoles[resourceKey] == nil {
		s.resourceRoles[resourceKey] = make(map[string]*ResourceRole)
	}
	if _, exists := s.resourceRoles[resourceKey][key]; exists {
		return nil, fmt.Errorf("resource role %q already exists on resource %q", key, resourceKey)
	}
	rr := &ResourceRole{
		BaseFields:  newBase(),
		Key:         key,
		Name:        name,
		Permissions: permissions,
		ResourceID:  res.ID,
	}
	s.resourceRoles[resourceKey][key] = rr
	return rr, nil
}

func (s *Store) GetResourceRole(resourceKey, key string) (*ResourceRole, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	roles, ok := s.resourceRoles[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource role %q not found on resource %q", key, resourceKey)
	}
	rr, ok := roles[key]
	if !ok {
		return nil, fmt.Errorf("resource role %q not found on resource %q", key, resourceKey)
	}
	return rr, nil
}

func (s *Store) ListResourceRoles(resourceKey string) []*ResourceRole {
	s.mu.RLock()
	defer s.mu.RUnlock()
	roles := s.resourceRoles[resourceKey]
	result := make([]*ResourceRole, 0, len(roles))
	for _, rr := range roles {
		result = append(result, rr)
	}
	return result
}

func (s *Store) UpdateResourceRole(resourceKey, key string, name *string, description *string, permissions []string) (*ResourceRole, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	roles, ok := s.resourceRoles[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource role %q not found on resource %q", key, resourceKey)
	}
	rr, ok := roles[key]
	if !ok {
		return nil, fmt.Errorf("resource role %q not found on resource %q", key, resourceKey)
	}
	if name != nil {
		rr.Name = *name
	}
	if description != nil {
		rr.Description = description
	}
	if permissions != nil {
		rr.Permissions = permissions
	}
	rr.UpdatedAt = time.Now().UTC()
	return rr, nil
}

func (s *Store) AssignPermissionsToResourceRole(resourceKey, key string, permissions []string) (*ResourceRole, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	roles, ok := s.resourceRoles[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource role %q not found on resource %q", key, resourceKey)
	}
	rr, ok := roles[key]
	if !ok {
		return nil, fmt.Errorf("resource role %q not found on resource %q", key, resourceKey)
	}
	existing := make(map[string]bool)
	for _, p := range rr.Permissions {
		existing[p] = true
	}
	for _, p := range permissions {
		if !existing[p] {
			rr.Permissions = append(rr.Permissions, p)
			existing[p] = true
		}
	}
	rr.UpdatedAt = time.Now().UTC()
	return rr, nil
}

func (s *Store) RemovePermissionsFromResourceRole(resourceKey, key string, permissions []string) (*ResourceRole, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	roles, ok := s.resourceRoles[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource role %q not found on resource %q", key, resourceKey)
	}
	rr, ok := roles[key]
	if !ok {
		return nil, fmt.Errorf("resource role %q not found on resource %q", key, resourceKey)
	}
	toRemove := make(map[string]bool)
	for _, p := range permissions {
		toRemove[p] = true
	}
	filtered := make([]string, 0)
	for _, p := range rr.Permissions {
		if !toRemove[p] {
			filtered = append(filtered, p)
		}
	}
	rr.Permissions = filtered
	rr.UpdatedAt = time.Now().UTC()
	return rr, nil
}

func (s *Store) DeleteResourceRole(resourceKey, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	roles, ok := s.resourceRoles[resourceKey]
	if !ok {
		return fmt.Errorf("resource role %q not found on resource %q", key, resourceKey)
	}
	if _, ok := roles[key]; !ok {
		return fmt.Errorf("resource role %q not found on resource %q", key, resourceKey)
	}
	delete(roles, key)
	return nil
}
