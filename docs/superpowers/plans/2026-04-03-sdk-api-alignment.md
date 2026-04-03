# SDK API Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align the Go mock server with the official Permit Go SDK (v1.2.8) so every SDK management API method works correctly with proper status codes and response shapes.

**Architecture:** Fix status codes on existing fact-creation endpoints. Add missing endpoints (resource actions CRUD, user assigned roles, tenant users, role parents, functional bulk operations). Add CORS middleware. Validate everything through SDK-only tests.

**Tech Stack:** Go 1.24, `github.com/permitio/permit-golang` v1.2.8 SDK, `net/http` stdlib, `testing` stdlib

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `pkg/server/middleware.go` | Modify | Add CORS middleware wrapping existing log middleware |
| `pkg/server/facts_tenants.go` | Modify | 201 status on create, route `tenants/{key}/users` sub-resource |
| `pkg/server/facts_users.go` | Modify | 201 status on create, add GET to `handleUserRoles` |
| `pkg/server/facts_instances.go` | Modify | 201 status on create |
| `pkg/server/facts_tuples.go` | Modify | 201 status on create |
| `pkg/server/facts_role_assignments.go` | Modify | 201 status on create, functional bulk with report responses |
| `pkg/server/schema_resources.go` | Modify | Route `actions` and `parents` sub-resources |
| `pkg/server/schema_roles.go` | Modify | Route `parents` sub-resource |
| `pkg/server/schema_actions.go` | Create | Resource actions CRUD handler |
| `pkg/store/store.go` | Modify | Add new store methods as needed |
| `pkg/store/types.go` | Modify | Add `ResourceActionRead` type |
| `pkg/store/resources.go` | Modify | Add action-level CRUD methods |
| `pkg/store/role_assignments.go` | Modify | Add `ListRoleAssignmentsForUser`, bulk helpers |
| `pkg/store/tenants.go` | Modify | Add `ListTenantUsers` |
| `pkg/store/roles.go` | Modify | Add `AddParentRole`, `RemoveParentRole` |
| `pkg/store/resource_roles.go` | Modify | Add `AddParent`, `RemoveParent` |
| `pkg/server/sdk_compat_test.go` | Create | Comprehensive SDK compatibility tests |
| `pkg/server/sdk_errors_test.go` | Create | Error case tests (404, 409) |

---

### Task 1: CORS Middleware

**Files:**
- Modify: `pkg/server/middleware.go`
- Modify: `pkg/server/server.go:60`

- [ ] **Step 1: Add CORS middleware to middleware.go**

Open `pkg/server/middleware.go` and add a `corsMiddleware` function above `logMiddleware`:

```go
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 2: Wrap the handler chain in server.go**

In `pkg/server/server.go`, change line 60 from:

```go
return logMiddleware(mux)
```

to:

```go
return corsMiddleware(logMiddleware(mux))
```

- [ ] **Step 3: Verify tests still pass**

Run: `cd /home/jarrod/.local/dev/permitio && go test ./pkg/server/ -v -count=1 2>&1 | tail -20`
Expected: All existing tests pass.

- [ ] **Step 4: Commit**

```bash
git add pkg/server/middleware.go pkg/server/server.go
git commit -m "feat: add CORS middleware for browser client support"
```

---

### Task 2: Status Code Fixes — Fact Creation Endpoints Return 201

**Files:**
- Modify: `pkg/server/facts_tenants.go:36`
- Modify: `pkg/server/facts_users.go:40`
- Modify: `pkg/server/facts_instances.go:32`
- Modify: `pkg/server/facts_tuples.go:33`
- Modify: `pkg/server/facts_role_assignments.go:33`

- [ ] **Step 1: Change tenant create to 201**

In `pkg/server/facts_tenants.go`, change line 36 from:

```go
writeJSON(w, http.StatusOK, t)
```

to:

```go
writeJSON(w, http.StatusCreated, t)
```

- [ ] **Step 2: Change user create to 201**

In `pkg/server/facts_users.go`, change line 40 from:

```go
writeJSON(w, http.StatusOK, u)
```

to:

```go
writeJSON(w, http.StatusCreated, u)
```

- [ ] **Step 3: Change resource instance create to 201**

In `pkg/server/facts_instances.go`, change line 32 from:

```go
writeJSON(w, http.StatusOK, ri)
```

to:

```go
writeJSON(w, http.StatusCreated, ri)
```

- [ ] **Step 4: Change relationship tuple create to 201**

In `pkg/server/facts_tuples.go`, change line 33 from:

```go
writeJSON(w, http.StatusOK, rt)
```

to:

```go
writeJSON(w, http.StatusCreated, rt)
```

- [ ] **Step 5: Change role assignment create to 201**

In `pkg/server/facts_role_assignments.go`, change line 33 from:

```go
writeJSON(w, http.StatusOK, ra)
```

to:

```go
writeJSON(w, http.StatusCreated, ra)
```

- [ ] **Step 6: Verify tests still pass**

Run: `cd /home/jarrod/.local/dev/permitio && go test ./pkg/server/ -v -count=1 2>&1 | tail -20`
Expected: All existing tests pass. The SDK considers any `< 300` status successful, so 201 works.

- [ ] **Step 7: Commit**

```bash
git add pkg/server/facts_tenants.go pkg/server/facts_users.go pkg/server/facts_instances.go pkg/server/facts_tuples.go pkg/server/facts_role_assignments.go
git commit -m "fix: return 201 Created for fact creation endpoints"
```

---

### Task 3: Resource Actions CRUD

The SDK has a full `ResourceActions` API (List, Get, Create, Update, Delete). Currently actions are only managed as nested fields inside the Resource object. We need dedicated endpoints.

**Files:**
- Create: `pkg/server/schema_actions.go`
- Modify: `pkg/server/schema_resources.go:23-30`
- Modify: `pkg/store/resources.go`
- Modify: `pkg/store/types.go`

- [ ] **Step 1: Add ResourceActionRead type to types.go**

Append to `pkg/store/types.go`:

```go
type ResourceActionRead struct {
	ID             string    `json:"id"`
	Key            string    `json:"key"`
	Name           string    `json:"name"`
	Description    *string   `json:"description,omitempty"`
	PermissionName string    `json:"permission_name"`
	ResourceID     string    `json:"resource_id"`
	OrganizationID string    `json:"organization_id"`
	ProjectID      string    `json:"project_id"`
	EnvironmentID  string    `json:"environment_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Add action-level store methods to resources.go**

Append to `pkg/store/resources.go`:

```go
func (s *Store) GetResourceAction(resourceKey, actionKey string) (*ResourceActionRead, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res, ok := s.resources[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceKey)
	}
	action, ok := res.Actions[actionKey]
	if !ok {
		return nil, fmt.Errorf("action %q not found on resource %q", actionKey, resourceKey)
	}
	return &ResourceActionRead{
		ID:             action.ID,
		Key:            actionKey,
		Name:           action.Name,
		Description:    action.Description,
		PermissionName: resourceKey + ":" + actionKey,
		ResourceID:     res.ID,
		OrganizationID: MockOrgID,
		ProjectID:      MockProjID,
		EnvironmentID:  MockEnvID,
		CreatedAt:      res.CreatedAt,
		UpdatedAt:      res.UpdatedAt,
	}, nil
}

