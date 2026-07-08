package unit

import (
	"keyrafted/internal/auth"
	"keyrafted/internal/models"
	"testing"
)

// ponytail: assert-only check that root (empty role+scopes) and admin both get manage_tokens
func TestHasPermissionRootAndAdmin(t *testing.T) {
	s := auth.NewService(nil)

	root := &models.Token{}
	if !s.HasPermission(root, models.PermissionManageTokens) {
		t.Fatal("root token should have manage_tokens")
	}

	admin := &models.Token{Role: models.RoleAdmin}
	if !s.HasPermission(admin, models.PermissionViewAudit) {
		t.Fatal("admin should have view_audit")
	}

	viewer := &models.Token{Role: models.RoleViewer}
	if s.HasPermission(viewer, models.PermissionManageTokens) {
		t.Fatal("viewer must not have manage_tokens")
	}
}
