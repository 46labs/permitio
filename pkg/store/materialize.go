package store

import (
	"fmt"
	"maps"
	"slices"
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
	s.tupleIndex = make(map[string]map[string][]tupleEdge)
	s.effectivePerms = make(map[string]bool)
	s.userPerms = make(map[string]map[string][]string)
	s.userRoles = make(map[string]map[string][]string)
	s.instancePerms = make(map[string]map[string]map[string][]string)
	s.instanceRoles = make(map[string]map[string]map[string][]string)

	for _, rt := range s.relationshipTuples {
		if s.tupleIndex[rt.Subject] == nil {
			s.tupleIndex[rt.Subject] = make(map[string][]tupleEdge)
		}
		s.tupleIndex[rt.Subject][rt.Relation] = append(s.tupleIndex[rt.Subject][rt.Relation], tupleEdge{object: rt.Object, tenant: rt.Tenant})
	}

	for _, ra := range s.roleAssignments {
		if ra.ResourceInstance != nil {
			s.materializeInstanceAssignment(ra)
			continue
		}

		role, ok := s.roles[ra.Role]
		if !ok {
			continue
		}
		s.userRoles[ra.User] = appendNested(s.userRoles[ra.User], ra.Tenant, ra.Role)
		for _, perm := range role.Permissions {
			s.userPerms[ra.User] = appendNested(s.userPerms[ra.User], ra.Tenant, perm)
			// A tenant-level role applies to every instance, marked "*".
			parts := strings.SplitN(perm, ":", 2)
			if len(parts) == 2 {
				s.effectivePerms[fmt.Sprintf("%s|%s|%s|*|%s", ra.User, parts[1], parts[0], ra.Tenant)] = true
			}
		}
	}
}

// tupleEdge is one relationship tuple seen from its subject.
type tupleEdge struct {
	object string
	tenant string
}

// derivedGrant is a resource role a user holds on one instance.
type derivedGrant struct {
	resType string
	key     string
	role    string
}

// materializeInstanceAssignment grants a resource role and follows implicit
// grants through relationship tuples to any depth.
func (s *Store) materializeInstanceAssignment(ra RoleAssignment) {
	parts := strings.SplitN(*ra.ResourceInstance, ":", 2)
	if len(parts) != 2 || !s.instanceInTenant(*ra.ResourceInstance, ra.Tenant) {
		return
	}

	queue := []derivedGrant{{resType: parts[0], key: parts[1], role: ra.Role}}
	visited := make(map[derivedGrant]bool)
	for len(queue) > 0 {
		g := queue[0]
		queue = queue[1:]
		if visited[g] {
			continue
		}
		visited[g] = true
		if _, ok := s.resourceRoles[g.resType][g.role]; !ok {
			continue
		}

		instKey := g.resType + ":" + g.key
		appendInstance(s.instanceRoles, ra.User, ra.Tenant, instKey, g.resType+"#"+g.role)
		for _, perm := range s.resolveResourceRolePermissions(g.resType, g.role) {
			s.effectivePerms[fmt.Sprintf("%s|%s|%s|%s|%s", ra.User, perm.action, perm.resource, g.key, ra.Tenant)] = true
			appendInstance(s.instancePerms, ra.User, ra.Tenant, instKey, perm.resource+":"+perm.action)
		}

		// An implicit grant gives TargetRole on Y when the user holds Role on
		// X of type OnResource and X --LinkedByRelation--> Y.
		for _, ig := range s.implicitGrants {
			if ig.OnResource != g.resType || ig.Role != g.role {
				continue
			}
			for _, edge := range s.tupleIndex[instKey][ig.LinkedByRelation] {
				tp := strings.SplitN(edge.object, ":", 2)
				if len(tp) != 2 || tp[0] != ig.TargetResource {
					continue
				}
				if !s.edgeInTenant(edge, ra.Tenant) {
					continue
				}
				queue = append(queue, derivedGrant{resType: tp[0], key: tp[1], role: ig.TargetRole})
			}
		}
	}
}

// instanceTenant returns the tenant of a created instance that has one.
func (s *Store) instanceTenant(instKey string) (string, bool) {
	ri, ok := s.resourceInstances[instKey]
	if !ok || ri.Tenant == nil || *ri.Tenant == "" {
		return "", false
	}
	return *ri.Tenant, true
}

