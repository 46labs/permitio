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
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "notfound") && !strings.Contains(errStr, "not found") {
		t.Fatalf("expected not-found error, got: %v", err)
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
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "conflict") && !strings.Contains(errStr, "already exists") {
		t.Fatalf("expected conflict error, got: %v", err)
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
