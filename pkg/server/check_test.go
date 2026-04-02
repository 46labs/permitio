package server

import (
	"context"
	"testing"

	"github.com/permitio/permit-golang/pkg/enforcement"
	"github.com/permitio/permit-golang/pkg/models"
)

// seedReBAC creates a full ReBAC policy for testing:
//
//	Resource "folder" with actions [read, write], role "owner" with [folder:read, folder:write]
//	Resource "document" with actions [read, edit], role "editor" with [document:read, document:edit], "viewer" with [document:read]
//	Relation: document has "parent" relation with subject_resource=folder
//	Implicit grant: folder#owner -> document#editor via parent
//	Tenant "default"
//	Users: user-1, user-2, user-nobody
//	Instances: folder:budget, document:report
//	Tuple: folder:budget --parent--> document:report
//	Role assignments: user-1 is owner of folder:budget, user-2 is viewer of folder:budget
func seedReBAC(t *testing.T, env *testEnv) {
	t.Helper()
	ctx := context.Background()

	// Create resources
	folderActions := map[string]models.ActionBlockEditable{"read": {}, "write": {}}
	folderRes := *models.NewResourceCreate("folder", "Folder", folderActions)
	_, err := env.client.Api.Resources.Create(ctx, folderRes)
	if err != nil {
		t.Fatalf("Create folder resource: %v", err)
	}

	docActions := map[string]models.ActionBlockEditable{"read": {}, "edit": {}}
	docRes := *models.NewResourceCreate("document", "Document", docActions)
	_, err = env.client.Api.Resources.Create(ctx, docRes)
	if err != nil {
		t.Fatalf("Create document resource: %v", err)
	}

	// Create resource roles
	ownerRole := *models.NewResourceRoleCreate("owner", "Owner")
	ownerRole.SetPermissions([]string{"folder:read", "folder:write"})
	_, err = env.client.Api.ResourceRoles.Create(ctx, "folder", ownerRole)
	if err != nil {
		t.Fatalf("Create folder owner role: %v", err)
	}

	viewerRole := *models.NewResourceRoleCreate("viewer", "Viewer")
	viewerRole.SetPermissions([]string{"folder:read"})
	_, err = env.client.Api.ResourceRoles.Create(ctx, "folder", viewerRole)
	if err != nil {
		t.Fatalf("Create folder viewer role: %v", err)
	}

	editorRole := *models.NewResourceRoleCreate("editor", "Editor")
	editorRole.SetPermissions([]string{"document:read", "document:edit"})
	_, err = env.client.Api.ResourceRoles.Create(ctx, "document", editorRole)
	if err != nil {
		t.Fatalf("Create document editor role: %v", err)
	}

	docViewerRole := *models.NewResourceRoleCreate("viewer", "Viewer")
	docViewerRole.SetPermissions([]string{"document:read"})
	_, err = env.client.Api.ResourceRoles.Create(ctx, "document", docViewerRole)
	if err != nil {
		t.Fatalf("Create document viewer role: %v", err)
	}

	// Create relation: document has "parent" relation to folder
	relCreate := *models.NewRelationCreate("parent", "Parent", "folder")
	_, err = env.client.Api.ResourceRelations.Create(ctx, "document", relCreate)
	if err != nil {
		t.Fatalf("Create parent relation: %v", err)
	}

	// Create implicit grant: folder#owner -> document#editor via parent
	// Created at resources/document/roles/editor/implicit_grants
	// Body: {role: "owner", on_resource: "folder", linked_by_relation: "parent"}
	igCreate := *models.NewDerivedRoleRuleCreate("owner", "folder", "parent")
	_, err = env.client.Api.ImplicitGrants.Create(ctx, "document", "editor", igCreate)
	if err != nil {
		t.Fatalf("Create implicit grant: %v", err)
	}

	// Create tenant
	tenantCreate := *models.NewTenantCreate("default", "Default Tenant")
	_, err = env.client.Api.Tenants.Create(ctx, tenantCreate)
	if err != nil {
		t.Fatalf("Create tenant: %v", err)
	}

	// Create users
	for _, key := range []string{"user-1", "user-2", "user-nobody"} {
		uc := *models.NewUserCreate(key)
		_, err = env.client.Api.Users.Create(ctx, uc)
		if err != nil {
			t.Fatalf("Create user %s: %v", key, err)
		}
	}

	// Create resource instances
	folderInst := *models.NewResourceInstanceCreate("budget", "folder")
	folderInst.SetTenant("default")
	_, err = env.client.Api.ResourceInstances.Create(ctx, folderInst)
	if err != nil {
		t.Fatalf("Create folder:budget instance: %v", err)
	}

	docInst := *models.NewResourceInstanceCreate("report", "document")
	docInst.SetTenant("default")
	_, err = env.client.Api.ResourceInstances.Create(ctx, docInst)
	if err != nil {
		t.Fatalf("Create document:report instance: %v", err)
	}

	// Create relationship tuple: folder:budget --parent--> document:report
	rtCreate := *models.NewRelationshipTupleCreate("folder:budget", "parent", "document:report")
	_, err = env.client.Api.RelationshipTuples.Create(ctx, rtCreate)
	if err != nil {
		t.Fatalf("Create relationship tuple: %v", err)
	}

	// Role assignments: user-1 is owner of folder:budget, user-2 is viewer of folder:budget
	_, err = env.client.Api.Users.AssignResourceRole(ctx, "user-1", "owner", "default", "folder:budget")
	if err != nil {
		t.Fatalf("Assign user-1 owner of folder:budget: %v", err)
	}

	_, err = env.client.Api.Users.AssignResourceRole(ctx, "user-2", "viewer", "default", "folder:budget")
	if err != nil {
		t.Fatalf("Assign user-2 viewer of folder:budget: %v", err)
	}
}

