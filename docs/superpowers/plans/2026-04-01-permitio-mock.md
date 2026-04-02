# Permit.io Mock PDP + Management API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a local-only mock of the Permit.io PDP and Management API that passes all CRUD and ReBAC enforcement tests using the official `permit-golang` SDK.

**Architecture:** Single Go binary serves both PDP check endpoints (`POST /allowed`, etc.) and Management API endpoints (`/v2/schema/...`, `/v2/facts/...`) on port 7766. In-memory store with materialize-on-write for ReBAC evaluation. Config from YAML files.

**Tech Stack:** Go 1.24, `github.com/permitio/permit-golang` (official SDK, used in tests), `github.com/spf13/viper` (config loading), standard library `net/http` (server)

**Spec:** `docs/superpowers/specs/2026-04-01-permitio-mock-design.md`

---

### Task 1: Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `.golangci.yml`
- Create: `cmd/main.go`
- Create: `pkg/config/config.go`
- Create: `pkg/config/loader.go`

- [ ] **Step 1: Initialize Go module and install dependencies**

```bash
cd /home/jarrod/.local/dev/permitio
go mod init github.com/46labs/permitio
go get github.com/spf13/viper@latest
go get github.com/permitio/permit-golang@latest
```

- [ ] **Step 2: Create .gitignore**

```gitignore
bin/
*.exe
*.test
*.out
.env
dev/certs/
.claude/settings.local.json
```

- [ ] **Step 3: Create .golangci.yml**

```yaml
run:
  timeout: 5m
```

- [ ] **Step 4: Create config types (`pkg/config/config.go`)**

```go
package config

type ActionBlock struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty" mapstructure:"name"`
}

type ResourceRoleConfig struct {
	Key         string   `json:"key" yaml:"key" mapstructure:"key"`
	Name        string   `json:"name" yaml:"name" mapstructure:"name"`
	Permissions []string `json:"permissions,omitempty" yaml:"permissions,omitempty" mapstructure:"permissions"`
	Extends     []string `json:"extends,omitempty" yaml:"extends,omitempty" mapstructure:"extends"`
}

type RelationConfig struct {
	Key             string `json:"key" yaml:"key" mapstructure:"key"`
	Name            string `json:"name" yaml:"name" mapstructure:"name"`
	SubjectResource string `json:"subject_resource" yaml:"subject_resource" mapstructure:"subject_resource"`
}

type ResourceConfig struct {
	Key       string                  `json:"key" yaml:"key" mapstructure:"key"`
	Name      string                  `json:"name" yaml:"name" mapstructure:"name"`
	Actions   map[string]ActionBlock  `json:"actions" yaml:"actions" mapstructure:"actions"`
	Roles     []ResourceRoleConfig    `json:"roles,omitempty" yaml:"roles,omitempty" mapstructure:"roles"`
	Relations []RelationConfig        `json:"relations,omitempty" yaml:"relations,omitempty" mapstructure:"relations"`
}

type RoleConfig struct {
	Key         string   `json:"key" yaml:"key" mapstructure:"key"`
	Name        string   `json:"name" yaml:"name" mapstructure:"name"`
	Permissions []string `json:"permissions,omitempty" yaml:"permissions,omitempty" mapstructure:"permissions"`
}

type ImplicitGrantConfig struct {
	Resource         string `json:"resource" yaml:"resource" mapstructure:"resource"`
	Role             string `json:"role" yaml:"role" mapstructure:"role"`
	OnResource       string `json:"on_resource" yaml:"on_resource" mapstructure:"on_resource"`
	DerivedRole      string `json:"derived_role" yaml:"derived_role" mapstructure:"derived_role"`
	LinkedByRelation string `json:"linked_by_relation" yaml:"linked_by_relation" mapstructure:"linked_by_relation"`
}

type SchemaConfig struct {
	Mode           string                `json:"mode,omitempty" yaml:"mode,omitempty" mapstructure:"mode"`
	Resources      []ResourceConfig      `json:"resources,omitempty" yaml:"resources,omitempty" mapstructure:"resources"`
	Roles          []RoleConfig          `json:"roles,omitempty" yaml:"roles,omitempty" mapstructure:"roles"`
	ImplicitGrants []ImplicitGrantConfig `json:"implicit_grants,omitempty" yaml:"implicit_grants,omitempty" mapstructure:"implicit_grants"`
}

type UserConfig struct {
	Key       string `json:"key" yaml:"key" mapstructure:"key"`
	Email     string `json:"email,omitempty" yaml:"email,omitempty" mapstructure:"email"`
	FirstName string `json:"first_name,omitempty" yaml:"first_name,omitempty" mapstructure:"first_name"`
	LastName  string `json:"last_name,omitempty" yaml:"last_name,omitempty" mapstructure:"last_name"`
}

type TenantConfig struct {
	Key  string `json:"key" yaml:"key" mapstructure:"key"`
	Name string `json:"name" yaml:"name" mapstructure:"name"`
}

type ResourceInstanceConfig struct {
	Key      string `json:"key" yaml:"key" mapstructure:"key"`
	Resource string `json:"resource" yaml:"resource" mapstructure:"resource"`
	Tenant   string `json:"tenant,omitempty" yaml:"tenant,omitempty" mapstructure:"tenant"`
}

type RelationshipTupleConfig struct {
	Subject  string `json:"subject" yaml:"subject" mapstructure:"subject"`
	Relation string `json:"relation" yaml:"relation" mapstructure:"relation"`
	Object   string `json:"object" yaml:"object" mapstructure:"object"`
}

type RoleAssignmentConfig struct {
	User             string `json:"user" yaml:"user" mapstructure:"user"`
	Role             string `json:"role" yaml:"role" mapstructure:"role"`
	Tenant           string `json:"tenant" yaml:"tenant" mapstructure:"tenant"`
	ResourceInstance string `json:"resource_instance,omitempty" yaml:"resource_instance,omitempty" mapstructure:"resource_instance"`
}

type DataConfig struct {
	Tenants            []TenantConfig            `json:"tenants,omitempty" yaml:"tenants,omitempty" mapstructure:"tenants"`
	Users              []UserConfig              `json:"users,omitempty" yaml:"users,omitempty" mapstructure:"users"`
	ResourceInstances  []ResourceInstanceConfig  `json:"resource_instances,omitempty" yaml:"resource_instances,omitempty" mapstructure:"resource_instances"`
	RelationshipTuples []RelationshipTupleConfig `json:"relationship_tuples,omitempty" yaml:"relationship_tuples,omitempty" mapstructure:"relationship_tuples"`
	RoleAssignments    []RoleAssignmentConfig    `json:"role_assignments,omitempty" yaml:"role_assignments,omitempty" mapstructure:"role_assignments"`
}

type Config struct {
	Port   int          `json:"port" yaml:"port" mapstructure:"port"`
	Schema SchemaConfig `json:"schema" yaml:"schema" mapstructure:"schema"`
	Data   DataConfig   `json:"data" yaml:"data" mapstructure:"data"`
}
```

- [ ] **Step 5: Create config loader (`pkg/config/loader.go`)**

```go
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("port", 7766)
	viper.AutomaticEnv()
}

type Option func(*Config)

func Load(opts ...Option) (*Config, error) {
	// Load schema.yaml
	schemaV := viper.New()
	schemaV.SetConfigName("schema")
	schemaV.SetConfigType("yaml")
	schemaV.AddConfigPath("/config")
	schemaV.AddConfigPath(".")
	_ = schemaV.ReadInConfig()

	// Load data.yaml
	dataV := viper.New()
	dataV.SetConfigName("data")
	dataV.SetConfigType("yaml")
	dataV.AddConfigPath("/config")
	dataV.AddConfigPath(".")
	_ = dataV.ReadInConfig()

	cfg := &Config{
		Port: viper.GetInt("port"),
	}

	if err := schemaV.Unmarshal(&cfg.Schema); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}

	if err := dataV.Unmarshal(&cfg.Data); err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg, nil
}
```

- [ ] **Step 6: Create main entrypoint (`cmd/main.go`)**

```go
package main

import (
	"log"

	"github.com/46labs/permitio/pkg/config"
	"github.com/46labs/permitio/pkg/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	srv := server.New(cfg)
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 7: Verify it compiles**

Run: `go build ./...`
Expected: Build succeeds (server package doesn't exist yet, so create a stub first — see Task 2)

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "feat: project scaffold with config types and loader"
```

---

### Task 2: Store Data Types and Initialization

**Files:**
- Create: `pkg/store/types.go`
- Create: `pkg/store/store.go`

- [ ] **Step 1: Create store data types (`pkg/store/types.go`)**

These are the internal types that match the JSON the SDK sends/expects. They are separate from config types (which are for YAML loading).

```go
package store

import "time"

// Common fields for all "read" responses
type BaseFields struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ProjectID      string    `json:"project_id"`
	EnvironmentID  string    `json:"environment_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// --- Schema types ---

type ActionBlock struct {
	Name        string  `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	ID          string  `json:"id,omitempty"`
}

type Resource struct {
	BaseFields
	Key         string                 `json:"key"`
	Name        string                 `json:"name"`
	Urn         *string                `json:"urn,omitempty"`
	Description *string                `json:"description,omitempty"`
	Actions     map[string]ActionBlock `json:"actions,omitempty"`
}

type Role struct {
	BaseFields
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Permissions []string          `json:"permissions,omitempty"`
	Extends     []string          `json:"extends,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

type ResourceRole struct {
	BaseFields
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Extends     []string `json:"extends,omitempty"`
	ResourceID  string   `json:"resource_id"`
}

type Relation struct {
	BaseFields
	Key               string `json:"key"`
	Name              string `json:"name"`
	Description       *string `json:"description,omitempty"`
	SubjectResource   string `json:"subject_resource"`
	SubjectResourceID string `json:"subject_resource_id"`
	ObjectResourceID  string `json:"object_resource_id"`
	ObjectResource    string `json:"object_resource"`
}

type ImplicitGrant struct {
	RoleID           string `json:"role_id"`
	ResourceID       string `json:"resource_id"`
	RelationID       string `json:"relation_id"`
	Role             string `json:"role"`
	OnResource       string `json:"on_resource"`
	LinkedByRelation string `json:"linked_by_relation"`
}

// --- Facts types ---

type Tenant struct {
	BaseFields
	Key          string                 `json:"key"`
	Name         string                 `json:"name"`
	Description  *string                `json:"description,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	LastActionAt time.Time              `json:"last_action_at"`
}

type User struct {
	BaseFields
	Key        string                 `json:"key"`
	Email      *string                `json:"email,omitempty"`
	FirstName  *string                `json:"first_name,omitempty"`
	LastName   *string                `json:"last_name,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type ResourceInstance struct {
	BaseFields
	Key        string                 `json:"key"`
	Resource   string                 `json:"resource"`
	ResourceID string                 `json:"resource_id"`
	Tenant     *string                `json:"tenant,omitempty"`
	TenantID   *string                `json:"tenant_id,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type RelationshipTuple struct {
	BaseFields
	Subject    string `json:"subject"`
	Relation   string `json:"relation"`
	Object     string `json:"object"`
	Tenant     string `json:"tenant"`
	SubjectID  string `json:"subject_id"`
	RelationID string `json:"relation_id"`
	ObjectID   string `json:"object_id"`
	TenantID   string `json:"tenant_id"`
}

type RoleAssignment struct {
	ID               string    `json:"id"`
	User             string    `json:"user"`
	Role             string    `json:"role"`
	Tenant           string    `json:"tenant"`
	ResourceInstance *string   `json:"resource_instance,omitempty"`
	UserID           string    `json:"user_id"`
	RoleID           string    `json:"role_id"`
	TenantID         string    `json:"tenant_id"`
	OrganizationID   string    `json:"organization_id"`
	ProjectID        string    `json:"project_id"`
	EnvironmentID    string    `json:"environment_id"`
	CreatedAt        time.Time `json:"created_at"`
}
```

- [ ] **Step 2: Create store with initialization (`pkg/store/store.go`)**

```go
package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/46labs/permitio/pkg/config"
)