func (s *Store) ListResourceActions(resourceKey string) ([]*ResourceActionRead, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res, ok := s.resources[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceKey)
	}
	var actions []*ResourceActionRead
	for k, v := range res.Actions {
		actions = append(actions, &ResourceActionRead{
			ID:             v.ID,
			Key:            k,
			Name:           v.Name,
			Description:    v.Description,
			PermissionName: resourceKey + ":" + k,
			ResourceID:     res.ID,
			OrganizationID: MockOrgID,
			ProjectID:      MockProjID,
			EnvironmentID:  MockEnvID,
			CreatedAt:      res.CreatedAt,
			UpdatedAt:      res.UpdatedAt,
		})
	}
	return actions, nil
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
	now := time.Now().UTC()
	action := ActionBlock{
		ID:          generateID(),
		Name:        name,
		Description: description,
	}
	res.Actions[actionKey] = action
	res.UpdatedAt = now
	s.materializeUnlocked()
	return &ResourceActionRead{
		ID:             action.ID,
		Key:            actionKey,
		Name:           action.Name,
		Description:    action.Description,
		PermissionName: resourceKey + ":" + actionKey,
		ResourceID:     res.ID,
		OrganizationID: MockOrgID,
		ProjectID:      MockProjID,
		EnvironmentID:  MockEnvID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (s *Store) UpdateResourceAction(resourceKey, actionKey string, name *string, description *string) (*ResourceActionRead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, ok := s.resources[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceKey)
	}
	action, ok := res.Actions[actionKey]
	if !ok {
		return nil, fmt.Errorf("action %q not found on resource %q", actionKey, resourceKey)
	}
	now := time.Now().UTC()
	if name != nil {
		action.Name = *name
	}
	if description != nil {
		action.Description = description
	}
	res.Actions[actionKey] = action
	res.UpdatedAt = now
	return &ResourceActionRead{
		ID:             action.ID,
		Key:            actionKey,
		Name:           action.Name,
		Description:    action.Description,
		PermissionName: resourceKey + ":" + actionKey,
		ResourceID:     res.ID,
		OrganizationID: MockOrgID,
		ProjectID:      MockProjID,
		EnvironmentID:  MockEnvID,
		CreatedAt:      res.CreatedAt,
		UpdatedAt:      now,
	}, nil
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
	res.UpdatedAt = time.Now().UTC()
	s.materializeUnlocked()
	return nil
}
```

- [ ] **Step 3: Create schema_actions.go handler**

Create `pkg/server/schema_actions.go`:

```go
package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleResourceActions(w http.ResponseWriter, r *http.Request, resourceKey string, segs []string) {
	switch r.Method {
	case http.MethodPost:
		if len(segs) > 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Key         string  `json:"key"`
			Name        string  `json:"name"`
			Description *string `json:"description,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		action, err := s.store.CreateResourceAction(resourceKey, body.Key, body.Name, body.Description)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, action)

	case http.MethodGet:
		if len(segs) == 0 {
			actions, err := s.store.ListResourceActions(resourceKey)
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, actions)
		} else {
			action, err := s.store.GetResourceAction(resourceKey, segs[0])
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, action)
		}

	case http.MethodPatch:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		var body struct {
			Name        *string `json:"name,omitempty"`
			Description *string `json:"description,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		action, err := s.store.UpdateResourceAction(resourceKey, segs[0], body.Name, body.Description)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, action)

	case http.MethodDelete:
		if len(segs) == 0 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if err := s.store.DeleteResourceAction(resourceKey, segs[0]); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
```

- [ ] **Step 4: Route actions in schema_resources.go**

In `pkg/server/schema_resources.go`, add a case for `"actions"` in the `handleResources` sub-resource switch (around line 23-30). Change:

```go
	if len(segs) >= 2 {
		resourceKey := segs[0]
		subResource := segs[1]
		switch subResource {
		case "roles":
			s.handleResourceRoles(w, r, resourceKey, segs[2:])
			return
		case "relations":
			s.handleRelations(w, r, resourceKey, segs[2:])
			return
		}
	}
```

to:

```go
	if len(segs) >= 2 {
		resourceKey := segs[0]
		subResource := segs[1]
		switch subResource {
		case "roles":
			s.handleResourceRoles(w, r, resourceKey, segs[2:])
			return
		case "relations":
			s.handleRelations(w, r, resourceKey, segs[2:])
			return
		case "actions":
			s.handleResourceActions(w, r, resourceKey, segs[2:])
			return
		}
	}
```

- [ ] **Step 5: Verify it compiles and existing tests pass**

Run: `cd /home/jarrod/.local/dev/permitio && go build ./... && go test ./pkg/server/ -v -count=1 2>&1 | tail -20`
Expected: Compiles, all tests pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/store/types.go pkg/store/resources.go pkg/server/schema_actions.go pkg/server/schema_resources.go
git commit -m "feat: add resource actions CRUD endpoints"
```

---

### Task 4: User GetAssignedRoles Endpoint

SDK method: `Users.GetAssignedRoles(ctx, userKey, tenantKey, page, perPage)` → `[]RoleAssignmentRead`

**Files:**
- Modify: `pkg/store/role_assignments.go`
- Modify: `pkg/server/facts_users.go:121-178`

- [ ] **Step 1: Add ListRoleAssignmentsForUser to store**

Append to `pkg/store/role_assignments.go`:

```go
func (s *Store) ListRoleAssignmentsForUser(userKey, tenantKey string) []RoleAssignment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []RoleAssignment
	for _, ra := range s.roleAssignments {
		if ra.User != userKey {
			continue
		}
		if tenantKey != "" && ra.Tenant != tenantKey {
			continue
		}
		result = append(result, ra)
	}
	return result
}
```

- [ ] **Step 2: Add GET handler to handleUserRoles**

In `pkg/server/facts_users.go`, in the `handleUserRoles` function, add a `MethodGet` case. Change:

```go
func (s *Server) handleUserRoles(w http.ResponseWriter, r *http.Request, userKey string) {
	switch r.Method {
	case http.MethodPost:
```

to:

```go
func (s *Server) handleUserRoles(w http.ResponseWriter, r *http.Request, userKey string) {
	switch r.Method {
	case http.MethodGet:
		tenant := r.URL.Query().Get("tenant")
		assignments := s.store.ListRoleAssignmentsForUser(userKey, tenant)
		if assignments == nil {
			assignments = []store.RoleAssignment{}
		}
		writeJSON(w, http.StatusOK, assignments)

	case http.MethodPost:
```

Note: the import for `"github.com/46labs/permitio/pkg/store"` is already in the file since `store.MockOrgID` etc. are used in `handleUserRoles`.

- [ ] **Step 3: Verify it compiles and tests pass**

Run: `cd /home/jarrod/.local/dev/permitio && go build ./... && go test ./pkg/server/ -v -count=1 2>&1 | tail -20`
Expected: Compiles, all tests pass.

- [ ] **Step 4: Commit**

```bash
git add pkg/store/role_assignments.go pkg/server/facts_users.go
git commit -m "feat: add GET /users/{key}/roles for GetAssignedRoles"
```

---

### Task 5: Tenant ListTenantUsers Endpoint

SDK method: `Tenants.ListTenantUsers(ctx, tenantKey, page, perPage)` → `[]UserRead`

**Files:**
- Modify: `pkg/store/tenants.go`
- Modify: `pkg/server/facts_tenants.go`

- [ ] **Step 1: Add ListTenantUsers to store**

Append to `pkg/store/tenants.go`:

```go
func (s *Store) ListTenantUsers(tenantKey string) []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find all users who have at least one role assignment in this tenant
	userKeys := make(map[string]bool)
	for _, ra := range s.roleAssignments {
		if ra.Tenant == tenantKey {
			userKeys[ra.User] = true
		}
	}

	var users []*User
	for key := range userKeys {
		if u, ok := s.users[key]; ok {
			users = append(users, u)
		}
	}
	return users
}
```

- [ ] **Step 2: Route tenant users in facts_tenants.go**

In `pkg/server/facts_tenants.go`, in `handleTenants`, we need to detect the `{key}/users` sub-path. Add routing at the top of the function, before the `switch r.Method`. Change the start of the function from:

```go
func (s *Server) handleTenants(w http.ResponseWriter, r *http.Request, segs []string) {
	switch r.Method {
```

to:

```go
func (s *Server) handleTenants(w http.ResponseWriter, r *http.Request, segs []string) {
	// Route: tenants/{key}/users
	if len(segs) >= 2 && segs[1] == "users" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		users := s.store.ListTenantUsers(segs[0])
		if users == nil {
			users = []*store.User{}
		}
		writeJSON(w, http.StatusOK, users)
		return
	}

	switch r.Method {
```

Add the import for `store` at the top of the file:

```go
import (
	"encoding/json"
	"net/http"

	"github.com/46labs/permitio/pkg/store"
)
```

- [ ] **Step 3: Verify it compiles and tests pass**

Run: `cd /home/jarrod/.local/dev/permitio && go build ./... && go test ./pkg/server/ -v -count=1 2>&1 | tail -20`
Expected: Compiles, all tests pass.

- [ ] **Step 4: Commit**

```bash
git add pkg/store/tenants.go pkg/server/facts_tenants.go
git commit -m "feat: add GET /tenants/{key}/users for ListTenantUsers"
```

---

### Task 6: Role Parent Relationships

SDK methods: `Roles.AddParentRole(ctx, roleKey, parentRoleKey)` and `Roles.RemoveParentRole(ctx, roleKey, parentRoleKey)`

**Files:**
- Modify: `pkg/store/roles.go`
- Modify: `pkg/server/schema_roles.go`

- [ ] **Step 1: Add parent role store methods**

Append to `pkg/store/roles.go`:

```go
func (s *Store) AddParentRole(key, parentKey string) (*Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	role, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	if _, ok := s.roles[parentKey]; !ok {
		return nil, fmt.Errorf("parent role %q not found", parentKey)
	}
	// Check if already a parent
	for _, e := range role.Extends {
		if e == parentKey {
			return role, nil
		}
	}
	role.Extends = append(role.Extends, parentKey)
	role.UpdatedAt = time.Now().UTC()
	s.materializeUnlocked()
	return role, nil
}

func (s *Store) RemoveParentRole(key, parentKey string) (*Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	role, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	filtered := make([]string, 0, len(role.Extends))
	for _, e := range role.Extends {
		if e != parentKey {
			filtered = append(filtered, e)
		}
	}
	role.Extends = filtered
	role.UpdatedAt = time.Now().UTC()
	s.materializeUnlocked()
	return role, nil
}
```

- [ ] **Step 2: Route parents in schema_roles.go**

In `pkg/server/schema_roles.go`, the current routing checks `segs[1] == "permissions"`. We need to also handle `"parents"`. Change the top of `handleRoles` from:

```go
func (s *Server) handleRoles(w http.ResponseWriter, r *http.Request, segs []string) {
	// segs: [] = list/create, [key] = get/update/delete
	// [key, "permissions"] = assign/remove permissions
	if len(segs) >= 2 && segs[1] == "permissions" {
		s.handleRolePermissions(w, r, segs[0])
		return
	}
```

to:

```go
func (s *Server) handleRoles(w http.ResponseWriter, r *http.Request, segs []string) {
	// segs: [] = list/create, [key] = get/update/delete
	// [key, "permissions"] = assign/remove permissions
	// [key, "parents", parentKey] = add/remove parent role
	if len(segs) >= 2 {
		switch segs[1] {
		case "permissions":
			s.handleRolePermissions(w, r, segs[0])
			return
		case "parents":
			if len(segs) < 3 {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			s.handleRoleParents(w, r, segs[0], segs[2])
			return
		}
	}
```

- [ ] **Step 3: Add handleRoleParents function to schema_roles.go**

Append to `pkg/server/schema_roles.go`:

```go
func (s *Server) handleRoleParents(w http.ResponseWriter, r *http.Request, roleKey, parentKey string) {
	switch r.Method {
	case http.MethodPut:
		role, err := s.store.AddParentRole(roleKey, parentKey)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, role)

	case http.MethodDelete:
		_, err := s.store.RemoveParentRole(roleKey, parentKey)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
```

- [ ] **Step 4: Verify it compiles and tests pass**

Run: `cd /home/jarrod/.local/dev/permitio && go build ./... && go test ./pkg/server/ -v -count=1 2>&1 | tail -20`
Expected: Compiles, all tests pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/store/roles.go pkg/server/schema_roles.go
git commit -m "feat: add role parent relationships (PUT/DELETE parents)"
```

---

### Task 7: ResourceRole Parent Relationships

SDK methods: `ResourceRoles.AddParent(ctx, resourceId, roleId, parentRoleId)` and `ResourceRoles.RemoveParent(ctx, resourceId, roleId, parentRoleId)`

**Files:**
- Modify: `pkg/store/resource_roles.go`
- Modify: `pkg/server/schema_resources.go`

- [ ] **Step 1: Add parent store methods to resource_roles.go**

Append to `pkg/store/resource_roles.go`:

```go
func (s *Store) AddResourceRoleParent(resourceKey, roleKey, parentKey string) (*ResourceRole, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roles, ok := s.resourceRoles[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceKey)
	}
	role, ok := roles[roleKey]
	if !ok {
		return nil, fmt.Errorf("role %q not found on resource %q", roleKey, resourceKey)
	}
	for _, e := range role.Extends {
		if e == parentKey {
			return role, nil
		}
	}
	role.Extends = append(role.Extends, parentKey)
	role.UpdatedAt = time.Now().UTC()
	s.materializeUnlocked()
	return role, nil
}

func (s *Store) RemoveResourceRoleParent(resourceKey, roleKey, parentKey string) (*ResourceRole, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roles, ok := s.resourceRoles[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", resourceKey)
	}
	role, ok := roles[roleKey]
	if !ok {
		return nil, fmt.Errorf("role %q not found on resource %q", roleKey, resourceKey)
	}
	filtered := make([]string, 0, len(role.Extends))
	for _, e := range role.Extends {
		if e != parentKey {
			filtered = append(filtered, e)
		}
	}
	role.Extends = filtered
	role.UpdatedAt = time.Now().UTC()
	s.materializeUnlocked()
	return role, nil
}
```

- [ ] **Step 2: Route parents in handleResourceRoles**

In `pkg/server/schema_resources.go`, in `handleResourceRoles` (line 166+), the existing sub-resource switch handles `"permissions"` and `"implicit_grants"`. Add `"parents"`. Change:

```go
	if len(segs) >= 2 {
		switch segs[1] {
		case "permissions":
			s.handleResourceRolePermissions(w, r, resourceKey, segs[0])
			return
		case "implicit_grants":
			s.handleImplicitGrants(w, r, resourceKey, segs[0])
			return
		}
	}
```

to:

```go
	if len(segs) >= 2 {
		switch segs[1] {
		case "permissions":
			s.handleResourceRolePermissions(w, r, resourceKey, segs[0])
			return
		case "implicit_grants":
			s.handleImplicitGrants(w, r, resourceKey, segs[0])
			return
		case "parents":
			if len(segs) < 3 {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			s.handleResourceRoleParents(w, r, resourceKey, segs[0], segs[2])
			return
		}
	}
```

- [ ] **Step 3: Add handleResourceRoleParents to schema_resources.go**

Append to `pkg/server/schema_resources.go`:

```go
func (s *Server) handleResourceRoleParents(w http.ResponseWriter, r *http.Request, resourceKey, roleKey, parentKey string) {
	switch r.Method {
	case http.MethodPut:
		role, err := s.store.AddResourceRoleParent(resourceKey, roleKey, parentKey)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, role)

	case http.MethodDelete:
		role, err := s.store.RemoveResourceRoleParent(resourceKey, roleKey, parentKey)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, role)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
```

- [ ] **Step 4: Verify it compiles and tests pass**

Run: `cd /home/jarrod/.local/dev/permitio && go build ./... && go test ./pkg/server/ -v -count=1 2>&1 | tail -20`
Expected: Compiles, all tests pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/store/resource_roles.go pkg/server/schema_resources.go
git commit -m "feat: add resource role parent relationships"
```

---

### Task 8: Functional Bulk Role Assignments

Replace the stubs with real bulk create/delete logic returning proper report responses.

**Files:**
- Modify: `pkg/server/facts_role_assignments.go`

- [ ] **Step 1: Replace bulk handler implementations**

Replace the entire `pkg/server/facts_role_assignments.go` with:

```go
package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleRoleAssignments(w http.ResponseWriter, r *http.Request, segs []string) {
	isBulk := len(segs) > 0 && segs[0] == "bulk"

	switch r.Method {
	case http.MethodPost:
		if isBulk {
			s.handleBulkRoleAssignmentCreate(w, r)
			return
		}
		var body struct {
			User   string `json:"user"`
			Role   string `json:"role"`
			Tenant string `json:"tenant"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ra, err := s.store.CreateRoleAssignment(body.User, body.Role, body.Tenant)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, ra)

	case http.MethodGet:
		if isBulk {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		assignments := s.store.ListRoleAssignments()
		writeJSON(w, http.StatusOK, assignments)

	case http.MethodDelete:
		if isBulk {
			s.handleBulkRoleAssignmentDelete(w, r)
			return
		}
		var body struct {
			User   string `json:"user"`
			Role   string `json:"role"`
			Tenant string `json:"tenant"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.store.DeleteRoleAssignment(body.User, body.Role, body.Tenant); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleBulkRoleAssignmentCreate(w http.ResponseWriter, r *http.Request) {
	var assignments []struct {
		User   string `json:"user"`
		Role   string `json:"role"`
		Tenant string `json:"tenant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&assignments); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	created := 0
	for _, a := range assignments {
		if _, err := s.store.CreateRoleAssignment(a.User, a.Role, a.Tenant); err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"assignments_created": created,
	})
}

func (s *Server) handleBulkRoleAssignmentDelete(w http.ResponseWriter, r *http.Request) {
	var unassignments []struct {
		User   string `json:"user"`
		Role   string `json:"role"`
		Tenant string `json:"tenant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&unassignments); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	removed := 0
	for _, u := range unassignments {
		if err := s.store.DeleteRoleAssignment(u.User, u.Role, u.Tenant); err == nil {
			removed++
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"assignments_removed": removed,
	})
}
```

- [ ] **Step 2: Verify it compiles and tests pass**

Run: `cd /home/jarrod/.local/dev/permitio && go build ./... && go test ./pkg/server/ -v -count=1 2>&1 | tail -20`
Expected: Compiles, all tests pass.

- [ ] **Step 3: Commit**

```bash
git add pkg/server/facts_role_assignments.go
git commit -m "feat: functional bulk role assignment create/delete with reports"
```

---

### Task 9: SDK Compatibility Test Suite

Comprehensive tests exercising every SDK sub-client method against the mock server. All tests use the official Go SDK exclusively — no raw HTTP calls.

**Files:**
- Create: `pkg/server/sdk_compat_test.go`

- [ ] **Step 1: Create sdk_compat_test.go with all management API tests**

Create `pkg/server/sdk_compat_test.go`:

```go
package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

// ---------- Resource Actions ----------

func TestResourceActionsCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup: create a resource with initial actions
	actions := map[string]models.ActionBlockEditable{"read": {}, "write": {}}
	rc := *models.NewResourceCreate("document", "Document", actions)
	_, err := env.client.Api.Resources.Create(ctx, rc)
	if err != nil {
		t.Fatalf("Create resource: %v", err)
	}

	// List actions
	actionList, err := env.client.Api.ResourceActions.List(ctx, "document", 1, 10)
	if err != nil {
		t.Fatalf("List actions: %v", err)
	}
	if len(actionList) < 2 {
		t.Fatalf("expected at least 2 actions, got %d", len(actionList))
	}

	// Get action
	action, err := env.client.Api.ResourceActions.Get(ctx, "document", "read")
	if err != nil {
		t.Fatalf("Get action: %v", err)
	}
	if action.Key != "read" {
		t.Fatalf("expected action key 'read', got %q", action.Key)
	}
	if action.PermissionName != "document:read" {
		t.Fatalf("expected permission_name 'document:read', got %q", action.PermissionName)
	}

	// Create new action
	ac := *models.NewResourceActionCreate("delete", "Delete")
	created, err := env.client.Api.ResourceActions.Create(ctx, "document", ac)
	if err != nil {
		t.Fatalf("Create action: %v", err)
	}
	if created.Key != "delete" {
		t.Fatalf("expected created action key 'delete', got %q", created.Key)
	}

	// Update action
	au := *models.NewResourceActionUpdate()
	newName := "Remove"
	au.SetName(newName)
	updated, err := env.client.Api.ResourceActions.Update(ctx, "document", "delete", au)
	if err != nil {
		t.Fatalf("Update action: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("expected name %q, got %q", newName, updated.Name)
	}

	// Delete action
	err = env.client.Api.ResourceActions.Delete(ctx, "document", "delete")
	if err != nil {
		t.Fatalf("Delete action: %v", err)
	}

	// Verify deleted
	_, err = env.client.Api.ResourceActions.Get(ctx, "document", "delete")
	if err == nil {
		t.Fatal("expected error getting deleted action")
	}
}

// ---------- User Assigned Roles ----------

func TestUserGetAssignedRoles(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup
	_, _ = env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("t1", "Tenant 1"))
	_, _ = env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("t2", "Tenant 2"))
	_, _ = env.client.Api.Roles.Create(ctx, *models.NewRoleCreate("admin", "Admin"))
	_, _ = env.client.Api.Roles.Create(ctx, *models.NewRoleCreate("viewer", "Viewer"))
	_, _ = env.client.Api.Users.Create(ctx, *models.NewUserCreate("u1"))

	// Assign roles in different tenants
	_, err := env.client.Api.Users.AssignRole(ctx, "u1", "admin", "t1")
	if err != nil {
		t.Fatalf("AssignRole admin/t1: %v", err)
	}
	_, err = env.client.Api.Users.AssignRole(ctx, "u1", "viewer", "t2")
	if err != nil {
		t.Fatalf("AssignRole viewer/t2: %v", err)
	}

	// GetAssignedRoles for all tenants
	roles, err := env.client.Api.Users.GetAssignedRoles(ctx, "u1", "", 1, 10)
	if err != nil {
		t.Fatalf("GetAssignedRoles: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("expected 2 assigned roles, got %d", len(roles))
	}

	// GetAssignedRoles filtered by tenant
	roles, err = env.client.Api.Users.GetAssignedRoles(ctx, "u1", "t1", 1, 10)
	if err != nil {
		t.Fatalf("GetAssignedRoles t1: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 assigned role for t1, got %d", len(roles))
	}
	if roles[0].Role != "admin" {
		t.Fatalf("expected role 'admin', got %q", roles[0].Role)
	}
}

// ---------- Tenant Users ----------

func TestTenantListUsers(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup
	_, _ = env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("t1", "Tenant 1"))
	_, _ = env.client.Api.Roles.Create(ctx, *models.NewRoleCreate("admin", "Admin"))
	_, _ = env.client.Api.Users.Create(ctx, *models.NewUserCreate("u1"))
	_, _ = env.client.Api.Users.Create(ctx, *models.NewUserCreate("u2"))
	_, _ = env.client.Api.Users.Create(ctx, *models.NewUserCreate("u3"))

	// Assign u1 and u2 to t1, u3 has no assignment
	_, _ = env.client.Api.Users.AssignRole(ctx, "u1", "admin", "t1")
	_, _ = env.client.Api.Users.AssignRole(ctx, "u2", "admin", "t1")

	// ListTenantUsers
	users, err := env.client.Api.Tenants.ListTenantUsers(ctx, "t1", 1, 10)
	if err != nil {
		t.Fatalf("ListTenantUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users in t1, got %d", len(users))
	}
	// Verify u3 is not included
	for _, u := range users {
		if u.Key == "u3" {
			t.Fatal("u3 should not be in tenant t1 users")
		}
	}
}

// ---------- Role Parents ----------

func TestRoleParentRelationships(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create roles
	_, _ = env.client.Api.Roles.Create(ctx, *models.NewRoleCreate("viewer", "Viewer"))
	_, _ = env.client.Api.Roles.Create(ctx, *models.NewRoleCreate("editor", "Editor"))

	// Add parent
	err := env.client.Api.Roles.AddParentRole(ctx, "editor", "viewer")
	if err != nil {
		t.Fatalf("AddParentRole: %v", err)
	}

	// Verify parent appears in extends
	role, err := env.client.Api.Roles.Get(ctx, "editor")
	if err != nil {
		t.Fatalf("Get role: %v", err)
	}
	found := false
	for _, ext := range role.Extends {
		if ext == "viewer" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'viewer' in extends, got %v", role.Extends)
	}

	// Remove parent
	err = env.client.Api.Roles.RemoveParentRole(ctx, "editor", "viewer")
	if err != nil {
		t.Fatalf("RemoveParentRole: %v", err)
	}

	// Verify removed
	role, err = env.client.Api.Roles.Get(ctx, "editor")
	if err != nil {
		t.Fatalf("Get role after remove: %v", err)
	}
	if len(role.Extends) != 0 {
		t.Fatalf("expected empty extends, got %v", role.Extends)
	}
}

// ---------- ResourceRole Parents ----------

func TestResourceRoleParentRelationships(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup resource with two roles
	actions := map[string]models.ActionBlockEditable{"read": {}, "write": {}}
	_, _ = env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("doc", "Document", actions))
	_, _ = env.client.Api.ResourceRoles.Create(ctx, "doc", *models.NewResourceRoleCreate("viewer", "Viewer"))
	_, _ = env.client.Api.ResourceRoles.Create(ctx, "doc", *models.NewResourceRoleCreate("editor", "Editor"))

	// Add parent
	result, err := env.client.Api.ResourceRoles.AddParent(ctx, "doc", "editor", "viewer")
	if err != nil {
		t.Fatalf("AddParent: %v", err)
	}
	found := false
	for _, ext := range result.Extends {
		if ext == "viewer" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'viewer' in extends after AddParent, got %v", result.Extends)
	}

	// Remove parent
	result, err = env.client.Api.ResourceRoles.RemoveParent(ctx, "doc", "editor", "viewer")
	if err != nil {
		t.Fatalf("RemoveParent: %v", err)
	}
	if len(result.Extends) != 0 {
		t.Fatalf("expected empty extends after RemoveParent, got %v", result.Extends)
	}
}

// ---------- Bulk Role Assignments ----------

func TestBulkRoleAssignments(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup
	_, _ = env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("t1", "Tenant 1"))
	_, _ = env.client.Api.Roles.Create(ctx, *models.NewRoleCreate("admin", "Admin"))
	_, _ = env.client.Api.Roles.Create(ctx, *models.NewRoleCreate("viewer", "Viewer"))
	_, _ = env.client.Api.Users.Create(ctx, *models.NewUserCreate("u1"))
	_, _ = env.client.Api.Users.Create(ctx, *models.NewUserCreate("u2"))

	// Bulk assign
	assignments := []models.RoleAssignmentCreate{
		*models.NewRoleAssignmentCreate("admin", "t1", "u1"),
		*models.NewRoleAssignmentCreate("viewer", "t1", "u2"),
	}
	report, err := env.client.Api.Roles.BulkAssignRole(ctx, assignments)
	if err != nil {
		t.Fatalf("BulkAssignRole: %v", err)
	}
	if report.AssignmentsCreated != 2 {
		t.Fatalf("expected 2 assignments_created, got %d", report.AssignmentsCreated)
	}

	// Verify via list
	raList, err := env.client.Api.RoleAssignments.List(ctx, 1, 10, "", "", "")
	if err != nil {
		t.Fatalf("List role assignments: %v", err)
	}
	if raList == nil || len(*raList) != 2 {
		t.Fatalf("expected 2 role assignments in list")
	}

	// Bulk unassign
	unassignments := []models.RoleAssignmentRemove{
		*models.NewRoleAssignmentRemove("admin", "t1", "u1"),
		*models.NewRoleAssignmentRemove("viewer", "t1", "u2"),
	}
	unReport, err := env.client.Api.Roles.BulkUnAssignRole(ctx, unassignments)
	if err != nil {
		t.Fatalf("BulkUnAssignRole: %v", err)
	}
	if unReport.AssignmentsRemoved != 2 {
		t.Fatalf("expected 2 assignments_removed, got %d", unReport.AssignmentsRemoved)
	}

	// Verify empty
	raList, err = env.client.Api.RoleAssignments.List(ctx, 1, 10, "", "", "")
	if err != nil {
		t.Fatalf("List after bulk unassign: %v", err)
	}
	if raList != nil && len(*raList) != 0 {
		t.Fatalf("expected 0 role assignments, got %d", len(*raList))
	}
}

// ---------- Role Remove Permissions ----------

func TestRoleRemovePermissions(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create role with permissions
	rc := *models.NewRoleCreate("editor", "Editor")
	_, _ = env.client.Api.Roles.Create(ctx, rc)
	_ = env.client.Api.Roles.AssignPermissions(ctx, "editor", []string{"doc:read", "doc:write", "doc:delete"})

	// Remove some permissions
	err := env.client.Api.Roles.RemovePermissions(ctx, "editor", []string{"doc:delete"})
	if err != nil {
		t.Fatalf("RemovePermissions: %v", err)
	}

	// Verify
	role, _ := env.client.Api.Roles.Get(ctx, "editor")
	if len(role.Permissions) != 2 {
		t.Fatalf("expected 2 permissions after remove, got %d: %v", len(role.Permissions), role.Permissions)
	}
	for _, p := range role.Permissions {
		if p == "doc:delete" {
			t.Fatal("doc:delete should have been removed")
		}
	}
}

// ---------- ResourceRole Permissions ----------

func TestResourceRoleAssignRemovePermissions(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	actions := map[string]models.ActionBlockEditable{"read": {}, "write": {}, "delete": {}}
	_, _ = env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("doc", "Document", actions))
	_, _ = env.client.Api.ResourceRoles.Create(ctx, "doc", *models.NewResourceRoleCreate("editor", "Editor"))

	// Assign permissions
	addPerms := *models.NewAddRolePermissions([]string{"doc:read", "doc:write"})
	rr, err := env.client.Api.ResourceRoles.AssignPermissions(ctx, "doc", "editor", addPerms)
	if err != nil {
		t.Fatalf("AssignPermissions: %v", err)
	}
	if len(rr.Permissions) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(rr.Permissions))
	}

	// Remove permissions
	removePerms := *models.NewRemoveRolePermissions([]string{"doc:write"})
	rr, err = env.client.Api.ResourceRoles.RemovePermissions(ctx, "doc", "editor", removePerms)
	if err != nil {
		t.Fatalf("RemovePermissions: %v", err)
	}
	if len(rr.Permissions) != 1 || rr.Permissions[0] != "doc:read" {
		t.Fatalf("expected [doc:read], got %v", rr.Permissions)
	}
}

// ---------- Relationship Tuples Bulk ----------

func TestRelationshipTuplesBulkOperations(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup
	actions := map[string]models.ActionBlockEditable{"read": {}}
	_, _ = env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("folder", "Folder", actions))
	_, _ = env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("file", "File", actions))
	_, _ = env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("default", "Default"))
	_, _ = env.client.Api.ResourceRelations.Create(ctx, "file", *models.NewRelationCreate("parent", "Parent", "folder"))

	f1 := *models.NewResourceInstanceCreate("docs", "folder")
	f1.SetTenant("default")
	_, _ = env.client.Api.ResourceInstances.Create(ctx, f1)
	f2 := *models.NewResourceInstanceCreate("pics", "folder")
	f2.SetTenant("default")
	_, _ = env.client.Api.ResourceInstances.Create(ctx, f2)
	fi1 := *models.NewResourceInstanceCreate("readme", "file")
	fi1.SetTenant("default")
	_, _ = env.client.Api.ResourceInstances.Create(ctx, fi1)
	fi2 := *models.NewResourceInstanceCreate("photo", "file")
	fi2.SetTenant("default")
	_, _ = env.client.Api.ResourceInstances.Create(ctx, fi2)

	// Bulk create
	bulkCreate := *models.NewRelationshipTupleCreateBulkOperation([]models.RelationshipTupleCreate{
		*models.NewRelationshipTupleCreate("folder:docs", "parent", "file:readme"),
		*models.NewRelationshipTupleCreate("folder:pics", "parent", "file:photo"),
	})
	err := env.client.Api.RelationshipTuples.BulkCreate(ctx, bulkCreate)
	if err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}

	// Verify
	tuples, err := env.client.Api.RelationshipTuples.List(ctx, 1, 10, "", "", "", "")
	if err != nil {
		t.Fatalf("List tuples: %v", err)
	}
	if tuples == nil || len(*tuples) != 2 {
		t.Fatalf("expected 2 tuples, got %v", tuples)
	}

	// Bulk delete
	bulkDelete := *models.NewRelationshipTupleDeleteBulkOperation([]models.RelationshipTupleDelete{
		*models.NewRelationshipTupleDelete("folder:docs", "parent", "file:readme"),
		*models.NewRelationshipTupleDelete("folder:pics", "parent", "file:photo"),
	})
	err = env.client.Api.RelationshipTuples.BulkDelete(ctx, bulkDelete)
	if err != nil {
		t.Fatalf("BulkDelete: %v", err)
	}

	// Verify empty
	tuples, err = env.client.Api.RelationshipTuples.List(ctx, 1, 10, "", "", "", "")
	if err != nil {
		t.Fatalf("List after bulk delete: %v", err)
	}
	if tuples != nil && len(*tuples) != 0 {
		t.Fatalf("expected 0 tuples, got %d", len(*tuples))
	}
}

// ---------- Resource Instance Update ----------

func TestResourceInstanceUpdate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	actions := map[string]models.ActionBlockEditable{"read": {}}
	_, _ = env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("doc", "Doc", actions))
	_, _ = env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("default", "Default"))

	ic := *models.NewResourceInstanceCreate("report", "doc")
	ic.SetTenant("default")
	_, err := env.client.Api.ResourceInstances.Create(ctx, ic)
	if err != nil {
		t.Fatalf("Create instance: %v", err)
	}

	// Update
	iu := *models.NewResourceInstanceUpdate()
	iu.SetAttributes(map[string]interface{}{"status": "draft"})
	updated, err := env.client.Api.ResourceInstances.Update(ctx, "doc:report", iu)
	if err != nil {
		t.Fatalf("Update instance: %v", err)
	}
	if updated.Attributes["status"] != "draft" {
		t.Fatalf("expected status=draft, got %v", updated.Attributes)
	}
}

// ---------- Resource Update ----------

func TestResourceUpdate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	actions := map[string]models.ActionBlockEditable{"read": {}}
	_, _ = env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("doc", "Document", actions))

	ru := *models.NewResourceUpdate()
	newName := "Updated Document"
	ru.SetName(newName)
	updated, err := env.client.Api.Resources.Update(ctx, "doc", ru)
	if err != nil {
		t.Fatalf("Update resource: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("expected name %q, got %q", newName, updated.Name)
	}
}

// ---------- ResourceRole Update ----------

func TestResourceRoleUpdate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	actions := map[string]models.ActionBlockEditable{"read": {}, "write": {}}
	_, _ = env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("doc", "Document", actions))
	_, _ = env.client.Api.ResourceRoles.Create(ctx, "doc", *models.NewResourceRoleCreate("editor", "Editor"))

	rru := *models.NewResourceRoleUpdate()
	newName := "Senior Editor"
	rru.SetName(newName)
	updated, err := env.client.Api.ResourceRoles.Update(ctx, "doc", "editor", rru)
	if err != nil {
		t.Fatalf("Update resource role: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("expected name %q, got %q", newName, updated.Name)
	}
}
```

- [ ] **Step 2: Run the tests**

Run: `cd /home/jarrod/.local/dev/permitio && go test ./pkg/server/ -run "TestResourceActions|TestUserGetAssigned|TestTenantList|TestRoleParent|TestResourceRoleParent|TestBulkRole|TestRoleRemove|TestResourceRoleAssignRemove|TestRelationshipTuplesBulk|TestResourceInstanceUpdate|TestResourceUpdate|TestResourceRoleUpdate" -v -count=1`

Expected: All new tests pass. If any fail, debug and fix the handler/store code.

- [ ] **Step 3: Commit**

```bash
git add pkg/server/sdk_compat_test.go
git commit -m "test: comprehensive SDK compatibility test suite"
```

---

### Task 10: SDK Error Case Tests

Test that the mock returns proper HTTP error codes that the SDK can interpret.

**Files:**
- Create: `pkg/server/sdk_errors_test.go`

- [ ] **Step 1: Create sdk_errors_test.go**

Create `pkg/server/sdk_errors_test.go`:

```go
package server

import (
	"context"
	"strings"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

func TestErrorNotFoundUser(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.client.Api.Users.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
	if !strings.Contains(err.Error(), "404") && !strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Fatalf("expected 404/not found error, got: %v", err)
	}
}

func TestErrorNotFoundTenant(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.client.Api.Tenants.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent tenant")
	}
}

func TestErrorNotFoundResource(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.client.Api.Resources.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent resource")
	}
}

func TestErrorNotFoundRole(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.client.Api.Roles.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent role")
	}
}

func TestErrorNotFoundResourceInstance(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.client.Api.ResourceInstances.Get(ctx, "doc:nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent resource instance")
	}
}

func TestErrorConflictDuplicateUser(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, _ = env.client.Api.Users.Create(ctx, *models.NewUserCreate("u1"))

	_, err := env.client.Api.Users.Create(ctx, *models.NewUserCreate("u1"))
	if err == nil {
		t.Fatal("expected error for duplicate user")
	}
	if !strings.Contains(err.Error(), "409") && !strings.Contains(strings.ToLower(err.Error()), "conflict") {
		t.Fatalf("expected 409/conflict error, got: %v", err)
	}
}

func TestErrorConflictDuplicateTenant(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, _ = env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("t1", "Tenant 1"))

	_, err := env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("t1", "Tenant 1"))
	if err == nil {
		t.Fatal("expected error for duplicate tenant")
	}
}

func TestErrorConflictDuplicateResource(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	actions := map[string]models.ActionBlockEditable{"read": {}}
	_, _ = env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("doc", "Doc", actions))

	_, err := env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("doc", "Doc", actions))
	if err == nil {
		t.Fatal("expected error for duplicate resource")
	}
}

func TestErrorConflictDuplicateRole(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, _ = env.client.Api.Roles.Create(ctx, *models.NewRoleCreate("admin", "Admin"))

	_, err := env.client.Api.Roles.Create(ctx, *models.NewRoleCreate("admin", "Admin"))
	if err == nil {
		t.Fatal("expected error for duplicate role")
	}
}

func TestErrorDeleteNonexistentUser(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	err := env.client.Api.Users.Delete(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error deleting nonexistent user")
	}
}

func TestErrorDeleteNonexistentTenant(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	err := env.client.Api.Tenants.Delete(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error deleting nonexistent tenant")
	}
}
```

- [ ] **Step 2: Run the error tests**

Run: `cd /home/jarrod/.local/dev/permitio && go test ./pkg/server/ -run "TestError" -v -count=1`
Expected: All pass.

- [ ] **Step 3: Run full test suite**

Run: `cd /home/jarrod/.local/dev/permitio && go test ./pkg/server/ -v -count=1`
Expected: All tests pass — existing + new compat + new error tests.

- [ ] **Step 4: Commit**

```bash
git add pkg/server/sdk_errors_test.go
git commit -m "test: SDK error case tests for 404 and 409 responses"
```

---

### Task 11: Final CI Validation

Run the full CI pipeline to ensure everything is clean.

- [ ] **Step 1: Run gofmt check**

Run: `cd /home/jarrod/.local/dev/permitio && gofmt -l ./pkg/`
Expected: No output (all files formatted).

- [ ] **Step 2: Run go vet**

Run: `cd /home/jarrod/.local/dev/permitio && go vet ./...`
Expected: No issues.

- [ ] **Step 3: Run full test suite with race detector**

Run: `cd /home/jarrod/.local/dev/permitio && go test -race ./... -count=1`
Expected: All pass, no race conditions.

- [ ] **Step 4: Run golangci-lint if available**

Run: `cd /home/jarrod/.local/dev/permitio && golangci-lint run ./...`
Expected: No new issues.

- [ ] **Step 5: Fix any issues found and commit**

If any lint/format/vet issues were found, fix them and commit:

```bash
git add -A
git commit -m "fix: address lint and format issues"
```
