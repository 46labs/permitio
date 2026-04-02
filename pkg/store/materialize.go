package store

// Materialize rebuilds all indexes and the effectivePerms map from current state.
// Called after every write operation.
func (s *Store) Materialize() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.materializeUnlocked()
}

func (s *Store) materializeUnlocked() {
	// Will be fully implemented in Task 12
	s.tupleIndex = make(map[string]map[string][]string)
	s.effectivePerms = make(map[string]bool)
	s.userPerms = make(map[string]map[string][]string)
	s.userRoles = make(map[string]map[string][]string)
}
