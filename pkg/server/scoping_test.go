package server

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/46labs/permitio/pkg/config"
	"github.com/46labs/permitio/pkg/store"
	"github.com/permitio/permit-golang/pkg/enforcement"
	"github.com/permitio/permit-golang/pkg/models"
)

// seedHierarchy builds location -> zone -> trunk_group with a viewer role on
// each, derived downward through "parent" relations, in tenants t1 and t2.
func seedHierarchy(t *testing.T, env *testEnv) {
	t.Helper()
	ctx := context.Background()

	types := []string{"location", "zone", "trunk_group"}
	for _, rt := range types {
		res := *models.NewResourceCreate(rt, rt, map[string]models.ActionBlockEditable{"read": {}})
		if _, err := env.client.Api.Resources.Create(ctx, res); err != nil {
			t.Fatalf("create resource %s: %v", rt, err)
		}
		role := *models.NewResourceRoleCreate("viewer", "Viewer")
		role.SetPermissions([]string{rt + ":read"})
		if _, err := env.client.Api.ResourceRoles.Create(ctx, rt, role); err != nil {
			t.Fatalf("create %s viewer: %v", rt, err)
		}
	}
	for i := 1; i < len(types); i++ {
		rel := *models.NewRelationCreate("parent", "Parent", types[i-1])
		if _, err := env.client.Api.ResourceRelations.Create(ctx, types[i], rel); err != nil {
			t.Fatalf("create %s parent relation: %v", types[i], err)
		}
		ig := *models.NewDerivedRoleRuleCreate("viewer", types[i-1], "parent")
		if _, err := env.client.Api.ImplicitGrants.Create(ctx, types[i], "viewer", ig); err != nil {
			t.Fatalf("create %s implicit grant: %v", types[i], err)
		}
	}

	for _, tenant := range []string{"t1", "t2"} {
		if _, err := env.client.Api.Tenants.Create(ctx, *models.NewTenantCreate(tenant, tenant)); err != nil {
			t.Fatalf("create tenant %s: %v", tenant, err)
		}
	}
	if _, err := env.client.Api.Users.Create(ctx, *models.NewUserCreate("u1")); err != nil {
		t.Fatalf("create user: %v", err)
	}

	instances := map[string]string{
		"location:L1": "t1", "zone:Z1": "t1", "trunk_group:TG1": "t1", "trunk_group:TG2": "t1",
		"location:L2": "t2",
	}
	for instKey, tenant := range instances {
		rt, key, _ := strings.Cut(instKey, ":")
		ri := *models.NewResourceInstanceCreate(key, rt)
		ri.SetTenant(tenant)
		if _, err := env.client.Api.ResourceInstances.Create(ctx, ri); err != nil {
			t.Fatalf("create instance %s: %v", instKey, err)
		}
	}
	for _, tuple := range [][2]string{{"location:L1", "zone:Z1"}, {"zone:Z1", "trunk_group:TG1"}, {"zone:Z1", "trunk_group:TG2"}} {
		rt := *models.NewRelationshipTupleCreate(tuple[0], "parent", tuple[1])
		if _, err := env.client.Api.RelationshipTuples.Create(ctx, rt); err != nil {
			t.Fatalf("create tuple %v: %v", tuple, err)
		}
	}
}

func checkRead(t *testing.T, env *testEnv, resType, key, tenant string) bool {
	t.Helper()
	user := enforcement.UserBuilder("u1").Build()
	resource := enforcement.ResourceBuilder(resType).WithKey(key).WithTenant(tenant).Build()
	allowed, err := env.client.Check(user, "read", resource)
	if err != nil {
		t.Fatalf("check %s:%s: %v", resType, key, err)
	}
	return allowed
}

func TestDerivationFollowsEveryLevel(t *testing.T) {
	env := setupTestEnv(t)
	seedHierarchy(t, env)
	if _, err := env.client.Api.Users.AssignResourceRole(context.Background(), "u1", "viewer", "t1", "location:L1"); err != nil {
		t.Fatalf("assign: %v", err)
	}

	tests := []struct {
		resType, key string
		want         bool
	}{
		{"location", "L1", true},
		{"zone", "Z1", true},
		{"trunk_group", "TG1", true},
		{"trunk_group", "TG2", true},
		{"location", "L2", false},
	}
	for _, tt := range tests {
		tenant := "t1"
		if tt.key == "L2" {
			tenant = "t2"
		}
		if got := checkRead(t, env, tt.resType, tt.key, tenant); got != tt.want {
			t.Errorf("%s:%s allowed = %v, want %v", tt.resType, tt.key, got, tt.want)
		}
	}
}

