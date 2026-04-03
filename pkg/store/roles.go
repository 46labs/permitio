package store

import (
	"fmt"
	"time"
)

func (s *Store) CreateRole(key, name string, permissions []string) (*Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.roles[key]; exists {
		return nil, fmt.Errorf("role %q already exists", key)
	}
	r := &Role{
		BaseFields:  newBase(),
		Key:         key,
		Name:        name,
		Permissions: permissions,
	}
	s.roles[key] = r
	return r, nil
}

func (s *Store) GetRole(key string) (*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	return r, nil
}

func (s *Store) ListRoles() []*Role {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Role, 0, len(s.roles))
	for _, r := range s.roles {
		result = append(result, r)
	}
	return result
}

func (s *Store) UpdateRole(key string, name *string, description *string, permissions []string) (*Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	if name != nil {
		r.Name = *name
	}
	if description != nil {
		r.Description = description
	}
	if permissions != nil {
		r.Permissions = permissions
	}
	r.UpdatedAt = time.Now().UTC()
	return r, nil
}

func (s *Store) AssignPermissionsToRole(key string, permissions []string) (*Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	existing := make(map[string]bool)
	for _, p := range r.Permissions {
		existing[p] = true
	}
	for _, p := range permissions {
		if !existing[p] {
			r.Permissions = append(r.Permissions, p)
			existing[p] = true
		}
	}
	r.UpdatedAt = time.Now().UTC()
	return r, nil
}

func (s *Store) RemovePermissionsFromRole(key string, permissions []string) (*Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	toRemove := make(map[string]bool)
	for _, p := range permissions {
		toRemove[p] = true
	}
	filtered := make([]string, 0)
	for _, p := range r.Permissions {
		if !toRemove[p] {
			filtered = append(filtered, p)
		}
	}
	r.Permissions = filtered
	r.UpdatedAt = time.Now().UTC()
	return r, nil
}

func (s *Store) DeleteRole(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.roles[key]; !ok {
		return fmt.Errorf("role %q not found", key)
	}
	delete(s.roles, key)
	return nil
}

func (s *Store) AddParentRole(key, parentKey string) (*Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	role, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	if _, ok := s.roles[parentKey]; !ok {
		return nil, fmt.Errorf("parent role %q not found", parentKey)
	}
	for _, e := range role.Extends {
		if e == parentKey {
			return role, nil
		}
	}
	role.Extends = append(role.Extends, parentKey)
	role.UpdatedAt = time.Now().UTC()
	s.materializeUnlocked()
	return role, nil
}

func (s *Store) RemoveParentRole(key, parentKey string) (*Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	role, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	filtered := make([]string, 0, len(role.Extends))
	for _, e := range role.Extends {
		if e != parentKey {
			filtered = append(filtered, e)
		}
	}
	role.Extends = filtered
	role.UpdatedAt = time.Now().UTC()
	s.materializeUnlocked()
	return role, nil
}
