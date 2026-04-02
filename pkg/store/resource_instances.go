package store

import (
	"fmt"
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
	return nil
}