const (
	MockOrgID = "org_mock"
	MockProjID = "proj_mock"
	MockEnvID  = "env_mock"
)

type Store struct {
	mu sync.RWMutex

	// Schema
	resources      map[string]*Resource                  // key -> resource
	roles          map[string]*Role                      // key -> global role
	resourceRoles  map[string]map[string]*ResourceRole   // resourceKey -> roleKey -> role
	relations      map[string]map[string]*Relation       // resourceKey -> relationKey -> relation
	implicitGrants []ImplicitGrant

	// Facts
	tenants            map[string]*Tenant
	users              map[string]*User
	resourceInstances  map[string]*ResourceInstance // "type:key" -> instance
	relationshipTuples []RelationshipTuple
	roleAssignments    []RoleAssignment

	// Indexes (rebuilt on materialize)
	tupleIndex     map[string]map[string][]string // "type:key" -> relation -> ["type:key", ...]
	effectivePerms map[string]bool                // "user|action|type|key|tenant" -> true
	userPerms      map[string]map[string][]string // user -> tenant -> [permissions...]
	userRoles      map[string]map[string][]string // user -> tenant -> [roles...]

	// Config
	allowAll bool
}

func New() *Store {
	return &Store{
		resources:      make(map[string]*Resource),
		roles:          make(map[string]*Role),
		resourceRoles:  make(map[string]map[string]*ResourceRole),
		relations:      make(map[string]map[string]*Relation),
		tenants:        make(map[string]*Tenant),
		users:          make(map[string]*User),
		resourceInstances: make(map[string]*ResourceInstance),
		tupleIndex:     make(map[string]map[string][]string),
		effectivePerms: make(map[string]bool),
		userPerms:      make(map[string]map[string][]string),
		userRoles:      make(map[string]map[string][]string),
	}
}

func (s *Store) SetAllowAll(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowAll = v
}

