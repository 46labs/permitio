package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/permitio/permit-golang/pkg/models"
)

// Lists page as Permit pages them, through the official SDK: every page
// distinct, together the whole set, a page past the end empty, and filters
// narrowing before paging.
func TestSDKListsPage(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	api := env.client.Api

	if _, err := api.Tenants.Create(ctx, *models.NewTenantCreate("default", "Default")); err != nil {
		t.Fatal(err)
	}
	actions := map[string]models.ActionBlockEditable{"read": {}}
	for _, r := range []string{"folder", "file"} {
		if _, err := api.Resources.Create(ctx, *models.NewResourceCreate(r, r, actions)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := api.ResourceRoles.Create(ctx, "file", *models.NewResourceRoleCreate("reader", "Reader")); err != nil {
		t.Fatal(err)
	}
	if _, err := api.ResourceRelations.Create(ctx, "file", *models.NewRelationCreate("parent", "Parent", "folder")); err != nil {
		t.Fatal(err)
	}
	folder := *models.NewResourceInstanceCreate("docs", "folder")
	folder.SetTenant("default")
	if _, err := api.ResourceInstances.Create(ctx, folder); err != nil {
		t.Fatal(err)
	}
	const n = 7
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("u%d", i)
		if _, err := api.Users.Create(ctx, *models.NewUserCreate(key)); err != nil {
			t.Fatal(err)
		}
		file := *models.NewResourceInstanceCreate(fmt.Sprintf("f%d", i), "file")
		file.SetTenant("default")
		if _, err := api.ResourceInstances.Create(ctx, file); err != nil {
			t.Fatal(err)
		}
		if _, err := api.Users.AssignResourceRole(ctx, key, "reader", "default", fmt.Sprintf("file:f%d", i)); err != nil {
			t.Fatal(err)
		}
		tenant := "default"
		if _, err := api.RelationshipTuples.Create(ctx, models.RelationshipTupleCreate{
			Subject: "folder:docs", Relation: "parent", Object: fmt.Sprintf("file:f%d", i), Tenant: &tenant,
		}); err != nil {
			t.Fatal(err)
		}
	}

	// pageAll walks pages of 3 and checks they are distinct and complete.
	pageAll := func(t *testing.T, name string, want int, list func(page int) ([]string, error)) {
		t.Helper()
		seen := map[string]bool{}
		for page := 1; page <= 4; page++ {
			keys, err := list(page)
			if err != nil {
				t.Fatalf("%s page %d: %v", name, page, err)
			}
			wantLen := min(3, max(0, want-(page-1)*3))
			if len(keys) != wantLen {
				t.Fatalf("%s page %d: %d items, want %d", name, page, len(keys), wantLen)
			}
			for _, k := range keys {
				if seen[k] {
					t.Fatalf("%s: %s on two pages", name, k)
				}
				seen[k] = true
			}
		}
		if len(seen) != want {
			t.Fatalf("%s: %d across pages, want %d", name, len(seen), want)
		}
	}

	pageAll(t, "users", n, func(page int) ([]string, error) {
		users, err := api.Users.List(ctx, page, 3)
		var keys []string
		for _, u := range users {
			keys = append(keys, u.Key)
		}
		return keys, err
	})
	pageAll(t, "file instances", n, func(page int) ([]string, error) {
		list, err := api.ResourceInstances.List(ctx, page, 3, "default", "file", "")
		var keys []string
		if list != nil {
			for _, i := range *list {
				if i.Resource != "file" {
					t.Fatalf("resource filter let %s:%s through", i.Resource, i.Key)
				}
				keys = append(keys, i.Key)
			}
		}
		return keys, err
	})
	pageAll(t, "role assignments", n, func(page int) ([]string, error) {
		list, err := api.RoleAssignments.List(ctx, page, 3, "", "", "default")
		var keys []string
		if list != nil {
			for _, r := range *list {
				keys = append(keys, r.User+"/"+*r.ResourceInstance)
			}
		}
		return keys, err
	})
	pageAll(t, "relationship tuples", n, func(page int) ([]string, error) {
		list, err := api.RelationshipTuples.List(ctx, page, 3, "default", "", "parent", "")
		var keys []string
		if list != nil {
			for _, r := range *list {
				keys = append(keys, r.Object)
			}
		}
		return keys, err
	})

	// Filters narrow before paging.
	one, err := api.RelationshipTuples.List(ctx, 1, 100, "default", "", "", "file:f3")
	if err != nil || one == nil || len(*one) != 1 || (*one)[0].Object != "file:f3" {
		t.Fatalf("object filter: %v %v", one, err)
	}
	folders, err := api.ResourceInstances.List(ctx, 1, 100, "default", "folder", "")
	if err != nil || folders == nil || len(*folders) != 1 {
		t.Fatalf("resource filter: %v %v", folders, err)
	}
}

// An implicit grant created twice is one grant, the role read shows it under
// granted_to, and deleting it removes it from the role it was granted on only.
func TestSDKImplicitGrantIsIdempotentAndReadable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	api := env.client.Api

	actions := map[string]models.ActionBlockEditable{"read": {}}
	for _, r := range []string{"folder", "file"} {
		if _, err := api.Resources.Create(ctx, *models.NewResourceCreate(r, r, actions)); err != nil {
			t.Fatal(err)
		}
		if _, err := api.ResourceRoles.Create(ctx, r, *models.NewResourceRoleCreate("viewer", "Viewer")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := api.ResourceRelations.Create(ctx, "file", *models.NewRelationCreate("parent", "Parent", "folder")); err != nil {
		t.Fatal(err)
	}
	rule := models.DerivedRoleRuleCreate{Role: "viewer", OnResource: "folder", LinkedByRelation: "parent"}
	for i := 0; i < 2; i++ {
		if _, err := api.ImplicitGrants.Create(ctx, "file", "viewer", rule); err != nil {
			t.Fatalf("create %d: %v", i+1, err)
		}
	}

	role, err := api.ResourceRoles.Get(ctx, "file", "viewer")
	if err != nil {
		t.Fatal(err)
	}
	if role.GrantedTo == nil || len(role.GrantedTo.UsersWithRole) != 1 {
		t.Fatalf("granted_to: %+v", role.GrantedTo)
	}
	got := role.GrantedTo.UsersWithRole[0]
	if got.Role != "viewer" || got.OnResource != "folder" || got.LinkedByRelation != "parent" {
		t.Fatalf("rule: %+v", got)
	}
	if other, err := api.ResourceRoles.Get(ctx, "folder", "viewer"); err != nil || other.GrantedTo != nil {
		t.Fatalf("folder#viewer should derive from nothing: %+v %v", other, err)
	}

	// permit-golang v1.3.0 names Delete's arguments (roleId, resourceId) but
	// hands them to the generated client the other way round, so the path it
	// builds is resources/<first>/roles/<second>: the resource goes first.
	if err := api.ImplicitGrants.Delete(ctx, "file", "viewer", models.DerivedRoleRuleDelete{
		Role: "viewer", OnResource: "folder", LinkedByRelation: "parent",
	}); err != nil {
		t.Fatal(err)
	}
	if role, err := api.ResourceRoles.Get(ctx, "file", "viewer"); err != nil || role.GrantedTo != nil {
		t.Fatalf("after delete: %+v %v", role, err)
	}
}
