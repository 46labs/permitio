package store

import (
	"fmt"
	"strings"
)

// Materialize rebuilds all indexes and the effectivePerms map from current state.
// Called after every write operation that affects permissions.
func (s *Store) Materialize() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.materializeUnlocked()
}

func (s *Store) materializeUnlocked() {
	s.tupleIndex = make(map[string]map[string][]string)
	s.effectivePerms = make(map[string]bool)
	s.userPerms = make(map[string]map[string][]string)
	s.userRoles = make(map[string]map[string][]string)

	// 1. Build tupleIndex from relationship tuples.
	//    tupleIndex maps subject "type:key" -> relation -> []object "type:key"
	for _, rt := range s.relationshipTuples {
		if s.tupleIndex[rt.Subject] == nil {
			s.tupleIndex[rt.Subject] = make(map[string][]string)
		}
		s.tupleIndex[rt.Subject][rt.Relation] = append(s.tupleIndex[rt.Subject][rt.Relation], rt.Object)
	}

	// 2. Process each role assignment.
	for _, ra := range s.roleAssignments {
		user := ra.User
		tenant := ra.Tenant

		// Ensure maps exist for this user.
		if s.userPerms[user] == nil {
			s.userPerms[user] = make(map[string][]string)
		}
		if s.userRoles[user] == nil {
			s.userRoles[user] = make(map[string][]string)
		}

		if ra.ResourceInstance != nil {
			// Resource-instance-level role assignment (ReBAC).
			instKey := *ra.ResourceInstance // e.g. "folder:budget"
			parts := strings.SplitN(instKey, ":", 2)
			if len(parts) != 2 {
				continue
			}
			resType := parts[0]
			instanceKey := parts[1]
			roleKey := ra.Role

			// Look up the resource role.
			resRoles, ok := s.resourceRoles[resType]
			if !ok {
				continue
			}
			rr, ok := resRoles[roleKey]
			if !ok {
				continue
			}

			// Track role for the user in this tenant.
			roleLabel := fmt.Sprintf("%s#%s", resType, roleKey)
			s.userRoles[user][tenant] = appendUnique(s.userRoles[user][tenant], roleLabel)

			// Grant direct permissions from the role.
			for _, perm := range s.resolveResourceRolePermissions(resType, roleKey) {
				permKey := fmt.Sprintf("%s|%s|%s|%s|%s", user, perm.action, perm.resource, instanceKey, tenant)
				s.effectivePerms[permKey] = true
				permLabel := fmt.Sprintf("%s:%s", perm.resource, perm.action)
				s.userPerms[user][tenant] = appendUnique(s.userPerms[user][tenant], permLabel)
			}

			// Follow implicit grants.
			_ = rr // already looked up
			for _, ig := range s.implicitGrants {
				// The implicit grant model:
				// Created at resources/{targetResource}/roles/{targetRole}/implicit_grants
				// Body: {role: sourceRole, on_resource: sourceResource, linked_by_relation: relation}
				//
				// Meaning: if a user has sourceRole on sourceResource instance X,
				// and X --relation--> targetResource instance Y (via tuple),
				// then the user gets targetRole on Y.
				//
				// Match: source resource type and source role must match.
				if ig.OnResource != resType || ig.Role != roleKey {
					continue
				}

				// Follow tuples from the source instance via the relation.
				targets := s.tupleIndex[instKey][ig.LinkedByRelation]
				for _, targetInst := range targets {
					targetParts := strings.SplitN(targetInst, ":", 2)
					if len(targetParts) != 2 {
						continue
					}
					targetType := targetParts[0]
					targetKey := targetParts[1]

					// Verify the target resource type matches.
					if targetType != ig.TargetResource {
						continue
					}

					// Grant the derived (target) role's permissions on the target instance.
					for _, perm := range s.resolveResourceRolePermissions(targetType, ig.TargetRole) {
						permKey := fmt.Sprintf("%s|%s|%s|%s|%s", user, perm.action, perm.resource, targetKey, tenant)
						s.effectivePerms[permKey] = true
						permLabel := fmt.Sprintf("%s:%s", perm.resource, perm.action)
						s.userPerms[user][tenant] = appendUnique(s.userPerms[user][tenant], permLabel)
					}

					// Track the derived role.
					derivedRoleLabel := fmt.Sprintf("%s#%s", targetType, ig.TargetRole)
					s.userRoles[user][tenant] = appendUnique(s.userRoles[user][tenant], derivedRoleLabel)
				}
			}
		} else {
			// Global (tenant-level) role assignment.
			roleKey := ra.Role
			role, ok := s.roles[roleKey]
			if !ok {
				continue
			}

			s.userRoles[user][tenant] = appendUnique(s.userRoles[user][tenant], roleKey)

			for _, perm := range role.Permissions {
				// Global role permissions are in "resource:action" format.
				s.userPerms[user][tenant] = appendUnique(s.userPerms[user][tenant], perm)

				// Also add to effectivePerms for wildcard matching.
				// For global roles, the permission applies to all instances.
				// We mark it with "*" as the instance key.
				parts := strings.SplitN(perm, ":", 2)
				if len(parts) == 2 {
					permKey := fmt.Sprintf("%s|%s|%s|*|%s", user, parts[1], parts[0], tenant)
					s.effectivePerms[permKey] = true
				}
			}
		}
	}
}