func (s *Store) IsAllowAll() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.allowAll
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func newBase() BaseFields {
	now := time.Now().UTC()
	return BaseFields{
		ID:             generateID(),
		OrganizationID: MockOrgID,
		ProjectID:      MockProjID,
		EnvironmentID:  MockEnvID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// Seed loads initial data from config files
func (s *Store) Seed(cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cfg.Schema.Mode == "allow_all" {
		s.allowAll = true
	}

	for _, rc := range cfg.Schema.Resources {
		res := &Resource{
			BaseFields: newBase(),
			Key:        rc.Key,
			Name:       rc.Name,
			Actions:    make(map[string]ActionBlock),
		}
		for ak, av := range rc.Actions {
			res.Actions[ak] = ActionBlock{Name: av.Name, ID: generateID()}
		}
		s.resources[rc.Key] = res

		if len(rc.Roles) > 0 {
			if s.resourceRoles[rc.Key] == nil {
				s.resourceRoles[rc.Key] = make(map[string]*ResourceRole)
			}
			for _, rr := range rc.Roles {
				s.resourceRoles[rc.Key][rr.Key] = &ResourceRole{
					BaseFields:  newBase(),
					Key:         rr.Key,
					Name:        rr.Name,
					Permissions: rr.Permissions,
					Extends:     rr.Extends,
					ResourceID:  res.ID,
				}
			}
		}

		if len(rc.Relations) > 0 {
			if s.relations[rc.Key] == nil {
				s.relations[rc.Key] = make(map[string]*Relation)
			}
			for _, rel := range rc.Relations {
				s.relations[rc.Key][rel.Key] = &Relation{
					BaseFields:      newBase(),
					Key:             rel.Key,
					Name:            rel.Name,
					SubjectResource: rel.SubjectResource,
					ObjectResource:  rc.Key,
				}
			}
		}
	}

	for _, rc := range cfg.Schema.Roles {
		s.roles[rc.Key] = &Role{
			BaseFields:  newBase(),
			Key:         rc.Key,
			Name:        rc.Name,
			Permissions: rc.Permissions,
		}
	}

	for _, ig := range cfg.Schema.ImplicitGrants {
		s.implicitGrants = append(s.implicitGrants, ImplicitGrant{
			Role:             ig.Role,
			OnResource:       ig.OnResource,
			LinkedByRelation: ig.LinkedByRelation,
		})
	}

	for _, tc := range cfg.Data.Tenants {
		s.tenants[tc.Key] = &Tenant{
			BaseFields:   newBase(),
			Key:          tc.Key,
			Name:         tc.Name,
			LastActionAt: time.Now().UTC(),
		}
	}

	for _, uc := range cfg.Data.Users {
		u := &User{
			BaseFields: newBase(),
			Key:        uc.Key,
		}
		if uc.Email != "" {
			u.Email = &uc.Email
		}
		if uc.FirstName != "" {
			u.FirstName = &uc.FirstName
		}
		if uc.LastName != "" {
			u.LastName = &uc.LastName
		}
		s.users[uc.Key] = u
	}

	for _, ri := range cfg.Data.ResourceInstances {
		instKey := fmt.Sprintf("%s:%s", ri.Resource, ri.Key)
		s.resourceInstances[instKey] = &ResourceInstance{
			BaseFields: newBase(),
			Key:        ri.Key,
			Resource:   ri.Resource,
			Tenant:     &ri.Tenant,
		}
	}

	for _, rt := range cfg.Data.RelationshipTuples {
		s.relationshipTuples = append(s.relationshipTuples, RelationshipTuple{
			BaseFields: newBase(),
			Subject:    rt.Subject,
			Relation:   rt.Relation,
			Object:     rt.Object,
		})
	}

	for _, ra := range cfg.Data.RoleAssignments {
		assign := RoleAssignment{
			ID:             generateID(),
			User:           ra.User,
			Role:           ra.Role,
			Tenant:         ra.Tenant,
			OrganizationID: MockOrgID,
			ProjectID:      MockProjID,
			EnvironmentID:  MockEnvID,
			CreatedAt:      time.Now().UTC(),
		}
		if ra.ResourceInstance != "" {
			assign.ResourceInstance = &ra.ResourceInstance
		}
		s.roleAssignments = append(s.roleAssignments, assign)
	}
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./pkg/store/...`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: store data types and initialization"
```

---

### Task 3: Server Skeleton + API Key Scope + Test Helper

**Files:**
- Create: `pkg/server/server.go`
- Create: `pkg/server/middleware.go`
- Create: `pkg/server/apikey.go`
- Create: `pkg/server/helpers.go`
- Create: `pkg/server/testhelper_test.go`

- [ ] **Step 1: Create JSON response helpers (`pkg/server/helpers.go`)**

```go
package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]interface{}{
		"error":   http.StatusText(status),
		"message": msg,
		"status":  status,
	})
}

// extractPathSegments strips a prefix and splits the remaining path.
// e.g. "/v2/schema/proj/env/resources/doc" with prefix "/v2/schema" returns ["proj", "env", "resources", "doc"]
func extractPathSegments(path, prefix string) []string {
	trimmed := strings.TrimPrefix(path, prefix)
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}
```

- [ ] **Step 2: Create middleware (`pkg/server/middleware.go`)**

```go
package server

import (
	"log"
	"net/http"
	"time"
)

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
```

- [ ] **Step 3: Create API key scope handler (`pkg/server/apikey.go`)**

```go
package server

import (
	"net/http"

	"github.com/46labs/permitio/pkg/store"
)

func (s *Server) handleAPIKeyScope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": store.MockOrgID,
		"project_id":      store.MockProjID,
		"environment_id":  store.MockEnvID,
		"access_level":    "environment",
	})
}
```

- [ ] **Step 4: Create server with handler registration (`pkg/server/server.go`)**

```go
package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/46labs/permitio/pkg/config"
	"github.com/46labs/permitio/pkg/store"
)

type Server struct {
	cfg   *config.Config
	store *store.Store
}

func New(cfg *config.Config) *Server {
	st := store.New()
	st.Seed(cfg)
	st.Materialize()

	return &Server{
		cfg:   cfg,
		store: st,
	}
}

// NewWithStore creates a server with an existing store (for testing)
func NewWithStore(cfg *config.Config, st *store.Store) *Server {
	return &Server{
		cfg:   cfg,
		store: st,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API key scope (SDK calls this first to resolve project/env context)
	mux.HandleFunc("/v2/api-key/scope", s.handleAPIKeyScope)

	// Schema endpoints: /v2/schema/{proj}/{env}/...
	mux.HandleFunc("/v2/schema/", s.routeSchema)

	// Facts endpoints: /v2/facts/{proj}/{env}/...
	mux.HandleFunc("/v2/facts/", s.routeFacts)

	// PDP check endpoints
	mux.HandleFunc("/allowed", s.handleCheck)
	mux.HandleFunc("/allowed/bulk", s.handleBulkCheck)
	mux.HandleFunc("/allowed/all-tenants", s.handleAllTenantsCheck)
	mux.HandleFunc("/user-permissions", s.handleUserPermissions)

	return logMiddleware(mux)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	log.Printf("Starting permit.io mock on %s", addr)
	log.Printf("PDP API: POST /allowed")
	log.Printf("Management API: /v2/schema/... and /v2/facts/...")
	return http.ListenAndServe(addr, s.Handler())
}

// routeSchema handles all /v2/schema/{proj}/{env}/... requests
func (s *Server) routeSchema(w http.ResponseWriter, r *http.Request) {
	// Strip /v2/schema/{proj}/{env}/ prefix — segments: [proj, env, resource_type, ...]
	segs := extractPathSegments(r.URL.Path, "/v2/schema")
	if len(segs) < 3 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	// segs[0] = proj, segs[1] = env, segs[2:] = resource path
	rest := segs[2:]
	s.handleSchemaRoute(w, r, rest)
}

// routeFacts handles all /v2/facts/{proj}/{env}/... requests
func (s *Server) routeFacts(w http.ResponseWriter, r *http.Request) {
	segs := extractPathSegments(r.URL.Path, "/v2/facts")
	if len(segs) < 3 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	rest := segs[2:]
	s.handleFactsRoute(w, r, rest)
}

// Stub routers — these will be implemented in subsequent tasks
func (s *Server) handleSchemaRoute(w http.ResponseWriter, r *http.Request, segs []string) {
	writeError(w, http.StatusNotImplemented, "not implemented yet")
}

func (s *Server) handleFactsRoute(w http.ResponseWriter, r *http.Request, segs []string) {
	writeError(w, http.StatusNotImplemented, "not implemented yet")
}

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented yet")
}

func (s *Server) handleBulkCheck(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented yet")
}

func (s *Server) handleAllTenantsCheck(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented yet")
}

func (s *Server) handleUserPermissions(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented yet")
}
```

- [ ] **Step 5: Create materialization stub (`pkg/store/materialize.go`)**

```go
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
```

- [ ] **Step 6: Create test helper (`pkg/server/testhelper_test.go`)**

```go
package server

import (
	"net/http/httptest"
	"testing"

	"github.com/46labs/permitio/pkg/config"
	"github.com/46labs/permitio/pkg/store"
	permitConfig "github.com/permitio/permit-golang/pkg/config"
	"github.com/permitio/permit-golang/pkg/permit"
)

type testEnv struct {
	server *Server
	ts     *httptest.Server
	client *permit.Client
	store  *store.Store
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	cfg := &config.Config{Port: 0}
	st := store.New()
	srv := NewWithStore(cfg, st)
	ts := httptest.NewServer(srv.Handler())

	client := permit.NewPermit(
		permitConfig.NewConfigBuilder("test-api-key").
			WithPdpUrl(ts.URL).
			WithApiUrl(ts.URL).
			Build(),
	)

	t.Cleanup(func() { ts.Close() })

	return &testEnv{
		server: srv,
		ts:     ts,
		client: client,
		store:  st,
	}
}
```

- [ ] **Step 7: Verify it compiles and the test helper works**

Run: `go build ./... && go vet ./...`
Expected: Build succeeds

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "feat: server skeleton with routing, API key scope, and test helper"
```

---

### Task 4: Tenants CRUD

**Files:**
- Create: `pkg/store/tenants.go`
- Create: `pkg/server/facts_tenants.go`
- Create: `pkg/server/facts_tenants_test.go`

- [ ] **Step 1: Write the failing test (`pkg/server/facts_tenants_test.go`)**

```go
package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

func TestTenantsCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	var createdKey string

	t.Run("Create", func(t *testing.T) {
		tenant, err := env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("test-tenant", "Test Tenant"))
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if tenant.Key != "test-tenant" {
			t.Errorf("expected key test-tenant, got %s", tenant.Key)
		}
		if tenant.Name != "Test Tenant" {
			t.Errorf("expected name Test Tenant, got %s", tenant.Name)
		}
		if tenant.Id == "" {
			t.Error("expected non-empty ID")
		}
		createdKey = tenant.Key
	})

	t.Run("Get", func(t *testing.T) {
		tenant, err := env.client.Api.Tenants.Get(ctx, createdKey)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if tenant.Key != createdKey {
			t.Errorf("expected key %s, got %s", createdKey, tenant.Key)
		}
	})

	t.Run("List", func(t *testing.T) {
		tenants, err := env.client.Api.Tenants.List(ctx, 1, 100)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		found := false
		for _, ten := range tenants {
			if ten.Key == createdKey {
				found = true
				break
			}
		}
		if !found {
			t.Error("created tenant not found in list")
		}
	})

	t.Run("Update", func(t *testing.T) {
		update := models.TenantUpdate{Name: strPtr("Updated Tenant")}
		tenant, err := env.client.Api.Tenants.Update(ctx, createdKey, update)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if tenant.Name != "Updated Tenant" {
			t.Errorf("expected name Updated Tenant, got %s", tenant.Name)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := env.client.Api.Tenants.Delete(ctx, createdKey)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = env.client.Api.Tenants.Get(ctx, createdKey)
		if err == nil {
			t.Error("expected error getting deleted tenant")
		}
	})
}

func strPtr(s string) *string { return &s }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/server/ -run TestTenantsCRUD -v -count=1`
Expected: FAIL (endpoints return 501 Not Implemented)

- [ ] **Step 3: Implement store CRUD for tenants (`pkg/store/tenants.go`)**

```go
package store

import "fmt"

func (s *Store) CreateTenant(key, name string, desc *string, attrs map[string]interface{}) *Tenant {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := &Tenant{
		BaseFields: newBase(),
		Key:        key,
		Name:       name,
		Description: desc,
		Attributes: attrs,
		LastActionAt: newBase().CreatedAt,
	}
	s.tenants[key] = t
	return t
}

func (s *Store) GetTenant(key string) (*Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tenants[key]
	if !ok {
		return nil, fmt.Errorf("tenant %q not found", key)
	}
	return t, nil
}

func (s *Store) ListTenants() []*Tenant {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		result = append(result, t)
	}
	return result
}

func (s *Store) UpdateTenant(key string, name *string, desc *string, attrs map[string]interface{}) (*Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tenants[key]
	if !ok {
		return nil, fmt.Errorf("tenant %q not found", key)
	}
	if name != nil {
		t.Name = *name
	}
	if desc != nil {
		t.Description = desc
	}
	if attrs != nil {
		t.Attributes = attrs
	}
	t.UpdatedAt = newBase().CreatedAt
	return t, nil
}

func (s *Store) DeleteTenant(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tenants[key]; !ok {
		return fmt.Errorf("tenant %q not found", key)
	}
	delete(s.tenants, key)
	return nil
}
```

- [ ] **Step 4: Implement facts tenant handler (`pkg/server/facts_tenants.go`)**

```go
package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleTenants(w http.ResponseWriter, r *http.Request, segs []string) {
	// segs[0] = "tenants", segs[1:] = optional ID
	switch {
	case len(segs) == 1 && r.Method == http.MethodPost:
		s.createTenant(w, r)
	case len(segs) == 1 && r.Method == http.MethodGet:
		s.listTenants(w, r)
	case len(segs) == 2 && r.Method == http.MethodGet:
		s.getTenant(w, r, segs[1])
	case len(segs) == 2 && r.Method == http.MethodPatch:
		s.updateTenant(w, r, segs[1])
	case len(segs) == 2 && r.Method == http.MethodDelete:
		s.deleteTenant(w, r, segs[1])
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) createTenant(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key        string                 `json:"key"`
		Name       string                 `json:"name"`
		Description *string               `json:"description,omitempty"`
		Attributes map[string]interface{} `json:"attributes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	t := s.store.CreateTenant(body.Key, body.Name, body.Description, body.Attributes)
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) listTenants(w http.ResponseWriter, r *http.Request) {
	tenants := s.store.ListTenants()
	writeJSON(w, http.StatusOK, tenants)
}

func (s *Server) getTenant(w http.ResponseWriter, r *http.Request, key string) {
	t, err := s.store.GetTenant(key)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) updateTenant(w http.ResponseWriter, r *http.Request, key string) {
	var body struct {
		Name       *string                `json:"name,omitempty"`
		Description *string               `json:"description,omitempty"`
		Attributes map[string]interface{} `json:"attributes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	t, err := s.store.UpdateTenant(key, body.Name, body.Description, body.Attributes)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) deleteTenant(w http.ResponseWriter, r *http.Request, key string) {
	if err := s.store.DeleteTenant(key); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 5: Wire tenant routes into the facts router**

Update `handleFactsRoute` in `pkg/server/server.go`:

```go
func (s *Server) handleFactsRoute(w http.ResponseWriter, r *http.Request, segs []string) {
	switch segs[0] {
	case "tenants":
		s.handleTenants(w, r, segs)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./pkg/server/ -run TestTenantsCRUD -v -count=1`
Expected: PASS (all subtests pass). If the SDK expects a different list response format (e.g., paginated wrapper), fix the `listTenants` response to match and re-run.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat: tenants CRUD with SDK tests"
```

---

### Task 5: Users CRUD

**Files:**
- Create: `pkg/store/users.go`
- Create: `pkg/server/facts_users.go`
- Create: `pkg/server/facts_users_test.go`

- [ ] **Step 1: Write the failing test (`pkg/server/facts_users_test.go`)**

```go
package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

func TestUsersCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		uc := models.NewUserCreate("test-user")
		uc.SetEmail("test@example.com")
		uc.SetFirstName("Test")
		uc.SetLastName("User")

		user, err := env.client.Api.Users.Create(ctx, *uc)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if user.Key != "test-user" {
			t.Errorf("expected key test-user, got %s", user.Key)
		}
		if user.GetEmail() != "test@example.com" {
			t.Errorf("expected email test@example.com, got %s", user.GetEmail())
		}
	})

	t.Run("Get", func(t *testing.T) {
		user, err := env.client.Api.Users.Get(ctx, "test-user")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if user.Key != "test-user" {
			t.Errorf("expected key test-user, got %s", user.Key)
		}
	})

	t.Run("List", func(t *testing.T) {
		users, err := env.client.Api.Users.List(ctx, 1, 100)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		found := false
		for _, u := range users {
			if u.Key == "test-user" {
				found = true
				break
			}
		}
		if !found {
			t.Error("created user not found in list")
		}
	})

	t.Run("SyncUser", func(t *testing.T) {
		uc := models.NewUserCreate("sync-user")
		uc.SetEmail("sync@example.com")

		user, err := env.client.Api.Users.SyncUser(ctx, *uc)
		if err != nil {
			t.Fatalf("SyncUser failed: %v", err)
		}
		if user.Key != "sync-user" {
			t.Errorf("expected key sync-user, got %s", user.Key)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := env.client.Api.Users.Delete(ctx, "test-user")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = env.client.Api.Users.Get(ctx, "test-user")
		if err == nil {
			t.Error("expected error getting deleted user")
		}
	})

	t.Run("AssignRole", func(t *testing.T) {
		// Create prerequisites
		env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("default", "Default"))
		uc := models.NewUserCreate("role-user")
		env.client.Api.Users.Create(ctx, *uc)

		rc := models.NewRoleCreate("viewer", "Viewer")
		env.client.Api.Roles.Create(ctx, *rc)

		assignment, err := env.client.Api.Users.AssignRole(ctx, "role-user", "viewer", "default")
		if err != nil {
			t.Fatalf("AssignRole failed: %v", err)
		}
		if assignment.User != "role-user" {
			t.Errorf("expected user role-user, got %s", assignment.User)
		}
		if assignment.Role != "viewer" {
			t.Errorf("expected role viewer, got %s", assignment.Role)
		}
	})

	t.Run("UnassignRole", func(t *testing.T) {
		err := env.client.Api.Users.UnassignRole(ctx, "role-user", "viewer", "default")
		if err != nil {
			t.Fatalf("UnassignRole failed: %v", err)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/server/ -run TestUsersCRUD -v -count=1`
Expected: FAIL

- [ ] **Step 3: Implement store CRUD for users (`pkg/store/users.go`)**

```go
package store

import "fmt"

func (s *Store) CreateUser(key string, email, firstName, lastName *string, attrs map[string]interface{}) *User {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := &User{
		BaseFields: newBase(),
		Key:        key,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
		Attributes: attrs,
	}
	s.users[key] = u
	return u
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

func (s *Store) UpdateUser(key string, email, firstName, lastName *string, attrs map[string]interface{}) (*User, error) {
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
	if attrs != nil {
		u.Attributes = attrs
	}
	u.UpdatedAt = newBase().CreatedAt
	return u, nil
}

func (s *Store) UpsertUser(key string, email, firstName, lastName *string, attrs map[string]interface{}) *User {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u, ok := s.users[key]; ok {
		if email != nil {
			u.Email = email
		}
		if firstName != nil {
			u.FirstName = firstName
		}
		if lastName != nil {
			u.LastName = lastName
		}
		if attrs != nil {
			u.Attributes = attrs
		}
		u.UpdatedAt = newBase().CreatedAt
		return u
	}

	u := &User{
		BaseFields: newBase(),
		Key:        key,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
		Attributes: attrs,
	}
	s.users[key] = u
	return u
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
```

- [ ] **Step 4: Implement facts user handler (`pkg/server/facts_users.go`)**

```go
package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request, segs []string) {
	switch {
	case len(segs) == 1 && r.Method == http.MethodPost:
		s.createUser(w, r)
	case len(segs) == 1 && r.Method == http.MethodGet:
		s.listUsers(w, r)
	case len(segs) == 2 && r.Method == http.MethodGet:
		s.getUser(w, r, segs[1])
	case len(segs) == 2 && r.Method == http.MethodPatch:
		s.updateUser(w, r, segs[1])
	case len(segs) == 2 && r.Method == http.MethodPut:
		s.upsertUser(w, r, segs[1])
	case len(segs) == 2 && r.Method == http.MethodDelete:
		s.deleteUser(w, r, segs[1])
	case len(segs) == 3 && segs[2] == "roles" && r.Method == http.MethodPost:
		s.assignUserRole(w, r, segs[1])
	case len(segs) == 3 && segs[2] == "roles" && r.Method == http.MethodDelete:
		s.unassignUserRole(w, r, segs[1])
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key        string                 `json:"key"`
		Email      *string                `json:"email,omitempty"`
		FirstName  *string                `json:"first_name,omitempty"`
		LastName   *string                `json:"last_name,omitempty"`
		Attributes map[string]interface{} `json:"attributes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	u := s.store.CreateUser(body.Key, body.Email, body.FirstName, body.LastName, body.Attributes)
	writeJSON(w, http.StatusCreated, u)
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListUsers())
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request, key string) {
	u, err := s.store.GetUser(key)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request, key string) {
	var body struct {
		Email      *string                `json:"email,omitempty"`
		FirstName  *string                `json:"first_name,omitempty"`
		LastName   *string                `json:"last_name,omitempty"`
		Attributes map[string]interface{} `json:"attributes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	u, err := s.store.UpdateUser(key, body.Email, body.FirstName, body.LastName, body.Attributes)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) upsertUser(w http.ResponseWriter, r *http.Request, key string) {
	var body struct {
		Key        string                 `json:"key"`
		Email      *string                `json:"email,omitempty"`
		FirstName  *string                `json:"first_name,omitempty"`
		LastName   *string                `json:"last_name,omitempty"`
		Attributes map[string]interface{} `json:"attributes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	u := s.store.UpsertUser(key, body.Email, body.FirstName, body.LastName, body.Attributes)
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request, key string) {
	if err := s.store.DeleteUser(key); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) assignUserRole(w http.ResponseWriter, r *http.Request, userKey string) {
	var body struct {
		Role             string  `json:"role"`
		Tenant           string  `json:"tenant"`
		ResourceInstance *string `json:"resource_instance,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	assignment := s.store.CreateRoleAssignment(userKey, body.Role, body.Tenant, body.ResourceInstance)
	s.store.Materialize()
	writeJSON(w, http.StatusCreated, assignment)
}

func (s *Server) unassignUserRole(w http.ResponseWriter, r *http.Request, userKey string) {
	var body struct {
		Role             string  `json:"role"`
		Tenant           string  `json:"tenant"`
		ResourceInstance *string `json:"resource_instance,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := s.store.DeleteRoleAssignment(userKey, body.Role, body.Tenant, body.ResourceInstance); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	s.store.Materialize()
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 5: Wire users into facts router**

Update `handleFactsRoute` in `pkg/server/server.go`:

```go
func (s *Server) handleFactsRoute(w http.ResponseWriter, r *http.Request, segs []string) {
	switch segs[0] {
	case "tenants":
		s.handleTenants(w, r, segs)
	case "users":
		s.handleUsers(w, r, segs)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}
```

- [ ] **Step 6: Add role assignment store methods to `pkg/store/role_assignments.go`**

```go
package store

import "fmt"

func (s *Store) CreateRoleAssignment(user, role, tenant string, resourceInstance *string) *RoleAssignment {
	s.mu.Lock()
	defer s.mu.Unlock()

	a := RoleAssignment{
		ID:               generateID(),
		User:             user,
		Role:             role,
		Tenant:           tenant,
		ResourceInstance: resourceInstance,
		UserID:           user,
		RoleID:           role,
		TenantID:         tenant,
		OrganizationID:   MockOrgID,
		ProjectID:        MockProjID,
		EnvironmentID:    MockEnvID,
		CreatedAt:        newBase().CreatedAt,
	}
	s.roleAssignments = append(s.roleAssignments, a)
	return &a
}

func (s *Store) DeleteRoleAssignment(user, role, tenant string, resourceInstance *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, a := range s.roleAssignments {
		if a.User == user && a.Role == role && a.Tenant == tenant {
			riMatch := (resourceInstance == nil && a.ResourceInstance == nil) ||
				(resourceInstance != nil && a.ResourceInstance != nil && *resourceInstance == *a.ResourceInstance)
			if riMatch {
				s.roleAssignments = append(s.roleAssignments[:i], s.roleAssignments[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("role assignment not found")
}

func (s *Store) ListRoleAssignments() []RoleAssignment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]RoleAssignment, len(s.roleAssignments))
	copy(result, s.roleAssignments)
	return result
}
```

- [ ] **Step 7: Add stub for global roles so AssignRole test works — create `pkg/store/roles.go` and `pkg/server/schema_roles.go` with minimal Create/List/Get**

This will be fully implemented in Task 7, but we need a stub for `Roles.Create` so the AssignRole test can create the prerequisite role. Add a minimal `handleSchemaRoute` that routes to roles.

Update `handleSchemaRoute` in `pkg/server/server.go`:

```go
func (s *Server) handleSchemaRoute(w http.ResponseWriter, r *http.Request, segs []string) {
	switch segs[0] {
	case "roles":
		s.handleGlobalRoles(w, r, segs)
	default:
		writeError(w, http.StatusNotImplemented, "not implemented yet")
	}
}
```

Create `pkg/store/roles.go`:

```go
package store

import "fmt"

func (s *Store) CreateRole(key, name string, desc *string, perms []string, extends []string) *Role {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := &Role{
		BaseFields:  newBase(),
		Key:         key,
		Name:        name,
		Description: desc,
		Permissions: perms,
		Extends:     extends,
	}
	s.roles[key] = r
	return r
}

func (s *Store) GetRole(key string) (*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	return r, nil
}

func (s *Store) ListRoles() []*Role {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Role, 0, len(s.roles))
	for _, r := range s.roles {
		result = append(result, r)
	}
	return result
}

func (s *Store) UpdateRole(key string, name *string, desc *string, perms []string) (*Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	if name != nil {
		r.Name = *name
	}
	if desc != nil {
		r.Description = desc
	}
	if perms != nil {
		r.Permissions = perms
	}
	r.UpdatedAt = newBase().CreatedAt
	return r, nil
}

func (s *Store) DeleteRole(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.roles[key]; !ok {
		return fmt.Errorf("role %q not found", key)
	}
	delete(s.roles, key)
	return nil
}

func (s *Store) AssignRolePermissions(key string, perms []string) (*Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	existing := make(map[string]bool)
	for _, p := range r.Permissions {
		existing[p] = true
	}
	for _, p := range perms {
		if !existing[p] {
			r.Permissions = append(r.Permissions, p)
		}
	}
	return r, nil
}

func (s *Store) RemoveRolePermissions(key string, perms []string) (*Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.roles[key]
	if !ok {
		return nil, fmt.Errorf("role %q not found", key)
	}
	toRemove := make(map[string]bool)
	for _, p := range perms {
		toRemove[p] = true
	}
	filtered := make([]string, 0)
	for _, p := range r.Permissions {
		if !toRemove[p] {
			filtered = append(filtered, p)
		}
	}
	r.Permissions = filtered
	return r, nil
}
```

Create `pkg/server/schema_roles.go`:

```go
package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleGlobalRoles(w http.ResponseWriter, r *http.Request, segs []string) {
	switch {
	case len(segs) == 1 && r.Method == http.MethodPost:
		s.createGlobalRole(w, r)
	case len(segs) == 1 && r.Method == http.MethodGet:
		s.listGlobalRoles(w, r)
	case len(segs) == 2 && r.Method == http.MethodGet:
		s.getGlobalRole(w, r, segs[1])
	case len(segs) == 2 && r.Method == http.MethodPatch:
		s.updateGlobalRole(w, r, segs[1])
	case len(segs) == 2 && r.Method == http.MethodDelete:
		s.deleteGlobalRole(w, r, segs[1])
	case len(segs) == 3 && segs[2] == "permissions" && r.Method == http.MethodPost:
		s.assignGlobalRolePermissions(w, r, segs[1])
	case len(segs) == 3 && segs[2] == "permissions" && r.Method == http.MethodDelete:
		s.removeGlobalRolePermissions(w, r, segs[1])
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) createGlobalRole(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key         string   `json:"key"`
		Name        string   `json:"name"`
		Description *string  `json:"description,omitempty"`
		Permissions []string `json:"permissions,omitempty"`
		Extends     []string `json:"extends,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	role := s.store.CreateRole(body.Key, body.Name, body.Description, body.Permissions, body.Extends)
	writeJSON(w, http.StatusCreated, role)
}

func (s *Server) listGlobalRoles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListRoles())
}

func (s *Server) getGlobalRole(w http.ResponseWriter, r *http.Request, key string) {
	role, err := s.store.GetRole(key)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, role)
}

func (s *Server) updateGlobalRole(w http.ResponseWriter, r *http.Request, key string) {
	var body struct {
		Name        *string  `json:"name,omitempty"`
		Description *string  `json:"description,omitempty"`
		Permissions []string `json:"permissions,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	role, err := s.store.UpdateRole(key, body.Name, body.Description, body.Permissions)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, role)
}

func (s *Server) deleteGlobalRole(w http.ResponseWriter, r *http.Request, key string) {
	if err := s.store.DeleteRole(key); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) assignGlobalRolePermissions(w http.ResponseWriter, r *http.Request, key string) {
	var body struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	role, err := s.store.AssignRolePermissions(key, body.Permissions)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, role)
}

func (s *Server) removeGlobalRolePermissions(w http.ResponseWriter, r *http.Request, key string) {
	var body struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	role, err := s.store.RemoveRolePermissions(key, body.Permissions)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, role)
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./pkg/server/ -run TestUsersCRUD -v -count=1`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "feat: users CRUD with role assignment and SDK tests"
```

---

### Task 6: Resources CRUD

**Files:**
- Create: `pkg/store/resources.go`
- Create: `pkg/server/schema_resources.go`
- Create: `pkg/server/schema_resources_test.go`

- [ ] **Step 1: Write the failing test (`pkg/server/schema_resources_test.go`)**

```go
package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

func TestResourcesCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		actions := map[string]models.ActionBlockEditable{
			"read":  *models.NewActionBlockEditable(),
			"write": *models.NewActionBlockEditable(),
		}
		rc := models.NewResourceCreate("document", "Document", actions)

		resource, err := env.client.Api.Resources.Create(ctx, *rc)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if resource.Key != "document" {
			t.Errorf("expected key document, got %s", resource.Key)
		}
		if resource.Name != "Document" {
			t.Errorf("expected name Document, got %s", resource.Name)
		}
	})

	t.Run("Get", func(t *testing.T) {
		resource, err := env.client.Api.Resources.Get(ctx, "document")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if resource.Key != "document" {
			t.Errorf("expected key document, got %s", resource.Key)
		}
	})

	t.Run("List", func(t *testing.T) {
		resources, err := env.client.Api.Resources.List(ctx, 1, 100)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		found := false
		for _, res := range resources {
			if res.Key == "document" {
				found = true
				break
			}
		}
		if !found {
			t.Error("created resource not found in list")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := env.client.Api.Resources.Delete(ctx, "document")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = env.client.Api.Resources.Get(ctx, "document")
		if err == nil {
			t.Error("expected error getting deleted resource")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/server/ -run TestResourcesCRUD -v -count=1`
Expected: FAIL

- [ ] **Step 3: Implement store CRUD for resources (`pkg/store/resources.go`)**

```go
package store

import "fmt"

func (s *Store) CreateResource(key, name string, urn, desc *string, actions map[string]ActionBlock) *Resource {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := &Resource{
		BaseFields:  newBase(),
		Key:         key,
		Name:        name,
		Urn:         urn,
		Description: desc,
		Actions:     actions,
	}
	if r.Actions == nil {
		r.Actions = make(map[string]ActionBlock)
	}
	for k, a := range r.Actions {
		if a.ID == "" {
			a.ID = generateID()
			r.Actions[k] = a
		}
	}
	s.resources[key] = r

	// Initialize sub-maps
	if s.resourceRoles[key] == nil {
		s.resourceRoles[key] = make(map[string]*ResourceRole)
	}
	if s.relations[key] == nil {
		s.relations[key] = make(map[string]*Relation)
	}
	return r
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

func (s *Store) UpdateResource(key string, name *string, desc *string, actions map[string]ActionBlock) (*Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.resources[key]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", key)
	}
	if name != nil {
		r.Name = *name
	}
	if desc != nil {
		r.Description = desc
	}
	if actions != nil {
		r.Actions = actions
	}
	r.UpdatedAt = newBase().CreatedAt
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
```

- [ ] **Step 4: Implement resource handler (`pkg/server/schema_resources.go`)**

```go
package server

import (
	"encoding/json"
	"net/http"

	"github.com/46labs/permitio/pkg/store"
)

func (s *Server) handleResources(w http.ResponseWriter, r *http.Request, segs []string) {
	// segs: ["resources"], ["resources", "{id}"], ["resources", "{id}", "roles", ...], etc.
	switch {
	case len(segs) == 1 && r.Method == http.MethodPost:
		s.createResource(w, r)
	case len(segs) == 1 && r.Method == http.MethodGet:
		s.listResources(w, r)
	case len(segs) == 2 && r.Method == http.MethodGet:
		s.getResource(w, r, segs[1])
	case len(segs) == 2 && r.Method == http.MethodPatch:
		s.updateResource(w, r, segs[1])
	case len(segs) == 2 && r.Method == http.MethodDelete:
		s.deleteResource(w, r, segs[1])
	case len(segs) >= 3 && segs[2] == "roles":
		s.handleResourceRoles(w, r, segs[1], segs[2:])
	case len(segs) >= 3 && segs[2] == "relations":
		s.handleResourceRelations(w, r, segs[1], segs[2:])
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) createResource(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key         string                       `json:"key"`
		Name        string                       `json:"name"`
		Urn         *string                      `json:"urn,omitempty"`
		Description *string                      `json:"description,omitempty"`
		Actions     map[string]store.ActionBlock `json:"actions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	res := s.store.CreateResource(body.Key, body.Name, body.Urn, body.Description, body.Actions)
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) listResources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListResources())
}

func (s *Server) getResource(w http.ResponseWriter, r *http.Request, key string) {
	res, err := s.store.GetResource(key)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) updateResource(w http.ResponseWriter, r *http.Request, key string) {
	var body struct {
		Name        *string                      `json:"name,omitempty"`
		Description *string                      `json:"description,omitempty"`
		Actions     map[string]store.ActionBlock `json:"actions,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	res, err := s.store.UpdateResource(key, body.Name, body.Description, body.Actions)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) deleteResource(w http.ResponseWriter, r *http.Request, key string) {
	if err := s.store.DeleteResource(key); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Stubs for resource sub-routes — implemented in Tasks 8 and 9
func (s *Server) handleResourceRoles(w http.ResponseWriter, r *http.Request, resourceKey string, segs []string) {
	writeError(w, http.StatusNotImplemented, "not implemented yet")
}

func (s *Server) handleResourceRelations(w http.ResponseWriter, r *http.Request, resourceKey string, segs []string) {
	writeError(w, http.StatusNotImplemented, "not implemented yet")
}
```

- [ ] **Step 5: Wire resources into schema router**

Update `handleSchemaRoute` in `pkg/server/server.go`:

```go
func (s *Server) handleSchemaRoute(w http.ResponseWriter, r *http.Request, segs []string) {
	switch segs[0] {
	case "resources":
		s.handleResources(w, r, segs)
	case "roles":
		s.handleGlobalRoles(w, r, segs)
	default:
		writeError(w, http.StatusNotImplemented, "not implemented yet")
	}
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./pkg/server/ -run TestResourcesCRUD -v -count=1`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat: resources CRUD with SDK tests"
```

---

### Task 7: Global Roles CRUD Test

**Files:**
- Create: `pkg/server/schema_roles_test.go`

The roles store and handler were already created in Task 5 as a prerequisite for user role assignment. This task adds the SDK test.

- [ ] **Step 1: Write the test (`pkg/server/schema_roles_test.go`)**

```go
package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

func TestGlobalRolesCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		rc := models.NewRoleCreate("admin", "Admin")
		rc.SetPermissions([]string{"document:read", "document:write"})

		role, err := env.client.Api.Roles.Create(ctx, *rc)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if role.Key != "admin" {
			t.Errorf("expected key admin, got %s", role.Key)
		}
	})

	t.Run("Get", func(t *testing.T) {
		role, err := env.client.Api.Roles.Get(ctx, "admin")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if role.Key != "admin" {
			t.Errorf("expected key admin, got %s", role.Key)
		}
	})

	t.Run("List", func(t *testing.T) {
		roles, err := env.client.Api.Roles.List(ctx, 1, 100)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		found := false
		for _, r := range roles {
			if r.Key == "admin" {
				found = true
				break
			}
		}
		if !found {
			t.Error("created role not found in list")
		}
	})

	t.Run("AssignPermissions", func(t *testing.T) {
		role, err := env.client.Api.Roles.AssignPermissions(ctx, "admin", []string{"document:delete"})
		if err != nil {
			t.Fatalf("AssignPermissions failed: %v", err)
		}
		hasDelete := false
		for _, p := range role.Permissions {
			if p == "document:delete" {
				hasDelete = true
			}
		}
		if !hasDelete {
			t.Error("expected document:delete in permissions")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := env.client.Api.Roles.Delete(ctx, "admin")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})
}
```

- [ ] **Step 2: Run test**

Run: `go test ./pkg/server/ -run TestGlobalRolesCRUD -v -count=1`
Expected: PASS (implementation already exists from Task 5)

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "test: global roles CRUD SDK tests"
```

---

### Task 8: Resource Roles, Relations, and Implicit Grants

**Files:**
- Create: `pkg/store/resource_roles.go`
- Create: `pkg/store/relations.go`
- Create: `pkg/store/implicit_grants.go`
- Modify: `pkg/server/schema_resources.go` (fill in stubs)
- Create: `pkg/server/schema_resource_roles_test.go`

- [ ] **Step 1: Write the test (`pkg/server/schema_resource_roles_test.go`)**

```go
package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

func TestResourceRolesAndRelations(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create prerequisite resources
	actions := map[string]models.ActionBlockEditable{
		"read":  *models.NewActionBlockEditable(),
		"write": *models.NewActionBlockEditable(),
	}
	env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("folder", "Folder", actions))
	env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("document", "Document", actions))

	t.Run("CreateResourceRole", func(t *testing.T) {
		rr := models.NewResourceRoleCreate("editor", "Editor")
		rr.SetPermissions([]string{"read", "write"})

		role, err := env.client.Api.ResourceRoles.Create(ctx, "document", *rr)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if role.Key != "editor" {
			t.Errorf("expected key editor, got %s", role.Key)
		}
	})

	t.Run("GetResourceRole", func(t *testing.T) {
		role, err := env.client.Api.ResourceRoles.Get(ctx, "document", "editor")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if role.Key != "editor" {
			t.Errorf("expected key editor, got %s", role.Key)
		}
	})

	t.Run("ListResourceRoles", func(t *testing.T) {
		roles, err := env.client.Api.ResourceRoles.List(ctx, "document", 1, 100)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(roles) == 0 {
			t.Error("expected at least one resource role")
		}
	})

	t.Run("CreateRelation", func(t *testing.T) {
		rel := models.NewRelationCreate("parent", "Parent")
		rel.SetSubjectResource("document")

		relation, err := env.client.Api.ResourceRelations.Create(ctx, "folder", *rel)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if relation.Key != "parent" {
			t.Errorf("expected key parent, got %s", relation.Key)
		}
	})

	t.Run("ListRelations", func(t *testing.T) {
		relations, err := env.client.Api.ResourceRelations.List(ctx, "folder", 1, 100)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(relations) == 0 {
			t.Error("expected at least one relation")
		}
	})

	t.Run("CreateImplicitGrant", func(t *testing.T) {
		// First create a resource role on folder
		ownerRole := models.NewResourceRoleCreate("owner", "Owner")
		ownerRole.SetPermissions([]string{"read", "write"})
		env.client.Api.ResourceRoles.Create(ctx, "folder", *ownerRole)

		ig := models.NewDerivedRoleRuleCreate("editor", "document", "parent")

		err := env.client.Api.ImplicitGrants.Create(ctx, "folder", "owner", *ig)
		if err != nil {
			t.Fatalf("CreateImplicitGrant failed: %v", err)
		}
	})

	t.Run("DeleteResourceRole", func(t *testing.T) {
		err := env.client.Api.ResourceRoles.Delete(ctx, "document", "editor")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})

	t.Run("DeleteRelation", func(t *testing.T) {
		err := env.client.Api.ResourceRelations.Delete(ctx, "folder", "parent")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/server/ -run TestResourceRolesAndRelations -v -count=1`
Expected: FAIL

- [ ] **Step 3: Implement store methods for resource roles, relations, and implicit grants**

Create `pkg/store/resource_roles.go`, `pkg/store/relations.go`, `pkg/store/implicit_grants.go` with CRUD methods following the same pattern as `roles.go` and `tenants.go`. Key differences:

- Resource roles are keyed by `(resourceKey, roleKey)` using the nested `resourceRoles` map
- Relations are keyed by `(resourceKey, relationKey)` using the nested `relations` map
- Implicit grants are stored as a slice and matched by `(resourceKey, roleKey, derivedRole, onResource, relation)`

- [ ] **Step 4: Fill in the handler stubs in `pkg/server/schema_resources.go`**

Implement `handleResourceRoles` and `handleResourceRelations` with the same CRUD handler pattern used in tenants/users. Add an `handleImplicitGrants` for the `/resources/{id}/roles/{role}/implicit_grants` path.

- [ ] **Step 5: Run test**

Run: `go test ./pkg/server/ -run TestResourceRolesAndRelations -v -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: resource roles, relations, and implicit grants with SDK tests"
```

---

### Task 9: Resource Instances and Relationship Tuples

**Files:**
- Create: `pkg/store/resource_instances.go`
- Create: `pkg/store/relationship_tuples.go`
- Create: `pkg/server/facts_instances.go`
- Create: `pkg/server/facts_tuples.go`
- Create: `pkg/server/facts_instances_test.go`

- [ ] **Step 1: Write the test (`pkg/server/facts_instances_test.go`)**

```go
package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

func TestResourceInstancesAndTuples(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Prerequisites
	actions := map[string]models.ActionBlockEditable{"read": *models.NewActionBlockEditable()}
	env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("folder", "Folder", actions))
	env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("document", "Document", actions))
	env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("default", "Default"))

	t.Run("CreateInstance", func(t *testing.T) {
		ic := models.NewResourceInstanceCreate("budget-2024", "folder", "default")

		instance, err := env.client.Api.ResourceInstances.Create(ctx, *ic)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if instance.Key != "budget-2024" {
			t.Errorf("expected key budget-2024, got %s", instance.Key)
		}
		if instance.Resource != "folder" {
			t.Errorf("expected resource folder, got %s", instance.Resource)
		}
	})

	t.Run("GetInstance", func(t *testing.T) {
		instance, err := env.client.Api.ResourceInstances.Get(ctx, "folder:budget-2024")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if instance.Key != "budget-2024" {
			t.Errorf("expected key budget-2024, got %s", instance.Key)
		}
	})

	t.Run("ListInstances", func(t *testing.T) {
		instances, err := env.client.Api.ResourceInstances.List(ctx, 1, 100)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(instances) == 0 {
			t.Error("expected at least one instance")
		}
	})

	// Create second instance for tuple test
	env.client.Api.ResourceInstances.Create(ctx, *models.NewResourceInstanceCreate("report-q4", "document", "default"))

	// Create relation for tuple
	rel := models.NewRelationCreate("parent", "Parent")
	rel.SetSubjectResource("document")
	env.client.Api.ResourceRelations.Create(ctx, "folder", *rel)

	t.Run("CreateTuple", func(t *testing.T) {
		tc := models.NewRelationshipTupleCreate("folder:budget-2024", "parent", "document:report-q4")

		tuple, err := env.client.Api.RelationshipTuples.Create(ctx, *tc)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if tuple.Subject != "folder:budget-2024" {
			t.Errorf("expected subject folder:budget-2024, got %s", tuple.Subject)
		}
	})

	t.Run("ListTuples", func(t *testing.T) {
		tuples, err := env.client.Api.RelationshipTuples.List(ctx, 1, 100)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(tuples) == 0 {
			t.Error("expected at least one tuple")
		}
	})

	t.Run("DeleteTuple", func(t *testing.T) {
		err := env.client.Api.RelationshipTuples.Delete(ctx, *models.NewRelationshipTupleDelete("folder:budget-2024", "parent", "document:report-q4"))
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})

	t.Run("DeleteInstance", func(t *testing.T) {
		err := env.client.Api.ResourceInstances.Delete(ctx, "folder:budget-2024")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/server/ -run TestResourceInstancesAndTuples -v -count=1`
Expected: FAIL

- [ ] **Step 3: Implement store and handler for resource instances and relationship tuples**

Follow the same CRUD pattern. Key specifics:
- Resource instances are keyed by `"type:key"` (e.g., `"folder:budget-2024"`)
- The SDK's `Get` for instances passes `"folder:budget-2024"` as the ID
- Relationship tuples use body-matching for delete (subject + relation + object)
- Relationship tuple create triggers `s.store.Materialize()`

- [ ] **Step 4: Wire into facts router**

Add `"resource_instances"` and `"relationship_tuples"` cases to `handleFactsRoute`.

- [ ] **Step 5: Run test**

Run: `go test ./pkg/server/ -run TestResourceInstancesAndTuples -v -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: resource instances and relationship tuples with SDK tests"
```

---

### Task 10: Role Assignments CRUD Test

**Files:**
- Create: `pkg/server/facts_role_assignments.go`
- Create: `pkg/server/facts_role_assignments_test.go`

The role assignment store was created in Task 5. This task adds the standalone `/role_assignments` endpoint (separate from the user-scoped endpoint) and tests it.

- [ ] **Step 1: Write the test (`pkg/server/facts_role_assignments_test.go`)**

```go
package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

func TestRoleAssignmentsCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Prerequisites
	env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("default", "Default"))
	env.client.Api.Users.Create(ctx, *models.NewUserCreate("user-1"))
	env.client.Api.Roles.Create(ctx, *models.NewRoleCreate("viewer", "Viewer"))

	t.Run("Assign", func(t *testing.T) {
		assignment, err := env.client.Api.Users.AssignRole(ctx, "user-1", "viewer", "default")
		if err != nil {
			t.Fatalf("Assign failed: %v", err)
		}
		if assignment.User != "user-1" {
			t.Errorf("expected user user-1, got %s", assignment.User)
		}
	})

	t.Run("List", func(t *testing.T) {
		assignments, err := env.client.Api.RoleAssignments.List(ctx, 1, 100)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		found := false
		for _, a := range assignments {
			if a.User == "user-1" && a.Role == "viewer" {
				found = true
				break
			}
		}
		if !found {
			t.Error("assignment not found in list")
		}
	})

	t.Run("Unassign", func(t *testing.T) {
		err := env.client.Api.Users.UnassignRole(ctx, "user-1", "viewer", "default")
		if err != nil {
			t.Fatalf("Unassign failed: %v", err)
		}
	})
}
```

- [ ] **Step 2: Implement role assignments list endpoint (`pkg/server/facts_role_assignments.go`)**

```go
package server

import "net/http"

func (s *Server) handleRoleAssignments(w http.ResponseWriter, r *http.Request, segs []string) {
	switch {
	case len(segs) == 1 && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, s.store.ListRoleAssignments())
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}
```

- [ ] **Step 3: Wire into facts router**

Add `"role_assignments"` case to `handleFactsRoute`.

- [ ] **Step 4: Run test**

Run: `go test ./pkg/server/ -run TestRoleAssignmentsCRUD -v -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: role assignments list endpoint with SDK tests"
```

---

### Task 11: Run All Management API Tests

- [ ] **Step 1: Run all tests together**

Run: `go test ./pkg/server/ -v -count=1`
Expected: ALL PASS

- [ ] **Step 2: Fix any failures**

If any tests fail due to response format mismatches (pagination wrappers, field naming), fix the handlers to match what the SDK expects. Use the error messages from the SDK to diagnose.

Common issues to watch for:
- The SDK may expect list endpoints to return an array directly vs. a paginated wrapper
- The SDK may expect `201 Created` vs `200 OK` for create operations
- The SDK's `SyncUser` calls GET first then PUT or POST — ensure both paths work
- Field names must match the SDK's JSON tags exactly

- [ ] **Step 3: Commit any fixes**

```bash
git add -A
git commit -m "fix: adjust response formats to match SDK expectations"
```

---

### Task 12: Materialization Engine

**Files:**
- Modify: `pkg/store/materialize.go`
- Create: `pkg/store/materialize_test.go`

- [ ] **Step 1: Write the test (`pkg/store/materialize_test.go`)**

```go
package store

import "testing"

func TestMaterialize_DirectPermission(t *testing.T) {
	s := New()

	// Setup: resource "folder" with action "read", resource role "viewer" with "read" permission
	s.CreateResource("folder", "Folder", nil, nil, map[string]ActionBlock{
		"read": {Name: "Read", ID: generateID()},
	})
	s.CreateResourceRole("folder", "viewer", "Viewer", nil, []string{"read"}, nil)
	s.CreateTenant("default", "Default", nil, nil)
	s.CreateUser("user-1", nil, nil, nil, nil)
	instKey := "folder:budget"
	s.CreateResourceInstance("budget", "folder", strPtr("default"), nil)
	s.CreateRoleAssignment("user-1", "viewer", "default", &instKey)
	s.Materialize()

	if !s.CheckPermission("user-1", "read", "folder", "budget", "default") {
		t.Error("expected user-1 to have read on folder:budget")
	}
	if s.CheckPermission("user-1", "write", "folder", "budget", "default") {
		t.Error("expected user-1 NOT to have write on folder:budget")
	}
	if s.CheckPermission("user-2", "read", "folder", "budget", "default") {
		t.Error("expected user-2 NOT to have read on folder:budget")
	}
}

func TestMaterialize_DerivedPermission(t *testing.T) {
	s := New()

	// Setup: folder --parent--> document, folder#owner derives document#editor
	s.CreateResource("folder", "Folder", nil, nil, map[string]ActionBlock{
		"read":  {Name: "Read", ID: generateID()},
		"write": {Name: "Write", ID: generateID()},
	})
	s.CreateResource("document", "Document", nil, nil, map[string]ActionBlock{
		"read": {Name: "Read", ID: generateID()},
		"edit": {Name: "Edit", ID: generateID()},
	})
	s.CreateResourceRole("folder", "owner", "Owner", nil, []string{"read", "write"}, nil)
	s.CreateResourceRole("document", "editor", "Editor", nil, []string{"read", "edit"}, nil)
	s.CreateRelation("folder", "parent", "Parent", "document")
	s.CreateImplicitGrant("folder", "owner", "editor", "document", "parent")

	s.CreateTenant("default", "Default", nil, nil)
	s.CreateUser("user-1", nil, nil, nil, nil)
	s.CreateResourceInstance("budget", "folder", strPtr("default"), nil)
	s.CreateResourceInstance("report", "document", strPtr("default"), nil)

	// Create tuple: folder:budget --parent--> document:report
	s.CreateRelationshipTuple("folder:budget", "parent", "document:report", "default")

	// Assign user-1 as folder:budget owner
	folderInst := "folder:budget"
	s.CreateRoleAssignment("user-1", "owner", "default", &folderInst)

	s.Materialize()

	// Direct permission on folder
	if !s.CheckPermission("user-1", "read", "folder", "budget", "default") {
		t.Error("expected user-1 to have read on folder:budget")
	}

	// Derived permission on document via implicit grant
	if !s.CheckPermission("user-1", "edit", "document", "report", "default") {
		t.Error("expected user-1 to have edit on document:report (derived via folder owner -> document editor)")
	}
	if !s.CheckPermission("user-1", "read", "document", "report", "default") {
		t.Error("expected user-1 to have read on document:report (derived)")
	}

	// Negative: user-2 has no access
	if s.CheckPermission("user-2", "edit", "document", "report", "default") {
		t.Error("expected user-2 NOT to have edit on document:report")
	}
}

func TestMaterialize_CrossTenantIsolation(t *testing.T) {
	s := New()

	s.CreateResource("folder", "Folder", nil, nil, map[string]ActionBlock{
		"read": {Name: "Read", ID: generateID()},
	})
	s.CreateResourceRole("folder", "viewer", "Viewer", nil, []string{"read"}, nil)
	s.CreateTenant("tenant-a", "Tenant A", nil, nil)
	s.CreateTenant("tenant-b", "Tenant B", nil, nil)
	s.CreateUser("user-1", nil, nil, nil, nil)
	s.CreateResourceInstance("doc", "folder", strPtr("tenant-a"), nil)

	instKey := "folder:doc"
	s.CreateRoleAssignment("user-1", "viewer", "tenant-a", &instKey)
	s.Materialize()

	if !s.CheckPermission("user-1", "read", "folder", "doc", "tenant-a") {
		t.Error("expected access in tenant-a")
	}
	if s.CheckPermission("user-1", "read", "folder", "doc", "tenant-b") {
		t.Error("expected NO access in tenant-b")
	}
}

func strPtr(s string) *string { return &s }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/store/ -run TestMaterialize -v -count=1`
Expected: FAIL (materializeUnlocked is a stub)

- [ ] **Step 3: Implement materialization (`pkg/store/materialize.go`)**

```go
package store

import "fmt"

func (s *Store) Materialize() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.materializeUnlocked()
}

func (s *Store) materializeUnlocked() {
	// Reset indexes
	s.tupleIndex = make(map[string]map[string][]string)
	s.effectivePerms = make(map[string]bool)
	s.userPerms = make(map[string]map[string][]string)
	s.userRoles = make(map[string]map[string][]string)

	// 1. Build tuple index: subject -> relation -> [objects]
	for _, t := range s.relationshipTuples {
		if s.tupleIndex[t.Subject] == nil {
			s.tupleIndex[t.Subject] = make(map[string][]string)
		}
		s.tupleIndex[t.Subject][t.Relation] = append(s.tupleIndex[t.Subject][t.Relation], t.Object)
	}

	// 2. For each role assignment, expand permissions
	for _, ra := range s.roleAssignments {
		tenant := ra.Tenant

		if ra.ResourceInstance != nil {
			// Resource-instance role assignment (e.g., user has "owner" on "folder:budget")
			instKey := *ra.ResourceInstance // "folder:budget"
			resType, _ := splitInstanceKey(instKey)

			// Find resource role
			if roles, ok := s.resourceRoles[resType]; ok {
				if role, ok := roles[ra.Role]; ok {
					// Add direct permissions
					for _, perm := range role.Permissions {
						action := perm // permission is the action name for resource roles
						permKey := fmt.Sprintf("%s|%s|%s|%s|%s", ra.User, action, resType, instanceKeyPart(instKey), tenant)
						s.effectivePerms[permKey] = true
						s.addUserPerm(ra.User, tenant, fmt.Sprintf("%s:%s", resType, action))
					}
					s.addUserRole(ra.User, tenant, fmt.Sprintf("%s:%s", resType, ra.Role))

					// Check implicit grants for this role+resource
					for _, ig := range s.implicitGrants {
						if ig.Role == ra.Role && (ig.Resource == "" || ig.Resource == resType) {
							// Follow tuples from instKey via ig.LinkedByRelation
							if rels, ok := s.tupleIndex[instKey]; ok {
								if targets, ok := rels[ig.LinkedByRelation]; ok {
									// For each reached target, add derived role permissions
									if derivedRoles, ok := s.resourceRoles[ig.OnResource]; ok {
										if derivedRole, ok := derivedRoles[ig.DerivedRole]; ok {
											for _, target := range targets {
												targetRes, targetKey := splitInstanceKey(target)
												if targetRes == ig.OnResource {
													for _, perm := range derivedRole.Permissions {
														permKey := fmt.Sprintf("%s|%s|%s|%s|%s", ra.User, perm, targetRes, targetKey, tenant)
														s.effectivePerms[permKey] = true
														s.addUserPerm(ra.User, tenant, fmt.Sprintf("%s:%s", targetRes, perm))
													}
													s.addUserRole(ra.User, tenant, fmt.Sprintf("%s:%s", targetRes, ig.DerivedRole))
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		} else {
			// Global role assignment
			if role, ok := s.roles[ra.Role]; ok {
				for _, perm := range role.Permissions {
					// Global permissions apply to all instances of the resource type
					// Format: "resource:action"
					s.addUserPerm(ra.User, tenant, perm)
					s.addUserRole(ra.User, tenant, ra.Role)
				}
			}
		}
	}
}

// CheckPermission checks if user has permission. This is the hot path for /allowed.
func (s *Store) CheckPermission(user, action, resourceType, instanceKey, tenant string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.allowAll {
		return true
	}

	permKey := fmt.Sprintf("%s|%s|%s|%s|%s", user, action, resourceType, instanceKey, tenant)
	return s.effectivePerms[permKey]
}

func (s *Store) GetUserPermissions(user string, tenants []string) map[string]UserPermissionSet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]UserPermissionSet)
	if perms, ok := s.userPerms[user]; ok {
		for tenant, permList := range perms {
			if len(tenants) > 0 && !contains(tenants, tenant) {
				continue
			}
			key := fmt.Sprintf("__tenant:%s", tenant)
			roles := s.userRoles[user][tenant]
			result[key] = UserPermissionSet{
				Tenant:      TenantRef{Key: tenant},
				Permissions: unique(permList),
				Roles:       unique(roles),
			}
		}
	}
	return result
}

type TenantRef struct {
	Key        string                 `json:"key"`
	Attributes map[string]interface{} `json:"attributes"`
}

type UserPermissionSet struct {
	Tenant      TenantRef `json:"tenant"`
	Permissions []string  `json:"permissions"`
	Roles       []string  `json:"roles"`
}

func (s *Store) addUserPerm(user, tenant, perm string) {
	if s.userPerms[user] == nil {
		s.userPerms[user] = make(map[string][]string)
	}
	s.userPerms[user][tenant] = append(s.userPerms[user][tenant], perm)
}

func (s *Store) addUserRole(user, tenant, role string) {
	if s.userRoles[user] == nil {
		s.userRoles[user] = make(map[string][]string)
	}
	s.userRoles[user][tenant] = append(s.userRoles[user][tenant], role)
}

func splitInstanceKey(key string) (string, string) {
	for i, c := range key {
		if c == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

func instanceKeyPart(key string) string {
	_, k := splitInstanceKey(key)
	return k
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func unique(items []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/store/ -run TestMaterialize -v -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: materialization engine with direct and derived permission tests"
```

---

### Task 13: PDP Check Endpoints

**Files:**
- Modify: `pkg/server/server.go` (fill in check stubs)
- Create: `pkg/server/check.go`
- Create: `pkg/server/check_test.go`

- [ ] **Step 1: Write the test (`pkg/server/check_test.go`)**

```go
package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/enforcement"
	"github.com/permitio/permit-golang/pkg/models"
)

func TestPDPCheck(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Seed full ReBAC policy via SDK
	actions := map[string]models.ActionBlockEditable{
		"read":  *models.NewActionBlockEditable(),
		"write": *models.NewActionBlockEditable(),
	}
	env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("folder", "Folder", actions))

	docActions := map[string]models.ActionBlockEditable{
		"read": *models.NewActionBlockEditable(),
		"edit": *models.NewActionBlockEditable(),
	}
	env.client.Api.Resources.Create(ctx, *models.NewResourceCreate("document", "Document", docActions))

	env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate("default", "Default"))
	env.client.Api.Users.Create(ctx, *models.NewUserCreate("user-1"))
	env.client.Api.Users.Create(ctx, *models.NewUserCreate("user-2"))

	// Resource roles
	ownerRole := models.NewResourceRoleCreate("owner", "Owner")
	ownerRole.SetPermissions([]string{"read", "write"})
	env.client.Api.ResourceRoles.Create(ctx, "folder", *ownerRole)

	viewerRole := models.NewResourceRoleCreate("viewer", "Viewer")
	viewerRole.SetPermissions([]string{"read"})
	env.client.Api.ResourceRoles.Create(ctx, "folder", *viewerRole)

	editorRole := models.NewResourceRoleCreate("editor", "Editor")
	editorRole.SetPermissions([]string{"read", "edit"})
	env.client.Api.ResourceRoles.Create(ctx, "document", *editorRole)

	// Relation: folder --parent--> document
	rel := models.NewRelationCreate("parent", "Parent")
	rel.SetSubjectResource("document")
	env.client.Api.ResourceRelations.Create(ctx, "folder", *rel)

	// Implicit grant: folder#owner -> document#editor via parent
	ig := models.NewDerivedRoleRuleCreate("editor", "document", "parent")
	env.client.Api.ImplicitGrants.Create(ctx, "folder", "owner", *ig)

	// Instances
	env.client.Api.ResourceInstances.Create(ctx, *models.NewResourceInstanceCreate("budget", "folder", "default"))
	env.client.Api.ResourceInstances.Create(ctx, *models.NewResourceInstanceCreate("report", "document", "default"))

	// Tuple: folder:budget --parent--> document:report
	env.client.Api.RelationshipTuples.Create(ctx, *models.NewRelationshipTupleCreate("folder:budget", "parent", "document:report"))

	// Role assignment: user-1 is owner of folder:budget
	env.client.Api.Users.AssignResourceRole(ctx, "user-1", "owner", "default", "folder:budget")

	// user-2 has viewer on folder:budget
	env.client.Api.Users.AssignResourceRole(ctx, "user-2", "viewer", "default", "folder:budget")

	t.Run("DirectPermission_Allow", func(t *testing.T) {
		user := enforcement.UserBuilder("user-1").Build()
		resource := enforcement.ResourceBuilder("folder").WithKey("budget").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "read", resource)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if !allowed {
			t.Error("expected allow for direct permission")
		}
	})

	t.Run("DerivedPermission_Allow", func(t *testing.T) {
		user := enforcement.UserBuilder("user-1").Build()
		resource := enforcement.ResourceBuilder("document").WithKey("report").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "edit", resource)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if !allowed {
			t.Error("expected allow for derived permission (folder owner -> document editor)")
		}
	})

	t.Run("NoAssignment_Deny", func(t *testing.T) {
		user := enforcement.UserBuilder("nobody").Build()
		resource := enforcement.ResourceBuilder("document").WithKey("report").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "edit", resource)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if allowed {
			t.Error("expected deny for user with no assignments")
		}
	})

	t.Run("InsufficientPermission_Deny", func(t *testing.T) {
		user := enforcement.UserBuilder("user-2").Build()
		resource := enforcement.ResourceBuilder("folder").WithKey("budget").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "write", resource)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if allowed {
			t.Error("expected deny for viewer trying to write")
		}
	})

	t.Run("BulkCheck", func(t *testing.T) {
		checks := []enforcement.CheckRequest{
			{
				User:     enforcement.UserBuilder("user-1").Build(),
				Action:   "read",
				Resource: enforcement.ResourceBuilder("folder").WithKey("budget").WithTenant("default").Build(),
			},
			{
				User:     enforcement.UserBuilder("nobody").Build(),
				Action:   "edit",
				Resource: enforcement.ResourceBuilder("document").WithKey("report").WithTenant("default").Build(),
			},
		}
		results, err := env.client.BulkCheck(checks)
		if err != nil {
			t.Fatalf("BulkCheck failed: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if !results[0] {
			t.Error("expected first check to allow")
		}
		if results[1] {
			t.Error("expected second check to deny")
		}
	})

	t.Run("UserPermissions", func(t *testing.T) {
		user := enforcement.UserBuilder("user-1").Build()
		perms, err := env.client.GetUserPermissions(ctx, user, "default")
		if err != nil {
			t.Fatalf("GetUserPermissions failed: %v", err)
		}
		if len(perms) == 0 {
			t.Error("expected non-empty permissions")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/server/ -run TestPDPCheck -v -count=1`
Expected: FAIL (check stubs return 501)

- [ ] **Step 3: Implement check handlers (`pkg/server/check.go`)**

```go
package server

import (
	"encoding/json"
	"net/http"
)

type checkRequest struct {
	User     checkUser     `json:"user"`
	Action   string        `json:"action"`
	Resource checkResource `json:"resource"`
	Context  interface{}   `json:"context,omitempty"`
}

type checkUser struct {
	Key string `json:"key"`
}

type checkResource struct {
	Type   string `json:"type"`
	Key    string `json:"key,omitempty"`
	Tenant string `json:"tenant,omitempty"`
}

type checkResponse struct {
	Allow  bool        `json:"allow"`
	Result bool        `json:"result"`
	Query  interface{} `json:"query,omitempty"`
	Debug  interface{} `json:"debug,omitempty"`
}

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req checkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	allowed := s.store.CheckPermission(req.User.Key, req.Action, req.Resource.Type, req.Resource.Key, req.Resource.Tenant)
	writeJSON(w, http.StatusOK, checkResponse{
		Allow:  allowed,
		Result: allowed,
	})
}

func (s *Server) handleBulkCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var reqs []checkRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	results := make([]checkResponse, len(reqs))
	for i, req := range reqs {
		allowed := s.store.CheckPermission(req.User.Key, req.Action, req.Resource.Type, req.Resource.Key, req.Resource.Tenant)
		results[i] = checkResponse{Allow: allowed, Result: allowed}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"allow": results,
	})
}

func (s *Server) handleAllTenantsCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req checkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	tenants := s.store.ListTenants()
	var allowedTenants []map[string]interface{}
	for _, t := range tenants {
		allowed := s.store.CheckPermission(req.User.Key, req.Action, req.Resource.Type, req.Resource.Key, t.Key)
		allowedTenants = append(allowedTenants, map[string]interface{}{
			"allow":  allowed,
			"result": allowed,
			"tenant": map[string]interface{}{
				"key":        t.Key,
				"attributes": t.Attributes,
			},
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"allowed_tenants": allowedTenants,
	})
}

func (s *Server) handleUserPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		User    checkUser `json:"user"`
		Tenants []string  `json:"tenants,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	perms := s.store.GetUserPermissions(req.User.Key, req.Tenants)
	writeJSON(w, http.StatusOK, perms)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/server/ -run TestPDPCheck -v -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: PDP check endpoints with ReBAC enforcement tests"
```

---

### Task 14: Allow-All Mode Test

**Files:**
- Create: `pkg/server/allow_all_test.go`

- [ ] **Step 1: Write the test**

```go
package server

import (
	"testing"

	"github.com/46labs/permitio/pkg/config"
	"github.com/46labs/permitio/pkg/store"
	permitConfig "github.com/permitio/permit-golang/pkg/config"
	"github.com/permitio/permit-golang/pkg/enforcement"
	"github.com/permitio/permit-golang/pkg/permit"
	"net/http/httptest"
)

func TestAllowAllMode(t *testing.T) {
	cfg := &config.Config{Port: 0}
	st := store.New()
	st.SetAllowAll(true)
	srv := NewWithStore(cfg, st)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := permit.NewPermit(
		permitConfig.NewConfigBuilder("test").
			WithPdpUrl(ts.URL).
			WithApiUrl(ts.URL).
			Build(),
	)

	user := enforcement.UserBuilder("anyone").Build()
	resource := enforcement.ResourceBuilder("anything").WithKey("any-instance").WithTenant("any-tenant").Build()

	allowed, err := client.Check(user, "any-action", resource)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !allowed {
		t.Error("expected allow in allow_all mode")
	}
}
```

- [ ] **Step 2: Run test**

Run: `go test ./pkg/server/ -run TestAllowAllMode -v -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "test: allow-all mode verification"
```

---

### Task 15: Infrastructure

**Files:**
- Create: `Dockerfile`
- Create: `justfile`
- Create: `Tiltfile`
- Create: `charts/permitio/Chart.yaml`
- Create: `charts/permitio/values.yaml`
- Create: `charts/permitio/templates/_helpers.tpl`
- Create: `charts/permitio/templates/deployment.yaml`
- Create: `charts/permitio/templates/service.yaml`
- Create: `charts/permitio/templates/configmap.yaml`
- Create: `charts/permitio/templates/ingress.yaml`
- Create: `schema.yaml` (example config)
- Create: `data.yaml` (example config)

- [ ] **Step 1: Create Dockerfile**

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

- [ ] **Step 2: Create justfile**

```just
set shell := ["bash", "-uc"]

default:
	@echo "Usage:"
	@echo "  just docker - Start with docker run"
	@echo "  just kind   - Start Kind cluster + Tilt"
	@echo "  just ci     - Run tests and lint"
	@echo "  just down   - Stop everything"

docker:
	@echo "Building..."
	docker build -t permitio:dev .
	@echo "Starting on http://localhost:7766"
	docker run --rm -d \
		--name permitio \
		-p 7766:7766 \
		permitio:dev
	@echo "PDP: POST http://localhost:7766/allowed"

ci:
	@echo "Checking format..."
	@gofmt -l .
	@echo "Running tests..."
	@go test -v ./pkg/...
	@echo "Running linter..."
	@golangci-lint run

_context-guard:
	#!/usr/bin/env bash
	set -euo pipefail
	CURRENT_CONTEXT=$(kubectl config current-context 2>/dev/null || echo "none")
	ALLOWED_CONTEXTS=("kind-permitio" "docker-desktop" "none")
	for allowed in "${ALLOWED_CONTEXTS[@]}"; do
		if [[ "$CURRENT_CONTEXT" == "$allowed" ]]; then exit 0; fi
	done
	echo "ERROR: Current kubectl context '$CURRENT_CONTEXT' is not allowed"
	exit 1

kind: _context-guard
	#!/usr/bin/env bash
	set -euo pipefail
	if ! kind get clusters 2>/dev/null | grep -q "^permitio$"; then
		echo "Creating Kind cluster..."
		kind create cluster --name permitio
		kubectl config use-context kind-permitio
	else
		echo "Kind cluster exists"
		kubectl config use-context kind-permitio
	fi
	echo "Starting Tilt..."
	tilt up

down:
	@echo "Stopping..."
	docker stop permitio 2>/dev/null || true
	tilt down 2>/dev/null || true
	kind delete cluster --name permitio 2>/dev/null || true
```

- [ ] **Step 3: Create Tiltfile**

```python
load('ext://helm_resource', 'helm_resource')
load('ext://namespace', 'namespace_create')

allow_k8s_contexts('kind-permitio')
namespace_create('permitio')
update_settings(k8s_upsert_timeout_secs=120)

docker_build(
    'ghcr.io/46labs/permitio',
    '.',
    dockerfile='./Dockerfile',
    live_update=[
        sync('./pkg', '/app/pkg'),
        sync('./cmd', '/app/cmd'),
        run('go build -o /app/permitio cmd/main.go', trigger=['./go.mod', './go.sum']),
    ],
)

k8s_yaml(helm(
    './charts/permitio',
    name='permitio',
    namespace='permitio',
    set=[
        'service.type=ClusterIP',
    ],
))

k8s_resource(
    'permitio',
    labels=['permitio'],
)

local_resource(
    'health-check',
    cmd='curl -sf http://localhost:7766/v2/api-key/scope | jq .project_id || echo "Not ready"',
    auto_init=False,
    labels=['helpers'],
)
```

- [ ] **Step 4: Create Helm chart files**

Create `charts/permitio/Chart.yaml`:
```yaml
apiVersion: v2
name: permitio
description: Permit.io PDP + Management API Mock
type: application
version: 0.1.0
appVersion: latest
```

Create `charts/permitio/values.yaml`:
```yaml
image:
  repository: ghcr.io/46labs/permitio
  tag: latest
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 7766

ingress:
  enabled: false
  host: permitio.46labs.test

config:
  port: 7766
  mode: enforce
```

Create `charts/permitio/templates/_helpers.tpl`:
```
{{- define "permitio.name" -}}permitio{{- end -}}
{{- define "permitio.fullname" -}}{{ .Release.Name }}-permitio{{- end -}}
```

Create `charts/permitio/templates/configmap.yaml`:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "permitio.fullname" . }}-config
data:
  schema.yaml: |
    mode: {{ .Values.config.mode }}
  data.yaml: |
    tenants: []
```

Create `charts/permitio/templates/deployment.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "permitio.fullname" . }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{ include "permitio.name" . }}
  template:
    metadata:
      labels:
        app: {{ include "permitio.name" . }}
    spec:
      containers:
        - name: permitio
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          ports:
            - containerPort: {{ .Values.config.port }}
          env:
            - name: PORT
              value: "{{ .Values.config.port }}"
          volumeMounts:
            - name: config
              mountPath: /config
      volumes:
        - name: config
          configMap:
            name: {{ include "permitio.fullname" . }}-config
```

Create `charts/permitio/templates/service.yaml`:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "permitio.fullname" . }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: {{ .Values.config.port }}
  selector:
    app: {{ include "permitio.name" . }}
```

Create `charts/permitio/templates/ingress.yaml`:
```yaml
{{- if .Values.ingress.enabled }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "permitio.fullname" . }}
spec:
  rules:
    - host: {{ .Values.ingress.host }}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: {{ include "permitio.fullname" . }}
                port:
                  number: {{ .Values.service.port }}
{{- end }}
```

- [ ] **Step 5: Create example config files**

Create `schema.yaml` at project root with the example from the spec.
Create `data.yaml` at project root with the example from the spec.

- [ ] **Step 6: Run all tests to verify nothing broke**

Run: `go test ./pkg/... -v -count=1`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat: infrastructure - Dockerfile, justfile, Helm chart, Tiltfile"
```

---

### Task 16: Final Integration Pass

- [ ] **Step 1: Run full test suite**

Run: `go test ./pkg/... -v -count=1 2>&1 | tail -20`
Expected: ALL PASS

- [ ] **Step 2: Run linter**

Run: `golangci-lint run`
Expected: No errors

- [ ] **Step 3: Build and verify Docker image**

Run: `docker build -t permitio:dev . && docker run --rm -d --name permitio-test -p 7766:7766 permitio:dev && sleep 2 && curl -s http://localhost:7766/v2/api-key/scope | jq . && docker stop permitio-test`
Expected: Returns `{"organization_id":"org_mock","project_id":"proj_mock","environment_id":"env_mock","access_level":"environment"}`

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "chore: final integration verification"
```
