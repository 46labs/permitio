package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

func TestResourceActionsCRUD(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create a resource with actions
	actions := map[string]models.ActionBlockEditable{
		"read":  {},
		"write": {},
	}
	rc := *models.NewResourceCreate("document", "Document", actions)
	_, err := env.client.Api.Resources.Create(ctx, rc)
	if err != nil {
		t.Fatalf("Create resource: %v", err)
	}

	// List actions — verify at least 2 come back
	actionList, err := env.client.Api.ResourceActions.List(ctx, "document", 1, 10)
	if err != nil {
		t.Fatalf("List actions: %v", err)
	}
	if len(actionList) < 2 {
		t.Fatalf("expected at least 2 actions, got %d", len(actionList))
	}

	// Get single action — verify key and permission_name
	readAction, err := env.client.Api.ResourceActions.Get(ctx, "document", "read")
	if err != nil {
		t.Fatalf("Get action: %v", err)
	}
	if readAction.Key != "read" {
		t.Fatalf("expected action key 'read', got %q", readAction.Key)
	}
	if readAction.PermissionName != "document:read" {
		t.Fatalf("expected permission_name 'document:read', got %q", readAction.PermissionName)
	}

	// Create a new action
	ac := *models.NewResourceActionCreate("delete", "Delete")
	created, err := env.client.Api.ResourceActions.Create(ctx, "document", ac)
	if err != nil {
		t.Fatalf("Create action: %v", err)
	}
	if created.Key != "delete" {
		t.Fatalf("expected created action key 'delete', got %q", created.Key)
	}

	// Update the action
	au := *models.NewResourceActionUpdate()
	au.SetName("Delete Permanently")
	updated, err := env.client.Api.ResourceActions.Update(ctx, "document", "delete", au)
	if err != nil {
		t.Fatalf("Update action: %v", err)
	}
	if updated.Name != "Delete Permanently" {
		t.Fatalf("expected updated name 'Delete Permanently', got %q", updated.Name)
	}

	// Delete the action
	err = env.client.Api.ResourceActions.Delete(ctx, "document", "delete")
	if err != nil {
		t.Fatalf("Delete action: %v", err)
	}

	// Verify deleted action returns error on Get
	_, err = env.client.Api.ResourceActions.Get(ctx, "document", "delete")
	if err == nil {
		t.Fatal("expected error getting deleted action, got nil")
	}
}

func TestUserGetAssignedRoles(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup: create tenants, role, user
	tc1 := *models.NewTenantCreate("t1", "Tenant One")
	_, err := env.client.Api.Tenants.Create(ctx, tc1)
	if err != nil {
		t.Fatalf("Create tenant t1: %v", err)
	}
	tc2 := *models.NewTenantCreate("t2", "Tenant Two")
	_, err = env.client.Api.Tenants.Create(ctx, tc2)
	if err != nil {
		t.Fatalf("Create tenant t2: %v", err)
	}

	rc := *models.NewRoleCreate("admin", "Admin")
	_, err = env.client.Api.Roles.Create(ctx, rc)
	if err != nil {
		t.Fatalf("Create role: %v", err)
	}

	uc := *models.NewUserCreate("user1")
	_, err = env.client.Api.Users.Create(ctx, uc)
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}

	// Assign role in both tenants
	_, err = env.client.Api.Users.AssignRole(ctx, "user1", "admin", "t1")
	if err != nil {
		t.Fatalf("AssignRole t1: %v", err)
	}
	_, err = env.client.Api.Users.AssignRole(ctx, "user1", "admin", "t2")
	if err != nil {
		t.Fatalf("AssignRole t2: %v", err)
	}

	// GetAssignedRoles with empty tenant -> all assignments
	allRoles, err := env.client.Api.Users.GetAssignedRoles(ctx, "user1", "", 1, 10)
	if err != nil {
		t.Fatalf("GetAssignedRoles (all): %v", err)
	}
	if len(allRoles) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(allRoles))
	}

	// GetAssignedRoles with specific tenant -> filtered
	t1Roles, err := env.client.Api.Users.GetAssignedRoles(ctx, "user1", "t1", 1, 10)
	if err != nil {
		t.Fatalf("GetAssignedRoles (t1): %v", err)
	}
	if len(t1Roles) != 1 {
		t.Fatalf("expected 1 assignment for t1, got %d", len(t1Roles))
	}
	if t1Roles[0].Tenant != "t1" {
		t.Fatalf("expected tenant 't1', got %q", t1Roles[0].Tenant)
	}
}

