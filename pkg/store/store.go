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
	MockOrgID  = "org_mock"
	MockProjID = "proj_mock"
	MockEnvID  = "env_mock"
)

type Store struct {
	mu sync.RWMutex

	// Schema
	resources      map[string]*Resource                // key -> resource
	roles          map[string]*Role                    // key -> global role
	resourceRoles  map[string]map[string]*ResourceRole // resourceKey -> roleKey -> role
	relations      map[string]map[string]*Relation     // resourceKey -> relationKey -> relation
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
		resources:          make(map[string]*Resource),
		roles:              make(map[string]*Role),
		resourceRoles:      make(map[string]map[string]*ResourceRole),
		relations:          make(map[string]map[string]*Relation),
		tenants:            make(map[string]*Tenant),
		users:              make(map[string]*User),
		resourceInstances:  make(map[string]*ResourceInstance),
		tupleIndex:         make(map[string]map[string][]string),
		effectivePerms:     make(map[string]bool),
		userPerms:          make(map[string]map[string][]string),
		userRoles:          make(map[string]map[string][]string),
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