func TestReBACEnforcement(t *testing.T) {
	env := setupTestEnv(t)
	seedReBAC(t, env)

	t.Run("DirectPermission_Allow_Read", func(t *testing.T) {
		user := enforcement.UserBuilder("user-1").Build()
		resource := enforcement.ResourceBuilder("folder").WithKey("budget").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "read", resource)
		if err != nil {
			t.Fatalf("Check error: %v", err)
		}
		if !allowed {
			t.Fatal("expected user-1 to be allowed to read folder:budget (direct owner)")
		}
	})

	t.Run("DirectPermission_Allow_Write", func(t *testing.T) {
		user := enforcement.UserBuilder("user-1").Build()
		resource := enforcement.ResourceBuilder("folder").WithKey("budget").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "write", resource)
		if err != nil {
			t.Fatalf("Check error: %v", err)
		}
		if !allowed {
			t.Fatal("expected user-1 to be allowed to write folder:budget (direct owner)")
		}
	})

	t.Run("DerivedPermission_Allow_Edit", func(t *testing.T) {
		user := enforcement.UserBuilder("user-1").Build()
		resource := enforcement.ResourceBuilder("document").WithKey("report").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "edit", resource)
		if err != nil {
			t.Fatalf("Check error: %v", err)
		}
		if !allowed {
			t.Fatal("expected user-1 to be allowed to edit document:report (derived via folder owner -> document editor via parent)")
		}
	})

	t.Run("DerivedPermission_Allow_Read", func(t *testing.T) {
		user := enforcement.UserBuilder("user-1").Build()
		resource := enforcement.ResourceBuilder("document").WithKey("report").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "read", resource)
		if err != nil {
			t.Fatalf("Check error: %v", err)
		}
		if !allowed {
			t.Fatal("expected user-1 to be allowed to read document:report (derived via editor)")
		}
	})

	t.Run("NoAssignment_Deny", func(t *testing.T) {
		user := enforcement.UserBuilder("user-nobody").Build()
		resource := enforcement.ResourceBuilder("document").WithKey("report").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "edit", resource)
		if err != nil {
			t.Fatalf("Check error: %v", err)
		}
		if allowed {
			t.Fatal("expected user-nobody to be denied editing document:report (no assignment)")
		}
	})

	t.Run("InsufficientPermission_Deny", func(t *testing.T) {
		user := enforcement.UserBuilder("user-2").Build()
		resource := enforcement.ResourceBuilder("folder").WithKey("budget").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "write", resource)
		if err != nil {
			t.Fatalf("Check error: %v", err)
		}
		if allowed {
			t.Fatal("expected user-2 (viewer) to be denied writing folder:budget")
		}
	})

	t.Run("ViewerNoDerived_Deny", func(t *testing.T) {
		// user-2 is viewer of folder:budget - there's no implicit grant for viewer -> document
		user := enforcement.UserBuilder("user-2").Build()
		resource := enforcement.ResourceBuilder("document").WithKey("report").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "edit", resource)
		if err != nil {
			t.Fatalf("Check error: %v", err)
		}
		if allowed {
			t.Fatal("expected user-2 (viewer, no implicit grant) to be denied editing document:report")
		}
	})

	t.Run("ViewerCanReadFolder", func(t *testing.T) {
		user := enforcement.UserBuilder("user-2").Build()
		resource := enforcement.ResourceBuilder("folder").WithKey("budget").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "read", resource)
		if err != nil {
			t.Fatalf("Check error: %v", err)
		}
		if !allowed {
			t.Fatal("expected user-2 (viewer) to be allowed to read folder:budget")
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
				User:     enforcement.UserBuilder("user-nobody").Build(),
				Action:   "edit",
				Resource: enforcement.ResourceBuilder("document").WithKey("report").WithTenant("default").Build(),
			},
			{
				User:     enforcement.UserBuilder("user-1").Build(),
				Action:   "edit",
				Resource: enforcement.ResourceBuilder("document").WithKey("report").WithTenant("default").Build(),
			},
		}
		results, err := env.client.BulkCheck(checks...)
		if err != nil {
			t.Fatalf("BulkCheck error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		// user-1 can read folder:budget
		if !results[0] {
			t.Error("expected results[0] (user-1 read folder:budget) to be true")
		}
		// user-nobody cannot edit document:report
		if results[1] {
			t.Error("expected results[1] (user-nobody edit document:report) to be false")
		}
		// user-1 can edit document:report (derived)
		if !results[2] {
			t.Error("expected results[2] (user-1 edit document:report) to be true")
		}
	})

	t.Run("UserPermissions", func(t *testing.T) {
		user := enforcement.UserBuilder("user-1").Build()
		perms, err := env.client.GetUserPermissions(user, "default")
		if err != nil {
			t.Fatalf("GetUserPermissions error: %v", err)
		}

		tenantPerms, ok := perms["__tenant:default"]
		if !ok {
			t.Fatalf("expected __tenant:default in permissions, got keys: %v", keysOf(perms))
		}

		if tenantPerms.Tenant.Key != "default" {
			t.Errorf("expected tenant key 'default', got %q", tenantPerms.Tenant.Key)
		}

		// user-1 has: folder:read, folder:write (direct), document:read, document:edit (derived)
		expectedPerms := map[string]bool{
			"folder:read":   true,
			"folder:write":  true,
			"document:read": true,
			"document:edit": true,
		}
		for _, p := range tenantPerms.Permissions {
			delete(expectedPerms, p)
		}
		if len(expectedPerms) > 0 {
			t.Errorf("missing permissions: %v, got: %v", expectedPerms, tenantPerms.Permissions)
		}

		// user-1 should have roles
		if len(tenantPerms.Roles) == 0 {
			t.Error("expected user-1 to have roles")
		}
	})

	t.Run("AllTenantsCheck", func(t *testing.T) {
		user := enforcement.UserBuilder("user-1").Build()
		resource := enforcement.ResourceBuilder("folder").WithKey("budget").Build()
		tenants, err := env.client.AllTenantsCheck(user, "read", resource)
		if err != nil {
			t.Fatalf("AllTenantsCheck error: %v", err)
		}
		if len(tenants) == 0 {
			t.Fatal("expected at least one allowed tenant")
		}
		found := false
		for _, td := range tenants {
			if td.Key == "default" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'default' tenant in allowed tenants")
		}
	})
}

