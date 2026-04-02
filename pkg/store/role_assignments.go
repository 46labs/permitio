package store

import (
	"fmt"
	"time"
)

func (s *Store) CreateRoleAssignment(user, role, tenant string) (*RoleAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Check for duplicates
	for _, ra := range s.roleAssignments {
		if ra.User == user && ra.Role == role && ra.Tenant == tenant {
			return nil, fmt.Errorf("role assignment already exists")
		}
	}
	ra := RoleAssignment{
		ID:             generateID(),
		User:           user,
		Role:           role,
		Tenant:         tenant,
		OrganizationID: MockOrgID,
		ProjectID:      MockProjID,
		EnvironmentID:  MockEnvID,
		CreatedAt:      time.Now().UTC(),
	}
	// Fill IDs from store
	if u, ok := s.users[user]; ok {
		ra.UserID = u.ID
	}
	if r, ok := s.roles[role]; ok {
		ra.RoleID = r.ID
	}
	if t, ok := s.tenants[tenant]; ok {
		ra.TenantID = t.ID
	}
	s.roleAssignments = append(s.roleAssignments, ra)
	return &ra, nil
}

func (s *Store) ListRoleAssignments() []RoleAssignment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]RoleAssignment, len(s.roleAssignments))
	copy(result, s.roleAssignments)
	return result
}

func (s *Store) DeleteRoleAssignment(user, role, tenant string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, ra := range s.roleAssignments {
		if ra.User == user && ra.Role == role && ra.Tenant == tenant {
			s.roleAssignments = append(s.roleAssignments[:i], s.roleAssignments[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("role assignment not found")
}
