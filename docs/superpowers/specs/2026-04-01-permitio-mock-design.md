# Permit.io PDP + Management API Mock

Local-only mock of the Permit.io PDP and Management API for development and testing. Zero cloud connectivity. Follows the same patterns as the auth0 mock at `~/.local/dev/auth0`.

## What This Replaces

The official `permitio/pdp-v2` Docker container requires an initial cloud connection to `api.permit.io` to bootstrap. It cannot run fully offline from a cold start. This mock eliminates that dependency entirely — configuration loads from YAML files, the official Go SDK connects to it, and all ReBAC checks evaluate locally.

## Single Service, Two API Surfaces

One Go binary on one port (default 7766) serves both:

- **PDP API** — `POST /allowed`, `/allowed/bulk`, `/allowed/all-tenants`, `/user-permissions`
- **Management API** — `GET/POST/PATCH/DELETE /v2/schema/{proj}/{env}/...` and `/v2/facts/{proj}/{env}/...`
- **Bootstrap** — `GET /v2/api-key/scope` returns a canned `{project_id, environment_id, access_level}` response

The Go SDK is configured with both `WithPdpUrl` and `WithApiUrl` pointing at the same host.

## Package Structure

```
cmd/main.go              - bootstrap: load config, create server, start
pkg/config/
  config.go              - struct definitions for schema + data
  loader.go              - viper-based YAML loading from /config dir + cwd
pkg/store/
  store.go               - in-memory maps, CRUD operations, mutex
  materialize.go         - rebuild effectivePerms from current state
pkg/server/
  server.go              - Server struct, Handler() with mux registration
  middleware.go          - auth header passthrough (accept anything), logging
  schema.go              - handlers for /v2/schema/... endpoints
  facts.go               - handlers for /v2/facts/... endpoints
  check.go               - handlers for /allowed, /allowed/bulk, /allowed/all-tenants
  permissions.go         - handler for /user-permissions
  apikey.go              - handler for /v2/api-key/scope
```

## Configuration

Two YAML files, both optional. Loaded from `/config` directory (for container mount) or current working directory (for local dev).

### schema.yaml

Defines the policy model: resource types, actions, roles, relations, and implicit grants (role derivations).

```yaml
resources:
  - key: folder
    name: Folder
    actions:
      read: { name: Read }
      write: { name: Write }
      manage: { name: Manage }
    roles:
      - key: owner
        name: Owner
        permissions: [read, write, manage]
      - key: viewer
        name: Viewer
        permissions: [read]
    relations:
      - key: parent
        name: Parent
        subject_resource: document

  - key: document
    name: Document
    actions:
      read: { name: Read }
      edit: { name: Edit }
      delete: { name: Delete }
    roles:
      - key: editor
        name: Editor
        permissions: [read, edit]
      - key: viewer
        name: Viewer
        permissions: [read]

roles:
  - key: admin
    name: Admin
    permissions: [folder:read, folder:write, folder:manage, document:read, document:edit, document:delete]

implicit_grants:
  - resource: folder
    role: owner
    on_resource: document
    derived_role: editor
    linked_by_relation: parent
```

### data.yaml

Pre-seeds runtime facts: tenants, users, resource instances, relationship tuples, role assignments.

```yaml
tenants:
  - key: default
    name: Default Tenant

users:
  - key: "auth0|user_devone"
    email: dev.one@nextel.test
    first_name: Dev
    last_name: One
  - key: "auth0|user_devtwo"
    email: dev.two@nextel.test
    first_name: Dev
    last_name: Two

resource_instances:
  - key: budget-2024
    resource: folder
    tenant: default
  - key: report-q4
    resource: document
    tenant: default

relationship_tuples:
  - subject: "folder:budget-2024"
    relation: parent
    object: "document:report-q4"

role_assignments:
  - user: "auth0|user_devone"
    role: owner
    resource_instance: "folder:budget-2024"
    tenant: default
```

## In-Memory Store

All state lives in maps behind a single `sync.RWMutex`. The store exposes typed CRUD methods that the HTTP handlers call.

### Core Data Structures

