# SDK API Alignment — Design Spec

Align the Go mock server's management API with the official Permit Go SDK (v1.2.8) expectations. Every SDK method that exercises a CRUD endpoint should work against this mock and return the correct status codes and response shapes.

## Scope

**In scope:** Status code corrections, missing endpoints the SDK calls, functional bulk operations, CORS, comprehensive SDK-only tests.

**Out of scope:** Rate limiting, TLS (handled by k8s), graceful shutdown, log level config, pagination limits (mock returns all results), kebab-case URL aliases (SDK uses snake_case exclusively).

## 1. Status Code Fixes

Fact creation endpoints currently return `200 OK`. The real API returns `201 Created`. The SDK accepts both (success = `< 300`), but for production parity:

| Endpoint | Current | Target |
|----------|---------|--------|
| `POST /v2/facts/.../tenants` | 200 | **201** |
| `POST /v2/facts/.../users` | 200 | **201** |
| `POST /v2/facts/.../resource_instances` | 200 | **201** |
| `POST /v2/facts/.../relationship_tuples` | 200 | **201** |
| `POST /v2/facts/.../role_assignments` | 200 | **201** |

Schema endpoints (`POST /v2/schema/.../resources`, `roles`, etc.) stay at `200`, matching both Rails and the real API.

## 2. Missing Endpoints

### 2a. Resource Actions CRUD

The SDK has a full `ResourceActions` API (8 methods). The server has no dedicated actions endpoints — actions are embedded in the resource object. Need to add:

- `GET /v2/schema/{p}/{e}/resources/{key}/actions` — list actions for a resource
- `POST /v2/schema/{p}/{e}/resources/{key}/actions` — create action on a resource
- `GET /v2/schema/{p}/{e}/resources/{key}/actions/{actionKey}` — get action
- `PATCH /v2/schema/{p}/{e}/resources/{key}/actions/{actionKey}` — update action
- `DELETE /v2/schema/{p}/{e}/resources/{key}/actions/{actionKey}` — delete action

Response shape (`ResourceActionRead`):
```json
{
  "id": "uuid",
  "key": "action_key",
  "name": "Action Name",
  "description": null,
  "permission_name": "resource_key:action_key",
  "resource_id": "uuid",
  "organization_id": "uuid",
  "project_id": "uuid",
  "environment_id": "uuid",
  "created_at": "RFC3339",
  "updated_at": "RFC3339"
}
```

Implementation: Actions already live in the store as part of `Resource.Actions`. These endpoints are thin wrappers that read/write individual action entries and return them in the `ResourceActionRead` shape.

### 2b. User GetAssignedRoles

SDK method: `Users.GetAssignedRoles(ctx, userKey, tenantKey, page, perPage)` → `[]RoleAssignmentRead`

Endpoint: `GET /v2/facts/{p}/{e}/users/{key}/roles`

Currently `handleUserRoles` only handles POST and DELETE. Add GET to return filtered role assignments for that user. Query params: `tenant` (optional filter).

### 2c. Tenant ListTenantUsers

SDK method: `Tenants.ListTenantUsers(ctx, tenantKey, page, perPage)` → `[]UserRead`

Endpoint: `GET /v2/facts/{p}/{e}/tenants/{key}/users`

Returns users who have at least one role assignment in the given tenant.

### 2d. Role Parent Relationships

SDK methods:
- `Roles.AddParentRole(ctx, roleKey, parentRoleKey)` → error
- `Roles.RemoveParentRole(ctx, roleKey, parentRoleKey)` → error

Endpoints:
- `PUT /v2/schema/{p}/{e}/roles/{key}/parents/{parentKey}` — returns `RoleRead`
- `DELETE /v2/schema/{p}/{e}/roles/{key}/parents/{parentKey}` — returns 204

These manage the `extends` field on global roles. `AddParentRole` appends the parent key to the role's `extends` array. `RemoveParentRole` removes it.

### 2e. ResourceRole Parent Relationships

SDK methods:
- `ResourceRoles.AddParent(ctx, resourceId, roleId, parentRoleId)` → `*ResourceRoleRead`
- `ResourceRoles.RemoveParent(ctx, resourceId, roleId, parentRoleId)` → `*ResourceRoleRead`

Endpoints:
- `PUT /v2/schema/{p}/{e}/resources/{key}/roles/{role}/parents/{parent}` — returns `ResourceRoleRead`
- `DELETE /v2/schema/{p}/{e}/resources/{key}/roles/{role}/parents/{parent}` — returns `ResourceRoleRead`

Same semantics as global role parents but scoped to a resource role.

## 3. Functional Bulk Operations

### 3a. Bulk Role Assignment

Currently stubs. Make functional:

`POST /v2/facts/{p}/{e}/role_assignments/bulk`
- Request: array of `RoleAssignmentCreate` objects
- Process each, skip errors silently (matches Rails behavior)
- Response: `{"assignments_created": N}` (SDK expects `BulkRoleAssignmentReport`)

`DELETE /v2/facts/{p}/{e}/role_assignments/bulk`
- Request: array of `RoleAssignmentRemove` objects
- Process each, skip errors silently
- Response: `{"assignments_removed": N}` (SDK expects `BulkRoleUnAssignmentReport`)

### 3b. Bulk Relationship Tuple Verification