func TestAllowAllMode(t *testing.T) {
	env := setupTestEnv(t)

	// Enable allow-all mode
	env.store.SetAllowAll(true)

	// Create tenant and user so we have valid data
	ctx := context.Background()
	tc := *models.NewTenantCreate("default", "Default")
	_, _ = env.client.Api.Tenants.Create(ctx, tc)
	uc := *models.NewUserCreate("anyone")
	_, _ = env.client.Api.Users.Create(ctx, uc)

	t.Run("AllowAll_PermitsEverything", func(t *testing.T) {
		user := enforcement.UserBuilder("anyone").Build()
		resource := enforcement.ResourceBuilder("anything").WithKey("whatever").WithTenant("default").Build()
		allowed, err := env.client.Check(user, "do-stuff", resource)
		if err != nil {
			t.Fatalf("Check error: %v", err)
		}
		if !allowed {
			t.Fatal("expected allow-all mode to permit everything")
		}
	})

	t.Run("AllowAll_BulkCheck", func(t *testing.T) {
		checks := []enforcement.CheckRequest{
			{
				User:     enforcement.UserBuilder("anyone").Build(),
				Action:   "read",
				Resource: enforcement.ResourceBuilder("x").WithKey("y").WithTenant("default").Build(),
			},
			{
				User:     enforcement.UserBuilder("nobody").Build(),
				Action:   "write",
				Resource: enforcement.ResourceBuilder("a").WithKey("b").WithTenant("default").Build(),
			},
		}
		results, err := env.client.BulkCheck(checks...)
		if err != nil {
			t.Fatalf("BulkCheck error: %v", err)
		}
		for i, r := range results {
			if !r {
				t.Errorf("expected results[%d] to be true in allow-all mode", i)
			}
		}
	})
}

func keysOf(m enforcement.UserPermissions) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