func TestTenantListUsers(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup: tenant, role, 3 users
	tc := *models.NewTenantCreate("default", "Default")
	_, err := env.client.Api.Tenants.Create(ctx, tc)
	if err != nil {
		t.Fatalf("Create tenant: %v", err)
	}

	rc := *models.NewRoleCreate("viewer", "Viewer")
	_, err = env.client.Api.Roles.Create(ctx, rc)
	if err != nil {
		t.Fatalf("Create role: %v", err)
	}

	for _, key := range []string{"u1", "u2", "u3"} {
		uc := *models.NewUserCreate(key)
		_, err = env.client.Api.Users.Create(ctx, uc)
		if err != nil {
			t.Fatalf("Create user %s: %v", key, err)
		}
	}

	// Assign 2 of 3 users to tenant
	_, err = env.client.Api.Users.AssignRole(ctx, "u1", "viewer", "default")
	if err != nil {
		t.Fatalf("AssignRole u1: %v", err)
	}
	_, err = env.client.Api.Users.AssignRole(ctx, "u2", "viewer", "default")
	if err != nil {
		t.Fatalf("AssignRole u2: %v", err)
	}

	// ListTenantUsers -> only the 2 assigned users
	users, err := env.client.Api.Tenants.ListTenantUsers(ctx, "default", 1, 10)
	if err != nil {
		t.Fatalf("ListTenantUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users in tenant, got %d", len(users))
	}
}

func TestRoleParentRelationships(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create two roles
	rc1 := *models.NewRoleCreate("child", "Child")
	_, err := env.client.Api.Roles.Create(ctx, rc1)
	if err != nil {
		t.Fatalf("Create child role: %v", err)
	}
	rc2 := *models.NewRoleCreate("parent", "Parent")
	_, err = env.client.Api.Roles.Create(ctx, rc2)
	if err != nil {
		t.Fatalf("Create parent role: %v", err)
	}

	// AddParentRole
	err = env.client.Api.Roles.AddParentRole(ctx, "child", "parent")
	if err != nil {
		t.Fatalf("AddParentRole: %v", err)
	}

	// Verify extends contains parent
	got, err := env.client.Api.Roles.Get(ctx, "child")
	if err != nil {
		t.Fatalf("Get child role: %v", err)
	}
	if len(got.Extends) == 0 || !containsStr(got.Extends, "parent") {
		t.Fatalf("expected extends to contain 'parent', got %v", got.Extends)
	}

	// RemoveParentRole
	err = env.client.Api.Roles.RemoveParentRole(ctx, "child", "parent")
	if err != nil {
		t.Fatalf("RemoveParentRole: %v", err)
	}

	// Verify extends is empty
	got, err = env.client.Api.Roles.Get(ctx, "child")
	if err != nil {
		t.Fatalf("Get child role after remove: %v", err)
	}
	if len(got.Extends) != 0 {
		t.Fatalf("expected empty extends, got %v", got.Extends)
	}
}

