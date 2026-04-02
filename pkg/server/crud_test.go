package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

func TestTenantsCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create
	tc := *models.NewTenantCreate("t1", "Tenant One")
	tenant, err := env.client.Api.Tenants.Create(ctx, tc)
	if err != nil {
		t.Fatalf("Create tenant: %v", err)
	}
	if tenant.Key != "t1" || tenant.Name != "Tenant One" {
		t.Fatalf("unexpected tenant: %+v", tenant)
	}

	// Get
	got, err := env.client.Api.Tenants.Get(ctx, "t1")
	if err != nil {
		t.Fatalf("Get tenant: %v", err)
	}
	if got.Key != "t1" {
		t.Fatalf("unexpected tenant key: %s", got.Key)
	}

	// List
	list, err := env.client.Api.Tenants.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("List tenants: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 tenant, got %d", len(list))
	}

	// Update
	tu := *models.NewTenantUpdate()
	newName := "Updated Tenant"
	tu.SetName(newName)
	updated, err := env.client.Api.Tenants.Update(ctx, "t1", tu)
	if err != nil {
		t.Fatalf("Update tenant: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("expected name %q, got %q", newName, updated.Name)
	}

	// Delete
	err = env.client.Api.Tenants.Delete(ctx, "t1")
	if err != nil {
		t.Fatalf("Delete tenant: %v", err)
	}

	// Verify deleted
	list, err = env.client.Api.Tenants.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("List tenants after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 tenants after delete, got %d", len(list))
	}
}

func TestUsersCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create
	uc := *models.NewUserCreate("u1")
	uc.SetEmail("u1@test.com")
	uc.SetFirstName("First")
	uc.SetLastName("Last")
	user, err := env.client.Api.Users.Create(ctx, uc)
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	if user.Key != "u1" {
		t.Fatalf("unexpected user key: %s", user.Key)
	}

	// Get
	got, err := env.client.Api.Users.Get(ctx, "u1")
	if err != nil {
		t.Fatalf("Get user: %v", err)
	}
	if got.Key != "u1" {
		t.Fatalf("unexpected user key: %s", got.Key)
	}
	if got.GetEmail() != "u1@test.com" {
		t.Fatalf("unexpected email: %s", got.GetEmail())
	}

	// List
	list, err := env.client.Api.Users.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("List users: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 user, got %d", len(list))
	}

	// SyncUser (upsert - user exists, should update)
	uc2 := *models.NewUserCreate("u1")
	uc2.SetEmail("updated@test.com")
	synced, err := env.client.Api.Users.SyncUser(ctx, uc2)
	if err != nil {
		t.Fatalf("SyncUser: %v", err)
	}
	if synced.GetEmail() != "updated@test.com" {
		t.Fatalf("expected updated email, got %s", synced.GetEmail())
	}

	// Delete
	err = env.client.Api.Users.Delete(ctx, "u1")
	if err != nil {
		t.Fatalf("Delete user: %v", err)
	}

	// SyncUser (upsert - user doesn't exist, should create)
	uc3 := *models.NewUserCreate("u2")
	uc3.SetEmail("new@test.com")
	created, err := env.client.Api.Users.SyncUser(ctx, uc3)
	if err != nil {
		t.Fatalf("SyncUser (new): %v", err)
	}
	if created.Key != "u2" {
		t.Fatalf("expected key u2, got %s", created.Key)
	}
}

func TestUserRoleAssignment(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup: create tenant, role, user
	tc := *models.NewTenantCreate("default", "Default")
	_, err := env.client.Api.Tenants.Create(ctx, tc)
	if err != nil {
		t.Fatalf("Create tenant: %v", err)
	}

	rc := *models.NewRoleCreate("admin", "Admin")
	_, err = env.client.Api.Roles.Create(ctx, rc)
	if err != nil {
		t.Fatalf("Create role: %v", err)
	}

	uc := *models.NewUserCreate("u1")
	_, err = env.client.Api.Users.Create(ctx, uc)
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}

	// Assign role
	ra, err := env.client.Api.Users.AssignRole(ctx, "u1", "admin", "default")
	if err != nil {
		t.Fatalf("AssignRole: %v", err)
	}
	if ra.User != "u1" || ra.Role != "admin" || ra.Tenant != "default" {
		t.Fatalf("unexpected role assignment: %+v", ra)
	}

	// Unassign role
	_, err = env.client.Api.Users.UnassignRole(ctx, "u1", "admin", "default")
	if err != nil {
		t.Fatalf("UnassignRole: %v", err)
	}
}

func TestResourcesCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create
	actions := map[string]models.ActionBlockEditable{
		"read":  {},
		"write": {},
	}
	rc := *models.NewResourceCreate("document", "Document", actions)
	res, err := env.client.Api.Resources.Create(ctx, rc)
	if err != nil {
		t.Fatalf("Create resource: %v", err)
	}
	if res.Key != "document" || res.Name != "Document" {
		t.Fatalf("unexpected resource: %+v", res)
	}

	// Get
	got, err := env.client.Api.Resources.Get(ctx, "document")
	if err != nil {
		t.Fatalf("Get resource: %v", err)
	}
	if got.Key != "document" {
		t.Fatalf("unexpected resource key: %s", got.Key)
	}

	// List
	list, err := env.client.Api.Resources.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("List resources: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(list))
	}

	// Delete
	err = env.client.Api.Resources.Delete(ctx, "document")
	if err != nil {
		t.Fatalf("Delete resource: %v", err)
	}
}

func TestRolesCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create
	rc := *models.NewRoleCreate("viewer", "Viewer")
	role, err := env.client.Api.Roles.Create(ctx, rc)
	if err != nil {
		t.Fatalf("Create role: %v", err)
	}
	if role.Key != "viewer" || role.Name != "Viewer" {
		t.Fatalf("unexpected role: %+v", role)
	}

	// Get
	got, err := env.client.Api.Roles.Get(ctx, "viewer")
	if err != nil {
		t.Fatalf("Get role: %v", err)
	}
	if got.Key != "viewer" {
		t.Fatalf("unexpected role key: %s", got.Key)
	}

	// List
	list, err := env.client.Api.Roles.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("List roles: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 role, got %d", len(list))
	}

	// AssignPermissions
	err = env.client.Api.Roles.AssignPermissions(ctx, "viewer", []string{"document:read"})
	if err != nil {
		t.Fatalf("AssignPermissions: %v", err)
	}

	// Verify permissions
	got, err = env.client.Api.Roles.Get(ctx, "viewer")
	if err != nil {
		t.Fatalf("Get role after assign: %v", err)
	}
	if len(got.Permissions) != 1 || got.Permissions[0] != "document:read" {
		t.Fatalf("expected permissions [document:read], got %v", got.Permissions)
	}

	// Delete
	err = env.client.Api.Roles.Delete(ctx, "viewer")
	if err != nil {
		t.Fatalf("Delete role: %v", err)
	}
}

func TestResourceRolesCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup: create a resource
	actions := map[string]models.ActionBlockEditable{"read": {}, "write": {}}
	resCreate := *models.NewResourceCreate("doc", "Document", actions)
	_, err := env.client.Api.Resources.Create(ctx, resCreate)
	if err != nil {
		t.Fatalf("Create resource: %v", err)
	}

	// Create resource role
	rrc := *models.NewResourceRoleCreate("editor", "Editor")
	rrc.SetPermissions([]string{"doc:read", "doc:write"})
	rr, err := env.client.Api.ResourceRoles.Create(ctx, "doc", rrc)
	if err != nil {
		t.Fatalf("Create resource role: %v", err)
	}
	if rr.Key != "editor" || rr.Name != "Editor" {
		t.Fatalf("unexpected resource role: %+v", rr)
	}

	// Get
	got, err := env.client.Api.ResourceRoles.Get(ctx, "doc", "editor")
	if err != nil {
		t.Fatalf("Get resource role: %v", err)
	}
	if got.Key != "editor" {
		t.Fatalf("unexpected resource role key: %s", got.Key)
	}

	// List
	listPtr, err := env.client.Api.ResourceRoles.List(ctx, 1, 10, "doc")
	if err != nil {
		t.Fatalf("List resource roles: %v", err)
	}
	if listPtr == nil || len(*listPtr) != 1 {
		t.Fatalf("expected 1 resource role in list")
	}

	// Delete
	err = env.client.Api.ResourceRoles.Delete(ctx, "doc", "editor")
	if err != nil {
		t.Fatalf("Delete resource role: %v", err)
	}
}

func TestResourceRelationsCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup: create two resources
	actions := map[string]models.ActionBlockEditable{"read": {}}
	r1 := *models.NewResourceCreate("folder", "Folder", actions)
	_, err := env.client.Api.Resources.Create(ctx, r1)
	if err != nil {
		t.Fatalf("Create folder resource: %v", err)
	}
	r2 := *models.NewResourceCreate("file", "File", actions)
	_, err = env.client.Api.Resources.Create(ctx, r2)
	if err != nil {
		t.Fatalf("Create file resource: %v", err)
	}

	// Create relation on file: parent -> folder
	relCreate := *models.NewRelationCreate("parent", "Parent", "folder")
	rel, err := env.client.Api.ResourceRelations.Create(ctx, "file", relCreate)
	if err != nil {
		t.Fatalf("Create relation: %v", err)
	}
	if rel.Key != "parent" {
		t.Fatalf("unexpected relation key: %s", rel.Key)
	}
	if rel.SubjectResource != "folder" {
		t.Fatalf("unexpected subject_resource: %s", rel.SubjectResource)
	}

	// List
	listPtr, err := env.client.Api.ResourceRelations.List(ctx, 1, 10, "file")
	if err != nil {
		t.Fatalf("List relations: %v", err)
	}
	if listPtr == nil || len(*listPtr) != 1 {
		t.Fatalf("expected 1 relation in list")
	}

	// Delete
	err = env.client.Api.ResourceRelations.Delete(ctx, "file", "parent")
	if err != nil {
		t.Fatalf("Delete relation: %v", err)
	}
}

func TestImplicitGrants(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup: create resources, roles, relations
	actions := map[string]models.ActionBlockEditable{"read": {}}
	r1 := *models.NewResourceCreate("folder", "Folder", actions)
	_, err := env.client.Api.Resources.Create(ctx, r1)
	if err != nil {
		t.Fatalf("Create folder: %v", err)
	}
	r2 := *models.NewResourceCreate("file", "File", actions)
	_, err = env.client.Api.Resources.Create(ctx, r2)
	if err != nil {
		t.Fatalf("Create file: %v", err)
	}

	// Create resource roles
	rrc := *models.NewResourceRoleCreate("viewer", "Viewer")
	_, err = env.client.Api.ResourceRoles.Create(ctx, "file", rrc)
	if err != nil {
		t.Fatalf("Create file viewer role: %v", err)
	}
	rrc2 := *models.NewResourceRoleCreate("viewer", "Viewer")
	_, err = env.client.Api.ResourceRoles.Create(ctx, "folder", rrc2)
	if err != nil {
		t.Fatalf("Create folder viewer role: %v", err)
	}

	// Create relation
	relCreate := *models.NewRelationCreate("parent", "Parent", "folder")
	_, err = env.client.Api.ResourceRelations.Create(ctx, "file", relCreate)
	if err != nil {
		t.Fatalf("Create relation: %v", err)
	}

	// Create implicit grant
	igCreate := *models.NewDerivedRoleRuleCreate("viewer", "folder", "parent")
	result, err := env.client.Api.ImplicitGrants.Create(ctx, "file", "viewer", igCreate)
	if err != nil {
		t.Fatalf("Create implicit grant: %v", err)
	}
	if result.Role != "viewer" || result.OnResource != "folder" || result.LinkedByRelation != "parent" {
		t.Fatalf("unexpected implicit grant result: %+v", result)
	}
}

func TestResourceInstancesCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup: create a resource and tenant
	actions := map[string]models.ActionBlockEditable{"read": {}}
	resCreate := *models.NewResourceCreate("folder", "Folder", actions)
	_, err := env.client.Api.Resources.Create(ctx, resCreate)
	if err != nil {
		t.Fatalf("Create resource: %v", err)
	}
	tc := *models.NewTenantCreate("default", "Default")
	_, err = env.client.Api.Tenants.Create(ctx, tc)
	if err != nil {
		t.Fatalf("Create tenant: %v", err)
	}

	// Create instance
	ic := *models.NewResourceInstanceCreate("budget", "folder")
	ic.SetTenant("default")
	inst, err := env.client.Api.ResourceInstances.Create(ctx, ic)
	if err != nil {
		t.Fatalf("Create resource instance: %v", err)
	}
	if inst.Key != "budget" || inst.Resource != "folder" {
		t.Fatalf("unexpected instance: %+v", inst)
	}

	// Get
	got, err := env.client.Api.ResourceInstances.Get(ctx, "folder:budget")
	if err != nil {
		t.Fatalf("Get resource instance: %v", err)
	}
	if got.Key != "budget" {
		t.Fatalf("unexpected instance key: %s", got.Key)
	}

	// List
	listPtr, err := env.client.Api.ResourceInstances.List(ctx, 1, 10, "", "", "")
	if err != nil {
		t.Fatalf("List resource instances: %v", err)
	}
	if listPtr == nil || len(*listPtr) != 1 {
		t.Fatalf("expected 1 resource instance in list")
	}

	// Delete
	err = env.client.Api.ResourceInstances.Delete(ctx, "folder:budget")
	if err != nil {
		t.Fatalf("Delete resource instance: %v", err)
	}
}

func TestRelationshipTuplesCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup resources, instances, relations
	actions := map[string]models.ActionBlockEditable{"read": {}}
	r1 := *models.NewResourceCreate("folder", "Folder", actions)
	_, err := env.client.Api.Resources.Create(ctx, r1)
	if err != nil {
		t.Fatalf("Create folder: %v", err)
	}
	r2 := *models.NewResourceCreate("file", "File", actions)
	_, err = env.client.Api.Resources.Create(ctx, r2)
	if err != nil {
		t.Fatalf("Create file: %v", err)
	}

	tc := *models.NewTenantCreate("default", "Default")
	_, err = env.client.Api.Tenants.Create(ctx, tc)
	if err != nil {
		t.Fatalf("Create tenant: %v", err)
	}

	// Create relation
	relCreate := *models.NewRelationCreate("parent", "Parent", "folder")
	_, err = env.client.Api.ResourceRelations.Create(ctx, "file", relCreate)
	if err != nil {
		t.Fatalf("Create relation: %v", err)
	}

	// Create instances
	ic1 := *models.NewResourceInstanceCreate("docs", "folder")
	ic1.SetTenant("default")
	_, err = env.client.Api.ResourceInstances.Create(ctx, ic1)
	if err != nil {
		t.Fatalf("Create folder instance: %v", err)
	}
	ic2 := *models.NewResourceInstanceCreate("readme", "file")
	ic2.SetTenant("default")
	_, err = env.client.Api.ResourceInstances.Create(ctx, ic2)
	if err != nil {
		t.Fatalf("Create file instance: %v", err)
	}

	// Create relationship tuple
	rtc := *models.NewRelationshipTupleCreate("folder:docs", "parent", "file:readme")
	rt, err := env.client.Api.RelationshipTuples.Create(ctx, rtc)
	if err != nil {
		t.Fatalf("Create relationship tuple: %v", err)
	}
	if rt.Subject != "folder:docs" || rt.Relation != "parent" || rt.Object != "file:readme" {
		t.Fatalf("unexpected tuple: %+v", rt)
	}

	// List
	listPtr, err := env.client.Api.RelationshipTuples.List(ctx, 1, 10, "", "", "", "")
	if err != nil {
		t.Fatalf("List relationship tuples: %v", err)
	}
	if listPtr == nil || len(*listPtr) != 1 {
		t.Fatalf("expected 1 tuple in list")
	}

	// Delete
	rtd := *models.NewRelationshipTupleDelete("folder:docs", "parent", "file:readme")
	err = env.client.Api.RelationshipTuples.Delete(ctx, rtd)
	if err != nil {
		t.Fatalf("Delete relationship tuple: %v", err)
	}
}

func TestRoleAssignmentsList(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup
	tc := *models.NewTenantCreate("default", "Default")
	_, err := env.client.Api.Tenants.Create(ctx, tc)
	if err != nil {
		t.Fatalf("Create tenant: %v", err)
	}
	rc := *models.NewRoleCreate("admin", "Admin")
	_, err = env.client.Api.Roles.Create(ctx, rc)
	if err != nil {
		t.Fatalf("Create role: %v", err)
	}
	uc := *models.NewUserCreate("u1")
	_, err = env.client.Api.Users.Create(ctx, uc)
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}

	// Assign via Users API
	_, err = env.client.Api.Users.AssignRole(ctx, "u1", "admin", "default")
	if err != nil {
		t.Fatalf("AssignRole: %v", err)
	}

	// List via RoleAssignments API
	raList, err := env.client.Api.RoleAssignments.List(ctx, 1, 10, "", "", "")
	if err != nil {
		t.Fatalf("List role assignments: %v", err)
	}
	if raList == nil || len(*raList) != 1 {
		t.Fatalf("expected 1 role assignment, got %v", raList)
	}
	ra := (*raList)[0]
	if ra.User != "u1" || ra.Role != "admin" || ra.Tenant != "default" {
		t.Fatalf("unexpected role assignment: %+v", ra)
	}
}