The existing bulk tuple endpoints exist and are partially implemented. Verify they work with the SDK's `BulkCreate` and `BulkDelete` methods, which return `error` (no body parsing). These should be fine as-is but need test coverage.

## 4. CORS Middleware

Add CORS middleware that:
- Allows all origins (`*`)
- Allows all methods: GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD
- Allows all headers
- Handles preflight OPTIONS requests with 204

This matches the Rails `Rack::Cors` configuration. Required for any browser-based client.

## 5. Response Format Alignment

### 5a. List Endpoints — Pagination vs Raw Array

The SDK expects different formats per endpoint. Current state and required state:

| Endpoint | SDK Expects | Current | Action |
|----------|-------------|---------|--------|
| ListUsers | `PaginatedResultUserRead` (`{data, total_count, page_count}`) | Paginated | No change |
| ListTenants | `[]TenantRead` (raw array) | Raw array | No change |
| ListResources | `[]ResourceRead` (raw array) | Raw array | No change |
| ListRoles | `[]RoleRead` (raw array) | Raw array | No change |
| ListResourceInstances | `[]ResourceInstanceRead` (raw array) | Raw array | No change |
| ListRelationshipTuples | `[]RelationshipTupleRead` (raw array) | Raw array | No change |
| ListRoleAssignments | `[]RoleAssignmentRead` (raw array) | Raw array | No change |
| ListResourceRoles | `[]ResourceRoleRead` (raw array) | Raw array | No change |
| ListRelations | Paginated (`{data, total_count, page_count}`) | Paginated | No change |

Current formats are already correct.

### 5b. Role Assignment Detailed Listing

SDK method: `RoleAssignments.ListDetailed(ctx, page, perPage, user, role, tenant)` → `*[]RoleAssignmentDetailedRead`

Sends `detailed=true` query parameter to `GET /role_assignments`. Response should include nested role/tenant/user detail objects. For the mock, return the same data but with additional detail fields populated.

## 6. Test Plan

All tests use the official `github.com/permitio/permit-golang` SDK exclusively. No raw HTTP calls. Tests live in `pkg/server/` as `_test.go` files.

### 6a. New Test File: `sdk_compat_test.go`

Comprehensive SDK compatibility tests organized by API sub-client:

**Users:**
- Create, Get, List, Update, Delete (existing, verify 201 status via SDK success)
- SyncUser (exists → update, not exists → create)
- GetAssignedRoles (new)
- AssignRole, UnassignRole (existing)
- AssignResourceRole, UnassignResourceRole (existing, verify resource_instance handling)

**Tenants:**
- Create, Get, List, Update, Delete
- ListTenantUsers (new)

**Resources:**
- Create, Get, List, Update, Delete

**Resource Actions:**
- Create, Get, List, Update, Delete (all new)

**Roles:**
- Create, Get, List, Update, Delete
- AssignPermissions, RemovePermissions
- BulkAssignRole, BulkUnAssignRole (new — functional)
- AddParentRole, RemoveParentRole (new)

**Resource Roles:**
- Create, Get, List, Update, Delete
- AssignPermissions, RemovePermissions
- AddParent, RemoveParent (new)

**Resource Relations:**
- Create, Get, List, Delete

**Resource Instances:**
- Create, Get, List, Update, Delete

**Relationship Tuples:**
- Create, List, Delete
- BulkCreate, BulkDelete (new test coverage)

**Role Assignments:**
- Create (via direct API), List, Delete
- BulkAssign, BulkUnassign (new — functional)

**Implicit Grants:**
- Create, Delete

### 6b. Error Case Tests: `sdk_errors_test.go`

**404 Not Found:**
- Get non-existent user, tenant, resource, role, instance
- Delete non-existent resources

**409 Conflict:**
- Create duplicate user, tenant, resource, role

**Verify error types:**
- SDK wraps errors as `PermitNotFoundError`, `PermitConflictError`, etc.
- Tests should assert the correct error type where the SDK exposes it

### 6c. Keep Existing Tests

`crud_test.go` and `check_test.go` remain. The new `sdk_compat_test.go` provides broader coverage but the existing tests serve as regression anchors.

## 7. Implementation Files

Changes by file:

| File | Changes |
|------|---------|
| `pkg/server/server.go` | Route resource actions, tenant users, role parents |
| `pkg/server/middleware.go` | Add CORS middleware |
| `pkg/server/facts_tenants.go` | 201 status, ListTenantUsers handler |
| `pkg/server/facts_users.go` | 201 status, GET handler for user roles |
| `pkg/server/facts_instances.go` | 201 status |
| `pkg/server/facts_tuples.go` | 201 status |
| `pkg/server/facts_role_assignments.go` | 201 status, functional bulk, detailed query param |
| `pkg/server/schema_resources.go` | Route actions sub-resource, role parents |
| `pkg/server/schema_roles.go` | Route parents sub-resource |
| `pkg/server/schema_actions.go` | New file — resource actions CRUD |
| `pkg/store/store.go` | Add store methods for new endpoints |
| `pkg/store/types.go` | Add types if needed for action reads |
| `pkg/server/sdk_compat_test.go` | New comprehensive SDK test suite |
| `pkg/server/sdk_errors_test.go` | New error case test suite |