func TestResourceRoleParentRelationships(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create resource with two roles
	actions := map[string]models.ActionBlockEditable{"read": {}, "write": {}}
	resCreate := *models.NewResourceCreate("doc", "Document", actions)
	_, err := env.client.Api.Resources.Create(ctx, resCreate)
	if err != nil {
		t.Fatalf("Create resource: %v", err)
	}

	rrc1 := *models.NewResourceRoleCreate("child", "Child Role")
	_, err = env.client.Api.ResourceRoles.Create(ctx, "doc", rrc1)
	if err != nil {
		t.Fatalf("Create child resource role: %v", err)
	}
	rrc2 := *models.NewResourceRoleCreate("parent", "Parent Role")
	_, err = env.client.Api.ResourceRoles.Create(ctx, "doc", rrc2)
	if err != nil {
		t.Fatalf("Create parent resource role: %v", err)
	}

	// AddParent
	result, err := env.client.Api.ResourceRoles.AddParent(ctx, "doc", "child", "parent")
	if err != nil {
		t.Fatalf("AddParent: %v", err)
	}
	if !containsStr(result.Extends, "parent") {
		t.Fatalf("expected extends to contain 'parent', got %v", result.Extends)
	}

	// RemoveParent
	result, err = env.client.Api.ResourceRoles.RemoveParent(ctx, "doc", "child", "parent")
	if err != nil {
		t.Fatalf("RemoveParent: %v", err)
	}
	if len(result.Extends) != 0 {
		t.Fatalf("expected empty extends after remove, got %v", result.Extends)
	}
}

func TestBulkRoleAssignments(t *testing.T) {
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
	for _, key := range []string{"u1", "u2"} {
		uc := *models.NewUserCreate(key)
		_, err = env.client.Api.Users.Create(ctx, uc)
		if err != nil {
			t.Fatalf("Create user %s: %v", key, err)
		}
	}

	// BulkAssignRole
	assignments := []models.RoleAssignmentCreate{
		*models.NewRoleAssignmentCreate("admin", "default", "u1"),
		*models.NewRoleAssignmentCreate("admin", "default", "u2"),
	}
	report, err := env.client.Api.Roles.BulkAssignRole(ctx, assignments)
	if err != nil {
		t.Fatalf("BulkAssignRole: %v", err)
	}
	if report.GetAssignmentsCreated() != 2 {
		t.Fatalf("expected 2 assignments created, got %d", report.GetAssignmentsCreated())
	}

	// List -> verify 2
	raList, err := env.client.Api.RoleAssignments.List(ctx, 1, 10, "", "", "")
	if err != nil {
		t.Fatalf("List role assignments: %v", err)
	}
	if raList == nil || len(*raList) != 2 {
		t.Fatalf("expected 2 role assignments, got %v", raList)
	}

	// BulkUnAssignRole
	unassignments := []models.RoleAssignmentRemove{
		*models.NewRoleAssignmentRemove("admin", "default", "u1"),
		*models.NewRoleAssignmentRemove("admin", "default", "u2"),
	}
	unReport, err := env.client.Api.Roles.BulkUnAssignRole(ctx, unassignments)
	if err != nil {
		t.Fatalf("BulkUnAssignRole: %v", err)
	}
	if unReport.GetAssignmentsRemoved() != 2 {
		t.Fatalf("expected 2 assignments removed, got %d", unReport.GetAssignmentsRemoved())
	}

	// List -> verify 0
	raList, err = env.client.Api.RoleAssignments.List(ctx, 1, 10, "", "", "")
	if err != nil {
		t.Fatalf("List role assignments after unassign: %v", err)
	}
	// SDK returns nil for empty arrays due to anyOf unmarshalling
	if raList != nil && len(*raList) != 0 {
		t.Fatalf("expected 0 role assignments, got %d", len(*raList))
	}
}