```go
type Store struct {
    mu sync.RWMutex

    // Schema
    resources      map[string]*Resource        // key -> resource
    roles          map[string]*Role            // key -> global role
    resourceRoles  map[string]map[string]*Role // resourceKey -> roleKey -> role
    relations      map[string]map[string]*Relation // resourceKey -> relationKey -> relation
    implicitGrants []ImplicitGrant

    // Facts
    tenants           map[string]*Tenant
    users             map[string]*User
    resourceInstances map[string]*ResourceInstance  // "type:key" -> instance
    relationshipTuples []RelationshipTuple
    roleAssignments   []RoleAssignment

    // Indexes (rebuilt on materialize)
    tupleIndex     map[string]map[string][]string   // "type:key" -> relation -> ["type:key", ...]
    effectivePerms map[string]bool                   // "user|action|type|key" -> true
    userPerms      map[string]map[string][]string    // user -> tenant -> [permissions...]
    userRoles      map[string]map[string][]string    // user -> tenant -> [roles...]
}
```

### Materialization

Called after every write operation. Rebuilds `effectivePerms` from scratch:

1. Build `tupleIndex` from all relationship tuples
2. For each role assignment:
   a. If it's a global role assignment (no resource instance), expand all permissions from the role
   b. If it's a resource-instance role assignment, expand permissions scoped to that instance
   c. For each implicit grant that matches this role+resource, follow tuples via the relation one hop, and add derived permissions on reached instances
3. Write results into `effectivePerms` map

The `POST /allowed` handler is then a single map lookup.

### Allow-All Mode

If `mode: allow_all` is set in schema.yaml, `POST /allowed` returns `{"allow": true}` unconditionally. Management API still works normally for integration testing. This is the fastest path to get the SDK wired into an API without writing any policy.

## Management API Endpoints

All endpoints accept any `Authorization: Bearer <token>` header without validation. The `{proj}` and `{env}` path segments are accepted but ignored (all data lives in a single namespace).

### Schema Endpoints (`/v2/schema/{proj}/{env}/...`)

**Resources:**
- `POST /resources` — create
- `GET /resources` — list
- `GET /resources/{id}` — get
- `PATCH /resources/{id}` — update
- `DELETE /resources/{id}` — delete

**Resource Roles:**
- `POST /resources/{id}/roles` — create
- `GET /resources/{id}/roles` — list
- `GET /resources/{id}/roles/{role}` — get
- `PATCH /resources/{id}/roles/{role}` — update
- `DELETE /resources/{id}/roles/{role}` — delete
- `POST /resources/{id}/roles/{role}/permissions` — assign permissions
- `DELETE /resources/{id}/roles/{role}/permissions` — remove permissions

**Resource Relations:**
- `POST /resources/{id}/relations` — create
- `GET /resources/{id}/relations` — list
- `GET /resources/{id}/relations/{rel}` — get
- `DELETE /resources/{id}/relations/{rel}` — delete

**Implicit Grants:**
- `POST /resources/{id}/roles/{role}/implicit_grants` — create
- `DELETE /resources/{id}/roles/{role}/implicit_grants` — delete

**Global Roles:**
- `POST /roles` — create
- `GET /roles` — list
- `GET /roles/{id}` — get
- `PATCH /roles/{id}` — update
- `DELETE /roles/{id}` — delete
- `POST /roles/{id}/permissions` — assign permissions
- `DELETE /roles/{id}/permissions` — remove permissions

### Facts Endpoints (`/v2/facts/{proj}/{env}/...`)

