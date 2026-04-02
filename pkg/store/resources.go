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
