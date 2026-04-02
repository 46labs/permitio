package store

import (
	"fmt"
	"time"
)

func (s *Store) CreateUser(key string, email, firstName, lastName *string, attributes map[string]interface{}) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[key]; exists {
		return nil, fmt.Errorf("user %q already exists", key)
	}
	u := &User{
		BaseFields: newBase(),
		Key:        key,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
		Attributes: attributes,
	}
	s.users[key] = u
	return u, nil
}

func (s *Store) GetUser(key string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[key]
	if !ok {
		return nil, fmt.Errorf("user %q not found", key)
	}
	return u, nil
}

func (s *Store) ListUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u)
	}
	return result
}

func (s *Store) UpdateUser(key string, email, firstName, lastName *string, attributes map[string]interface{}) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[key]
	if !ok {
		return nil, fmt.Errorf("user %q not found", key)
	}
	if email != nil {
		u.Email = email
	}
	if firstName != nil {
		u.FirstName = firstName
	}
	if lastName != nil {
		u.LastName = lastName
	}
	if attributes != nil {
		u.Attributes = attributes
	}
	u.UpdatedAt = time.Now().UTC()
	return u, nil
}

func (s *Store) UpsertUser(key string, email, firstName, lastName *string, attributes map[string]interface{}) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, exists := s.users[key]
	if exists {
		if email != nil {
			u.Email = email
		}
		if firstName != nil {
			u.FirstName = firstName
		}
		if lastName != nil {
			u.LastName = lastName
		}
		if attributes != nil {
			u.Attributes = attributes
		}
		u.UpdatedAt = time.Now().UTC()
		return u, nil
	}
	u = &User{
		BaseFields: newBase(),
		Key:        key,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
		Attributes: attributes,
	}
	s.users[key] = u
	return u, nil
}

func (s *Store) DeleteUser(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[key]; !ok {
		return fmt.Errorf("user %q not found", key)
	}
	delete(s.users, key)
	return nil
}