func TestRoleRemovePermissions(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create role, assign 3 permissions
	rc := *models.NewRoleCreate("editor", "Editor")
	_, err := env.client.Api.Roles.Create(ctx, rc)
	if err != nil {
		t.Fatalf("Create role: %v", err)
	}

	err = env.client.Api.Roles.AssignPermissions(ctx, "editor", []string{"doc:read", "doc:write", "doc:delete"})
	if err != nil {
		t.Fatalf("AssignPermissions: %v", err)
	}

	// Verify 3 permissions
	got, err := env.client.Api.Roles.Get(ctx, "editor")
	if err != nil {
		t.Fatalf("Get role: %v", err)
	}
	if len(got.Permissions) != 3 {
		t.Fatalf("expected 3 permissions, got %d: %v", len(got.Permissions), got.Permissions)
	}

	// Remove 1 permission
	err = env.client.Api.Roles.RemovePermissions(ctx, "editor", []string{"doc:delete"})
	if err != nil {
		t.Fatalf("RemovePermissions: %v", err)
	}

	// Verify 2 remain
	got, err = env.client.Api.Roles.Get(ctx, "editor")
	if err != nil {
		t.Fatalf("Get role after remove: %v", err)
	}
	if len(got.Permissions) != 2 {
		t.Fatalf("expected 2 permissions after remove, got %d: %v", len(got.Permissions), got.Permissions)
	}
}

func TestResourceRoleAssignRemovePermissions(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create resource + role
	actions := map[string]models.ActionBlockEditable{"read": {}, "write": {}}
	resCreate := *models.NewResourceCreate("doc", "Document", actions)
	_, err := env.client.Api.Resources.Create(ctx, resCreate)
	if err != nil {
		t.Fatalf("Create resource: %v", err)
	}

	rrc := *models.NewResourceRoleCreate("editor", "Editor")
	_, err = env.client.Api.ResourceRoles.Create(ctx, "doc", rrc)
	if err != nil {
		t.Fatalf("Create resource role: %v", err)
	}

	// Assign 2 permissions
	addPerms := *models.NewAddRolePermissions([]string{"doc:read", "doc:write"})
	result, err := env.client.Api.ResourceRoles.AssignPermissions(ctx, "doc", "editor", addPerms)
	if err != nil {
		t.Fatalf("AssignPermissions: %v", err)
	}
	if len(result.Permissions) != 2 {
		t.Fatalf("expected 2 permissions after assign, got %d: %v", len(result.Permissions), result.Permissions)
	}

	// Remove 1 permission
	removePerms := *models.NewRemoveRolePermissions([]string{"doc:write"})
	result, err = env.client.Api.ResourceRoles.RemovePermissions(ctx, "doc", "editor", removePerms)
	if err != nil {
		t.Fatalf("RemovePermissions: %v", err)
	}
	if len(result.Permissions) != 1 {
		t.Fatalf("expected 1 permission after remove, got %d: %v", len(result.Permissions), result.Permissions)
	}
	if result.Permissions[0] != "doc:read" {
		t.Fatalf("expected remaining permission 'doc:read', got %q", result.Permissions[0])
	}
}

func TestRelationshipTuplesBulkOperations(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup resources, relation, instances
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
		t.Fatalf("Create file instance 'readme': %v", err)
	}
	ic3 := *models.NewResourceInstanceCreate("notes", "file")
	ic3.SetTenant("default")
	_, err = env.client.Api.ResourceInstances.Create(ctx, ic3)
	if err != nil {
		t.Fatalf("Create file instance 'notes': %v", err)
	}

	// BulkCreate 2 tuples
	bulkCreate := *models.NewRelationshipTupleCreateBulkOperation([]models.RelationshipTupleCreate{
		*models.NewRelationshipTupleCreate("folder:docs", "parent", "file:readme"),
		*models.NewRelationshipTupleCreate("folder:docs", "parent", "file:notes"),
	})
	err = env.client.Api.RelationshipTuples.BulkCreate(ctx, bulkCreate)
	if err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}

	// List -> shows 2
	tuples, err := env.client.Api.RelationshipTuples.List(ctx, 1, 10, "", "", "", "")
	if err != nil {
		t.Fatalf("List tuples: %v", err)
	}
	if tuples == nil || len(*tuples) != 2 {
		t.Fatalf("expected 2 tuples, got %v", tuples)
	}

	// BulkDelete 2 tuples
	bulkDelete := *models.NewRelationshipTupleDeleteBulkOperation([]models.RelationshipTupleDelete{
		*models.NewRelationshipTupleDelete("folder:docs", "parent", "file:readme"),
		*models.NewRelationshipTupleDelete("folder:docs", "parent", "file:notes"),
	})
	err = env.client.Api.RelationshipTuples.BulkDelete(ctx, bulkDelete)
	if err != nil {
		t.Fatalf("BulkDelete: %v", err)
	}

	// List -> shows 0
	tuples, err = env.client.Api.RelationshipTuples.List(ctx, 1, 10, "", "", "", "")
	if err != nil {
		t.Fatalf("List tuples after delete: %v", err)
	}
	if tuples == nil || len(*tuples) != 0 {
		t.Fatalf("expected 0 tuples, got %d", len(*tuples))
	}
}