**Users:**
- `POST /users` — create
- `GET /users` — list
- `GET /users/{id}` — get
- `PATCH /users/{id}` — update
- `PUT /users/{id}` — replace (used by SDK's SyncUser)
- `DELETE /users/{id}` — delete
- `POST /users/{id}/roles` — assign role
- `DELETE /users/{id}/roles` — unassign role

**Tenants:**
- `POST /tenants` — create
- `GET /tenants` — list
- `GET /tenants/{id}` — get
- `PATCH /tenants/{id}` — update
- `DELETE /tenants/{id}` — delete

**Resource Instances:**
- `POST /resource_instances` — create
- `GET /resource_instances` — list
- `GET /resource_instances/{id}` — get
- `PATCH /resource_instances/{id}` — update
- `DELETE /resource_instances/{id}` — delete

**Relationship Tuples:**
- `POST /relationship_tuples` — create
- `GET /relationship_tuples` — list
- `DELETE /relationship_tuples` — delete
- `POST /relationship_tuples/bulk` — bulk create
- `DELETE /relationship_tuples/bulk` — bulk delete

**Role Assignments:**
- `POST /role_assignments` — assign
- `GET /role_assignments` — list
- `DELETE /role_assignments` — unassign
- `POST /role_assignments/bulk` — bulk assign
- `DELETE /role_assignments/bulk` — bulk unassign

## PDP Check Endpoints

**`POST /allowed`**

Request:
```json
{
  "user": {"key": "user-id"},
  "action": "edit",
  "resource": {"type": "document", "key": "doc-123", "tenant": "default"}
}
```

Response:
```json
{"allow": true, "result": true}
```

Implementation: `effectivePerms["user-id|edit|document|doc-123"]` lookup. If `mode: allow_all`, always true.

**`POST /allowed/bulk`**

Request: JSON array of authorization queries.
Response: `{"allow": [{...}, {...}]}` — array of results in same order.

**`POST /allowed/all-tenants`**

Request: same as `/allowed`.
Response: `{"allowed_tenants": [{tenant, allow, result}, ...]}` — checks across all tenants.

**`POST /user-permissions`**

Request:
```json
{"user": {"key": "user-id"}, "tenants": ["default"]}
```

Response: map keyed by `"__tenant:<tenant>"` with permissions and roles arrays.

## Testing Strategy

Tests import `github.com/permitio/permit-golang` and use `httptest.NewServer` to stand up the mock. The SDK is configured to point both API and PDP URLs at the test server.

### Test Categories

**1. Management API CRUD Tests**

For each resource type (resources, roles, resource roles, relations, instances, tuples, users, tenants, role assignments, implicit grants):
- Create via SDK → verify returned object
- Get via SDK → verify matches what was created
- List via SDK → verify appears in list
- Update via SDK (where applicable) → verify changes
- Delete via SDK → verify removed from list

**2. ReBAC Enforcement Tests**

Seed a full policy via SDK management calls, then test check outcomes:

- Direct role assignment: user has `owner` on `folder:budget-2024` → `Check(user, "read", folder:budget-2024)` = allow
- Derived permission via implicit grant: `folder:budget-2024 --parent--> document:report-q4`, folder owner derives document editor → `Check(user, "edit", document:report-q4)` = allow
- Negative case: user with no assignments → `Check(user, "edit", document:report-q4)` = deny
- Negative case: user has `viewer` role (read-only) → `Check(user, "edit", ...)` = deny
- Cross-tenant isolation: assignment in tenant A doesn't grant access in tenant B
- Bulk check: multiple queries in one call, verify mixed allow/deny results
- All-tenants check: verify which tenants grant access
- User permissions: verify returned permission list matches expected

**3. Allow-All Mode Test**

Configure mock with `mode: allow_all`, verify all checks return allow regardless of state.

## Infrastructure

### Dockerfile

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o permitio cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/permitio .
EXPOSE 7766
CMD ["./permitio"]
```

### justfile

- `just docker` — build + run on localhost:7766
- `just kind` — Kind cluster + Tilt
- `just ci` — format check + tests + lint
- `just down` — stop everything

### Helm Chart (`charts/permitio/`)

- ConfigMap with schema.yaml + data.yaml
- Deployment mounting config at `/config`
- Service (ClusterIP)
- Ingress (optional)
- Environment variable overrides for port, mode

### Tiltfile

- Docker build with live_update syncing pkg/ and cmd/
- Helm deploy to Kind cluster
- Health check hitting `POST /allowed` with a test payload

## Differences from Real Permit.io

1. **No cloud connectivity** — all state is local and in-memory
2. **No API key validation** — any bearer token accepted
3. **Single environment** — proj/env path segments are ignored
4. **No persistence** — state resets on restart (config files re-loaded)
5. **No OPAL/OPA** — materialization replaces the policy engine
6. **Simplified pagination** — list endpoints return all results
7. **No audit log** — no decision logging