type resolvedPerm struct {
	resource string
	action   string
}

// resolveResourceRolePermissions returns the full set of permissions for a resource role,
// including permissions from extended roles.
func (s *Store) resolveResourceRolePermissions(resType, roleKey string) []resolvedPerm {
	resRoles, ok := s.resourceRoles[resType]
	if !ok {
		return nil
	}
	rr, ok := resRoles[roleKey]
	if !ok {
		return nil
	}

	var perms []resolvedPerm
	seen := make(map[string]bool)

	// Direct permissions.
	for _, p := range rr.Permissions {
		parts := strings.SplitN(p, ":", 2)
		if len(parts) == 2 {
			key := parts[0] + ":" + parts[1]
			if !seen[key] {
				perms = append(perms, resolvedPerm{resource: parts[0], action: parts[1]})
				seen[key] = true
			}
		}
	}

	// Extended roles (recursion with visited set to avoid loops).
	for _, ext := range rr.Extends {
		for _, p := range s.resolveResourceRolePermissions(resType, ext) {
			key := p.resource + ":" + p.action
			if !seen[key] {
				perms = append(perms, p)
				seen[key] = true
			}
		}
	}

	return perms
}

// CheckPermission checks if a user has a specific permission.
func (s *Store) CheckPermission(user, action, resourceType, instanceKey, tenant string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.allowAll {
		return true
	}

	// Check specific instance permission.
	permKey := fmt.Sprintf("%s|%s|%s|%s|%s", user, action, resourceType, instanceKey, tenant)
	if s.effectivePerms[permKey] {
		return true
	}

	// Check wildcard (global role) permission.
	wildcardKey := fmt.Sprintf("%s|%s|%s|*|%s", user, action, resourceType, tenant)
	if s.effectivePerms[wildcardKey] {
		return true
	}

	return false
}

// TenantCheckResult represents a single tenant's check result.
type TenantCheckResult struct {
	Allow  bool                   `json:"allow"`
	Result bool                   `json:"result"`
	Tenant TenantResult           `json:"tenant"`
}

// TenantResult represents tenant info in a check response.
type TenantResult struct {
	Key        string                 `json:"key"`
	Attributes map[string]interface{} `json:"attributes"`
}

// UserPermissionSet represents permissions for a user in a tenant.
type UserPermissionSet struct {
	Tenant      TenantResult `json:"tenant"`
	Permissions []string     `json:"permissions"`
	Roles       []string     `json:"roles"`
}

// GetUserPermissions returns the permissions for a user across the given tenants.
// If tenants is empty, returns permissions for all tenants.
func (s *Store) GetUserPermissions(user string, tenants []string) map[string]UserPermissionSet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]UserPermissionSet)

	userPerms := s.userPerms[user]
	userRoles := s.userRoles[user]

	if userPerms == nil && userRoles == nil {
		return result
	}

	// Determine which tenants to include.
	tenantSet := make(map[string]bool)
	if len(tenants) > 0 {
		for _, t := range tenants {
			tenantSet[t] = true
		}
	} else {
		// All tenants.
		for t := range s.tenants {
			tenantSet[t] = true
		}
	}

	for tenantKey := range tenantSet {
		perms := userPerms[tenantKey]
		roles := userRoles[tenantKey]
		if len(perms) == 0 && len(roles) == 0 {
			continue
		}

		attrs := make(map[string]interface{})
		if t, ok := s.tenants[tenantKey]; ok && t.Attributes != nil {
			attrs = t.Attributes
		}

		key := fmt.Sprintf("__tenant:%s", tenantKey)
		result[key] = UserPermissionSet{
			Tenant: TenantResult{
				Key:        tenantKey,
				Attributes: attrs,
			},
			Permissions: perms,
			Roles:       roles,
		}
	}

	return result
}

// GetAllowedTenants checks permission across all tenants and returns the ones where it's allowed.
func (s *Store) GetAllowedTenants(user, action, resourceType, instanceKey string) []TenantCheckResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []TenantCheckResult

	for tenantKey, tenant := range s.tenants {
		allowed := false
		if s.allowAll {
			allowed = true
		} else {
			permKey := fmt.Sprintf("%s|%s|%s|%s|%s", user, action, resourceType, instanceKey, tenantKey)
			if s.effectivePerms[permKey] {
				allowed = true
			}
			if !allowed {
				wildcardKey := fmt.Sprintf("%s|%s|%s|*|%s", user, action, resourceType, tenantKey)
				if s.effectivePerms[wildcardKey] {
					allowed = true
				}
			}
		}

		if allowed {
			attrs := make(map[string]interface{})
			if tenant.Attributes != nil {
				attrs = tenant.Attributes
			}
			results = append(results, TenantCheckResult{
				Allow:  true,
				Result: true,
				Tenant: TenantResult{
					Key:        tenantKey,
					Attributes: attrs,
				},
			})
		}
	}

	return results
}

func appendUnique(slice []string, val string) []string {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}