func TestResourceInstanceUpdate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Setup resource and tenant
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
	_, err = env.client.Api.ResourceInstances.Create(ctx, ic)
	if err != nil {
		t.Fatalf("Create resource instance: %v", err)
	}

	// Update attributes
	update := *models.NewResourceInstanceUpdate()
	update.SetAttributes(map[string]interface{}{"priority": "high", "size": float64(100)})
	updated, err := env.client.Api.ResourceInstances.Update(ctx, "folder:budget", update)
	if err != nil {
		t.Fatalf("Update resource instance: %v", err)
	}

	// Verify attributes
	attrs := updated.GetAttributes()
	if attrs["priority"] != "high" {
		t.Fatalf("expected attribute priority='high', got %v", attrs["priority"])
	}
}

func TestResourceUpdate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create resource
	actions := map[string]models.ActionBlockEditable{"read": {}, "write": {}}
	rc := *models.NewResourceCreate("document", "Document", actions)
	_, err := env.client.Api.Resources.Create(ctx, rc)
	if err != nil {
		t.Fatalf("Create resource: %v", err)
	}

	// Update name
	update := *models.NewResourceUpdate()
	update.SetName("Updated Document")
	updated, err := env.client.Api.Resources.Update(ctx, "document", update)
	if err != nil {
		t.Fatalf("Update resource: %v", err)
	}
	if updated.Name != "Updated Document" {
		t.Fatalf("expected name 'Updated Document', got %q", updated.Name)
	}

	// Verify with Get
	got, err := env.client.Api.Resources.Get(ctx, "document")
	if err != nil {
		t.Fatalf("Get resource: %v", err)
	}
	if got.Name != "Updated Document" {
		t.Fatalf("expected name 'Updated Document' after get, got %q", got.Name)
	}
}

func TestResourceRoleUpdate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create resource + role
	actions := map[string]models.ActionBlockEditable{"read": {}, "write": {}}
	resCreate := *models.NewResourceCreate("doc", "Document", actions)
	_, err := env.client.Api.Resources.Create(ctx, resCreate)
	if err != nil {
		t.Fatalf("Create resource: %v", err)
	}

	rrc := *models.NewResourceRoleCreate("editor", "Editor")
	_, err = env.client.Api.ResourceRoles.Create(ctx, "doc", rrc)
	if err != nil {
		t.Fatalf("Create resource role: %v", err)
	}

	// Update name
	update := *models.NewResourceRoleUpdate()
	update.SetName("Senior Editor")
	updated, err := env.client.Api.ResourceRoles.Update(ctx, "doc", "editor", update)
	if err != nil {
		t.Fatalf("Update resource role: %v", err)
	}
	if updated.Name != "Senior Editor" {
		t.Fatalf("expected name 'Senior Editor', got %q", updated.Name)
	}

	// Verify with Get
	got, err := env.client.Api.ResourceRoles.Get(ctx, "doc", "editor")
	if err != nil {
		t.Fatalf("Get resource role: %v", err)
	}
	if got.Name != "Senior Editor" {
		t.Fatalf("expected name 'Senior Editor' after get, got %q", got.Name)
	}
}

// containsStr checks if a string slice contains a specific string.
func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
