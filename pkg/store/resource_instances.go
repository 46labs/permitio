package store

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

func (s *Store) CreateResourceInstance(key, resource string, tenant *string, attributes map[string]interface{}) (*ResourceInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	instKey := fmt.Sprintf("%s:%s", resource, key)
	if _, exists := s.resourceInstances[instKey]; exists {
		return nil, fmt.Errorf("resource instance %q already exists", instKey)
	}
	ri := &ResourceInstance{
		BaseFields: newBase(),
		Key:        key,
		Resource:   resource,
		Tenant:     tenant,
		Attributes: attributes,
	}
	// Fill in resource ID
	if res, ok := s.resources[resource]; ok {
		ri.ResourceID = res.ID
	}
	// Fill in tenant ID
	if tenant != nil {
		if t, ok := s.tenants[*tenant]; ok {
			ri.TenantID = &t.ID
		}
	}
	s.resourceInstances[instKey] = ri
	// An assignment accepted before the instance existed is rechecked against its tenant.
	s.materializeUnlocked()
	return ri, nil
}

func (s *Store) GetResourceInstance(instKey string) (*ResourceInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ri, ok := s.resourceInstances[instKey]
	if !ok {
		return nil, fmt.Errorf("resource instance %q not found", instKey)
	}
	return ri, nil
}

func (s *Store) ListResourceInstances() []*ResourceInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ResourceInstance, 0, len(s.resourceInstances))
	for _, ri := range s.resourceInstances {
		result = append(result, ri)
	}
	return result
}

func (s *Store) UpdateResourceInstance(instKey string, attributes map[string]interface{}) (*ResourceInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ri, ok := s.resourceInstances[instKey]
	if !ok {
		return nil, fmt.Errorf("resource instance %q not found", instKey)
	}
	if attributes != nil {
		ri.Attributes = attributes
	}
	ri.UpdatedAt = time.Now().UTC()
	return ri, nil
}

func (s *Store) DeleteResourceInstance(instKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.resourceInstances[instKey]; !ok {
		return fmt.Errorf("resource instance %q not found", instKey)
	}
	delete(s.resourceInstances, instKey)

	// Permit removes an instance's role assignments and relationships with it.
	s.roleAssignments = slices.DeleteFunc(s.roleAssignments, func(ra RoleAssignment) bool {
		return ra.ResourceInstance != nil && *ra.ResourceInstance == instKey
	})
	s.relationshipTuples = slices.DeleteFunc(s.relationshipTuples, func(rt RelationshipTuple) bool {
		return rt.Subject == instKey || rt.Object == instKey
	})
	s.materializeUnlocked()
	return nil
}

// ListResourceInstancesFiltered lists instances in a stable order, narrowed
// by tenant, resource type, and a key substring when given.
func (s *Store) ListResourceInstancesFiltered(tenant, resource, search string) []*ResourceInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ResourceInstance, 0, len(s.resourceInstances))
	for _, ri := range s.resourceInstances {
		if tenant != "" && (ri.Tenant == nil || *ri.Tenant != tenant) {
			continue
		}
		if resource != "" && ri.Resource != resource {
			continue
		}
		if search != "" && !strings.Contains(ri.Key, search) {
			continue
		}
		result = append(result, ri)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Resource != result[j].Resource {
			return result[i].Resource < result[j].Resource
		}
		return result[i].Key < result[j].Key
	})
	return result
}
