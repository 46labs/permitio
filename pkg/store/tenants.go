package store

import (
	"fmt"
	"time"
)

func (s *Store) CreateTenant(key, name string) (*Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tenants[key]; exists {
		return nil, fmt.Errorf("tenant %q already exists", key)
	}
	t := &Tenant{
		BaseFields:   newBase(),
		Key:          key,
		Name:         name,
		LastActionAt: time.Now().UTC(),
	}
	s.tenants[key] = t
	return t, nil
}

func (s *Store) GetTenant(key string) (*Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tenants[key]
	if !ok {
		return nil, fmt.Errorf("tenant %q not found", key)
	}
	return t, nil
}

func (s *Store) ListTenants() []*Tenant {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		result = append(result, t)
	}
	return result
}

func (s *Store) UpdateTenant(key string, name *string, description *string, attributes map[string]interface{}) (*Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[key]
	if !ok {
		return nil, fmt.Errorf("tenant %q not found", key)
	}
	if name != nil {
		t.Name = *name
	}
	if description != nil {
		t.Description = description
	}
	if attributes != nil {
		t.Attributes = attributes
	}
	t.UpdatedAt = time.Now().UTC()
	return t, nil
}

func (s *Store) DeleteTenant(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[key]; !ok {
		return fmt.Errorf("tenant %q not found", key)
	}
	delete(s.tenants, key)
	return nil
}

func (s *Store) ListTenantUsers(tenantKey string) []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userKeys := make(map[string]bool)
	for _, ra := range s.roleAssignments {
		if ra.Tenant == tenantKey {
			userKeys[ra.User] = true
		}
	}

	var users []*User
	for key := range userKeys {
		if u, ok := s.users[key]; ok {
			users = append(users, u)
		}
	}
	return users
}