func TestUnassignResourceRoleRevokes(t *testing.T) {
	env := setupTestEnv(t)
	seedHierarchy(t, env)
	ctx := context.Background()
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "viewer", "t1", "zone:Z1"); err != nil {
		t.Fatalf("assign zone: %v", err)
	}
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "viewer", "t1", "trunk_group:TG2"); err != nil {
		t.Fatalf("assign tg: %v", err)
	}
	if _, err := env.client.Api.Users.UnassignResourceRole(ctx, "u1", "viewer", "t1", "zone:Z1"); err != nil {
		t.Fatalf("unassign: %v", err)
	}

	if checkRead(t, env, "zone", "Z1", "t1") || checkRead(t, env, "trunk_group", "TG1", "t1") {
		t.Error("expected zone grant and its derivations revoked")
	}
	if !checkRead(t, env, "trunk_group", "TG2", "t1") {
		t.Error("expected the unrelated trunk group grant kept")
	}
	if got := len(env.store.ListRoleAssignmentsForUser("u1", "t1")); got != 1 {
		t.Errorf("expected 1 remaining assignment, got %d", got)
	}
}

func TestDeleteInstanceRemovesAssignmentsAndTuples(t *testing.T) {
	env := setupTestEnv(t)
	seedHierarchy(t, env)
	ctx := context.Background()
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "viewer", "t1", "zone:Z1"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if err := env.client.Api.ResourceInstances.Delete(ctx, "zone:Z1"); err != nil {
		t.Fatalf("delete instance: %v", err)
	}

	if len(env.store.ListRoleAssignmentsForUser("u1", "t1")) != 0 {
		t.Error("expected the deleted instance's assignments removed")
	}
	if checkRead(t, env, "trunk_group", "TG1", "t1") {
		t.Error("expected no derivation through a deleted instance")
	}
	for _, rt := range env.store.ListRelationshipTuples() {
		if rt.Subject == "zone:Z1" || rt.Object == "zone:Z1" {
			t.Errorf("expected tuples with zone:Z1 removed, found %+v", rt)
		}
	}
}

func TestAssignmentRefusesInstanceOfAnotherTenant(t *testing.T) {
	env := setupTestEnv(t)
	seedHierarchy(t, env)
	if _, err := env.client.Api.Users.AssignResourceRole(context.Background(), "u1", "viewer", "t1", "location:L2"); err == nil {
		t.Fatal("expected assignment of a t2 instance in t1 to be refused")
	}
	if checkRead(t, env, "location", "L2", "t1") || checkRead(t, env, "location", "L2", "t2") {
		t.Error("expected no access to location:L2")
	}
}

func TestUserPermissionsFilters(t *testing.T) {
	env := setupTestEnv(t)
	seedHierarchy(t, env)
	ctx := context.Background()
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "viewer", "t1", "zone:Z1"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if _, err := env.client.Api.Roles.Create(ctx, *models.NewRoleCreate("member", "Member")); err != nil {
		t.Fatalf("create tenant role: %v", err)
	}
	if _, err := env.client.Api.Users.AssignRole(ctx, "u1", "member", "t1"); err != nil {
		t.Fatalf("assign tenant role: %v", err)
	}

	tests := []struct {
		name string
		body map[string]any
		want []string
	}{
		{"unfiltered", map[string]any{}, []string{"__tenant:t1", "zone:Z1", "trunk_group:TG1", "trunk_group:TG2"}},
		{"resource type", map[string]any{"resource_types": []string{"trunk_group"}}, []string{"trunk_group:TG1", "trunk_group:TG2"}},
		{"tenant type", map[string]any{"resource_types": []string{store.TenantResourceType}}, []string{"__tenant:t1"}},
		{"resource", map[string]any{"resources": []string{"trunk_group:TG2"}}, []string{"trunk_group:TG2"}},
		{"other tenant", map[string]any{"tenants": []string{"t2"}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{"user": map[string]string{"key": "u1"}}
			for k, v := range tt.body {
				body[k] = v
			}
			payload, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			res, err := http.Post(env.ts.URL+"/user-permissions", "application/json", bytes.NewReader(payload))
			if err != nil {
				t.Fatalf("post: %v", err)
			}
			defer func() {
				if err := res.Body.Close(); err != nil {
					t.Errorf("close body: %v", err)
				}
			}()
			var got map[string]store.UserPermissionSet
			if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got keys %v, want %v", slices.Collect(maps.Keys(got)), tt.want)
			}
			for _, k := range tt.want {
				if _, ok := got[k]; !ok {
					t.Errorf("missing %s, got %v", k, slices.Collect(maps.Keys(got)))
				}
			}
		})
	}
}

func TestDeferredInstanceInAnotherTenantRevokes(t *testing.T) {
	env := setupTestEnv(t)
	seedHierarchy(t, env)
	ctx := context.Background()
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "viewer", "t1", "zone:Z9"); err != nil {
		t.Fatalf("assign before instance exists: %v", err)
	}
	if !checkRead(t, env, "zone", "Z9", "t1") {
		t.Fatal("expected the deferred assignment to grant in t1")
	}
	ri := *models.NewResourceInstanceCreate("Z9", "zone")
	ri.SetTenant("t2")
	if _, err := env.client.Api.ResourceInstances.Create(ctx, ri); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if checkRead(t, env, "zone", "Z9", "t1") {
		t.Error("expected the t1 assignment inert once zone:Z9 is created in t2")
	}
}

