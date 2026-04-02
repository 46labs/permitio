package store

import (
	"fmt"
	"time"
)

func (s *Store) CreateRoleAssignment(user, role, tenant string) (*RoleAssignment, error) {
	return s.CreateRoleAssignmentWithInstance(user, role, tenant, nil)
}

func (s *Store) CreateRoleAssignmentWithInstance(user, role, tenant string, resourceInstance *string) (*RoleAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Check for duplicates
	for _, ra := range s.roleAssignments {
		if ra.User == user && ra.Role == role && ra.Tenant == tenant && ptrStrEq(ra.ResourceInstance, resourceInstance) {
			return nil, fmt.Errorf("role assignment already exists")
		}
	}
	ra := RoleAssignment{
		ID:               generateID(),
		User:             user,
		Role:             role,
		Tenant:           tenant,
		ResourceInstance: resourceInstance,
		OrganizationID:   MockOrgID,
		ProjectID:        MockProjID,
		EnvironmentID:    MockEnvID,
		CreatedAt:        time.Now().UTC(),
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
	s.materializeUnlocked()
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
	return s.DeleteRoleAssignmentWithInstance(user, role, tenant, nil)
}

func (s *Store) DeleteRoleAssignmentWithInstance(user, role, tenant string, resourceInstance *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, ra := range s.roleAssignments {
		if ra.User == user && ra.Role == role && ra.Tenant == tenant && ptrStrEq(ra.ResourceInstance, resourceInstance) {
			s.roleAssignments = append(s.roleAssignments[:i], s.roleAssignments[i+1:]...)
			s.materializeUnlocked()
			return nil
		}
	}
	return fmt.Errorf("role assignment not found")
}

func ptrStrEq(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