// instanceInTenant accepts an instance not yet created or created without a tenant.
func (s *Store) instanceInTenant(instKey, tenant string) bool {
	owner, ok := s.instanceTenant(instKey)
	return !ok || owner == tenant
}

// edgeInTenant uses the tuple's tenant only when its target has none, as Permit does.
func (s *Store) edgeInTenant(edge tupleEdge, tenant string) bool {
	if owner, ok := s.instanceTenant(edge.object); ok {
		return owner == tenant
	}
	return edge.tenant == "" || edge.tenant == tenant
}

// appendInstance adds val under user -> tenant -> instKey.
func appendInstance(m map[string]map[string]map[string][]string, user, tenant, instKey, val string) {
	if m[user] == nil {
		m[user] = make(map[string]map[string][]string)
	}
	m[user][tenant] = appendNested(m[user][tenant], instKey, val)
}

func appendNested(m map[string][]string, key, val string) map[string][]string {
	if m == nil {
		m = make(map[string][]string)
	}
	m[key] = appendUnique(m[key], val)
	return m
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
	return s.effectivePerms[wildcardKey]
}

// TenantCheckResult represents a single tenant's check result.
type TenantCheckResult struct {
	Allow  bool         `json:"allow"`
	Result bool         `json:"result"`
	Tenant TenantResult `json:"tenant"`
}

// TenantResult represents tenant info in a check response.
type TenantResult struct {
	Key        string                 `json:"key"`
	Attributes map[string]interface{} `json:"attributes"`
}

// UserPermissionSet represents permissions for a user on a tenant or on one
// resource instance.
type UserPermissionSet struct {
	Tenant      TenantResult     `json:"tenant"`
	Resource    *ResourceDetails `json:"resource,omitempty"`
	Permissions []string         `json:"permissions"`
	Roles       []string         `json:"roles"`
}

// ResourceDetails identifies the instance a permission set applies to.
type ResourceDetails struct {
	Type       string                 `json:"type"`
	Key        string                 `json:"key"`
	Attributes map[string]interface{} `json:"attributes"`
}

// TenantResourceType is the resource type naming tenant-level permission sets.
const TenantResourceType = "__tenant"

// GetUserPermissions returns a user's permission sets, keyed "__tenant:<key>"
// for tenant-level roles and "<type>:<key>" for resource instances. Empty
// tenants means all tenants. resources ("type:key") and resourceTypes narrow
// the sets returned; "__tenant" in resourceTypes selects tenant-level sets.
func (s *Store) GetUserPermissions(user string, tenants, resources, resourceTypes []string) map[string]UserPermissionSet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]UserPermissionSet)
	if len(tenants) == 0 {
		tenants = slices.Collect(maps.Keys(s.tenants))
	}
	wanted := func(resType, instKey string) bool {
		if len(resources) == 0 && len(resourceTypes) == 0 {
			return true
		}
		return slices.Contains(resources, instKey) || slices.Contains(resourceTypes, resType)
	}

	for _, tenantKey := range tenants {
		tenant := TenantResult{Key: tenantKey, Attributes: s.tenantAttributes(tenantKey)}

		perms, roles := s.userPerms[user][tenantKey], s.userRoles[user][tenantKey]
		if (len(perms) > 0 || len(roles) > 0) && wanted(TenantResourceType, TenantResourceType+":"+tenantKey) {
			result[TenantResourceType+":"+tenantKey] = UserPermissionSet{
				Tenant:      tenant,
				Permissions: nonNil(perms),
				Roles:       nonNil(roles),
			}
		}

		for instKey, instRoles := range s.instanceRoles[user][tenantKey] {
			parts := strings.SplitN(instKey, ":", 2)
			if !wanted(parts[0], instKey) {
				continue
			}
			attrs := make(map[string]interface{})
			if ri, ok := s.resourceInstances[instKey]; ok && ri.Attributes != nil {
				attrs = ri.Attributes
			}
			result[instKey] = UserPermissionSet{
				Tenant:      tenant,
				Resource:    &ResourceDetails{Type: parts[0], Key: parts[1], Attributes: attrs},
				Permissions: nonNil(s.instancePerms[user][tenantKey][instKey]),
				Roles:       instRoles,
			}
		}
	}

	return result
}

func (s *Store) tenantAttributes(key string) map[string]interface{} {
	if t, ok := s.tenants[key]; ok && t.Attributes != nil {
		return t.Attributes
	}
	return make(map[string]interface{})
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
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