func TestUserPermissionsReportsRoleWithoutPermissions(t *testing.T) {
	env := setupTestEnv(t)
	seedHierarchy(t, env)
	ctx := context.Background()
	empty := *models.NewResourceRoleCreate("member", "Member")
	if _, err := env.client.Api.ResourceRoles.Create(ctx, "zone", empty); err != nil {
		t.Fatalf("create role: %v", err)
	}
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "member", "t1", "zone:Z1"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	perms, err := env.client.GetUserPermissions(enforcement.UserBuilder("u1").Build(), "t1")
	if err != nil {
		t.Fatalf("user permissions: %v", err)
	}
	set, ok := perms["zone:Z1"]
	if !ok || len(set.Roles) != 1 || set.Roles[0] != "zone#member" {
		t.Errorf("expected zone:Z1 with role zone#member, got %+v (keys %v)", set, keysOf(perms))
	}
}

func TestDerivationStaysInTupleTenant(t *testing.T) {
	env := setupTestEnv(t)
	seedHierarchy(t, env)
	ctx := context.Background()
	tuple := *models.NewRelationshipTupleCreate("zone:Z1", "parent", "trunk_group:TG9")
	tuple.SetTenant("t2")
	if _, err := env.client.Api.RelationshipTuples.Create(ctx, tuple); err != nil {
		t.Fatalf("create t2 tuple: %v", err)
	}
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "viewer", "t1", "location:L1"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if !checkRead(t, env, "trunk_group", "TG1", "t1") {
		t.Error("expected the t1 chain to derive")
	}
	if checkRead(t, env, "trunk_group", "TG9", "t1") {
		t.Error("expected no derivation through a t2 tuple")
	}
}

func TestTenantlessInstanceHasOneOwningTenant(t *testing.T) {
	env := setupTestEnv(t)
	seedHierarchy(t, env)
	ctx := context.Background()
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "viewer", "t1", "zone:Z9"); err != nil {
		t.Fatalf("assign in t1: %v", err)
	}
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "viewer", "t2", "zone:Z9"); err == nil {
		t.Fatal("expected a second tenant's assignment on zone:Z9 to be refused")
	}
}

func TestSeededInstanceWithoutTenantIsAssignable(t *testing.T) {
	st := store.New()
	st.Seed(&config.Config{
		Schema: config.SchemaConfig{Resources: []config.ResourceConfig{{
			Key:     "zone",
			Name:    "Zone",
			Actions: map[string]config.ActionBlock{"read": {}},
			Roles:   []config.ResourceRoleConfig{{Key: "viewer", Name: "Viewer", Permissions: []string{"zone:read"}}},
		}}},
		Data: config.DataConfig{
			Tenants:           []config.TenantConfig{{Key: "t1", Name: "t1"}},
			ResourceInstances: []config.ResourceInstanceConfig{{Key: "Z1", Resource: "zone"}},
			RoleAssignments:   []config.RoleAssignmentConfig{{User: "u1", Role: "viewer", Tenant: "t1", ResourceInstance: "zone:Z1"}},
		},
	})
	st.Materialize()
	if !st.CheckPermission("u1", "read", "zone", "Z1", "t1") {
		t.Error("expected a seeded tenantless instance to be assignable")
	}
}

func TestOwnershipFollowsCreatedInstance(t *testing.T) {
	env := setupTestEnv(t)
	seedHierarchy(t, env)
	ctx := context.Background()
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "viewer", "t1", "zone:Z9"); err != nil {
		t.Fatalf("assign in t1: %v", err)
	}
	ri := *models.NewResourceInstanceCreate("Z9", "zone")
	ri.SetTenant("t2")
	if _, err := env.client.Api.ResourceInstances.Create(ctx, ri); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "viewer", "t2", "zone:Z9"); err != nil {
		t.Fatalf("expected the owning tenant's assignment accepted: %v", err)
	}
	if !checkRead(t, env, "zone", "Z9", "t2") {
		t.Error("expected access in t2")
	}
}

func TestTupleTenantIgnoredForExistingInstances(t *testing.T) {
	env := setupTestEnv(t)
	seedHierarchy(t, env)
	ctx := context.Background()
	ri := *models.NewResourceInstanceCreate("TG3", "trunk_group")
	ri.SetTenant("t1")
	if _, err := env.client.Api.ResourceInstances.Create(ctx, ri); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	tuple := *models.NewRelationshipTupleCreate("zone:Z1", "parent", "trunk_group:TG3")
	tuple.SetTenant("t2")
	if _, err := env.client.Api.RelationshipTuples.Create(ctx, tuple); err != nil {
		t.Fatalf("create tuple: %v", err)
	}
	if _, err := env.client.Api.Users.AssignResourceRole(ctx, "u1", "viewer", "t1", "zone:Z1"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if !checkRead(t, env, "trunk_group", "TG3", "t1") {
		t.Error("expected derivation to the existing t1 instance despite the tuple's tenant")
	}
}
