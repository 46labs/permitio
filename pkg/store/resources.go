package store

import (
	"fmt"
	"time"
)

type ActionBlockInput struct {
	Name        string
	Description *string
}

func (s *Store) CreateResource(key, name string, actions map[string]ActionBlockInput) (*Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.resources[key]; exists {
		return nil, fmt.Errorf("resource %q already exists", key)
	}
	r := &Resource{
		BaseFields: newBase(),
		Key:        key,
		Name:       name,
		Actions:    make(map[string]ActionBlock),
	}
	for ak, av := range actions {
		r.Actions[ak] = ActionBlock{Name: av.Name, Description: av.Description, ID: generateID()}
	}
	s.resources[key] = r
	return r, nil
}

func (s *Store) GetResource(key string) (*Resource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.resources[key]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", key)
	}
	return r, nil
}

func (s *Store) ListResources() []*Resource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Resource, 0, len(s.resources))
	for _, r := range s.resources {
		result = append(result, r)
	}
	return result
}

func (s *Store) UpdateResource(key string, name *string, actions map[string]ActionBlockInput) (*Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.resources[key]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", key)
	}
	if name != nil {
		r.Name = *name
	}
	if actions != nil {
		r.Actions = make(map[string]ActionBlock)
		for ak, av := range actions {
			r.Actions[ak] = ActionBlock{Name: av.Name, Description: av.Description, ID: generateID()}
		}
	}
	r.UpdatedAt = time.Now().UTC()
	return r, nil
}

func (s *Store) DeleteResource(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.resources[key]; !ok {
		return fmt.Errorf("resource %q not found", key)
	}
	delete(s.resources, key)
	delete(s.resourceRoles, key)
	delete(s.relations, key)
	return nil
}

// --- Resource Actions CRUD ---

func (s *Store) actionToRead(resourceKey string, actionKey string, ab ActionBlock, res *Resource) *ResourceActionRead {
	now := time.Now().UTC()
	return &ResourceActionRead{
		ID:             ab.ID,
		Key:            actionKey,
		Name:           ab.Name,
		Description:    ab.Description,
		PermissionName: resourceKey + ":" + actionKey,
		ResourceID:     res.ID,
		OrganizationID: MockOrgID,
		ProjectID:      MockProjID,
		EnvironmentID:  MockEnvID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (s *Store) GetResourceAction(resourceKey, actionKey string) (*ResourceActionRead, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res, ok := s.resources[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceKey)
	}
	ab, ok := res.Actions[actionKey]
	if !ok {
		return nil, fmt.Errorf("action %q not found on resource %q", actionKey, resourceKey)
	}
	return s.actionToRead(resourceKey, actionKey, ab, res), nil
}

func (s *Store) ListResourceActions(resourceKey string) ([]*ResourceActionRead, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res, ok := s.resources[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceKey)
	}
	result := make([]*ResourceActionRead, 0, len(res.Actions))
	for ak, ab := range res.Actions {
		result = append(result, s.actionToRead(resourceKey, ak, ab, res))
	}
	return result, nil
}

func (s *Store) CreateResourceAction(resourceKey, actionKey, name string, description *string) (*ResourceActionRead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, ok := s.resources[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceKey)
	}
	if _, exists := res.Actions[actionKey]; exists {
		return nil, fmt.Errorf("action %q already exists on resource %q", actionKey, resourceKey)
	}
	ab := ActionBlock{
		Name:        name,
		Description: description,
		ID:          generateID(),
	}
	res.Actions[actionKey] = ab
	s.materializeUnlocked()
	return s.actionToRead(resourceKey, actionKey, ab, res), nil
}

func (s *Store) UpdateResourceAction(resourceKey, actionKey string, name *string, description *string) (*ResourceActionRead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, ok := s.resources[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceKey)
	}
	ab, ok := res.Actions[actionKey]
	if !ok {
		return nil, fmt.Errorf("action %q not found on resource %q", actionKey, resourceKey)
	}
	if name != nil {
		ab.Name = *name
	}
	if description != nil {
		ab.Description = description
	}
	res.Actions[actionKey] = ab
	return s.actionToRead(resourceKey, actionKey, ab, res), nil
}

func (s *Store) DeleteResourceAction(resourceKey, actionKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, ok := s.resources[resourceKey]
	if !ok {
		return fmt.Errorf("resource %q not found", resourceKey)
	}
	if _, ok := res.Actions[actionKey]; !ok {
		return fmt.Errorf("action %q not found on resource %q", actionKey, resourceKey)
	}
	delete(res.Actions, actionKey)
	s.materializeUnlocked()
	return nil
}
