package unit

import (
	"keyrafted/internal/auth"
	"keyrafted/internal/models"
	"keyrafted/internal/storage"
	"path/filepath"
	"strings"
	"testing"
)

func TestCannotRevokeLastRootToken(t *testing.T) {
	store := storage.NewBoltDBStorage(filepath.Join(t.TempDir(), "tokens.db"))
	if err := store.Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	svc := auth.NewService(store)
	root, err := svc.InitializeRootToken()
	if err != nil {
		t.Fatalf("init root: %v", err)
	}

	err = svc.RevokeToken(root.Token)
	if err == nil || !strings.Contains(err.Error(), "last root token") {
		t.Fatalf("expected last-root error, got %v", err)
	}

	second, err := svc.GenerateToken(nil, models.RoleAdmin, nil, map[string]string{
		"name": "root-2",
		"type": "root",
	})
	if err != nil {
		t.Fatalf("second root: %v", err)
	}

	if err := svc.RevokeToken(root.Token); err != nil {
		t.Fatalf("should revoke when another root exists: %v", err)
	}

	err = svc.RevokeToken(second.Token)
	if err == nil || !strings.Contains(err.Error(), "last root token") {
		t.Fatalf("expected last-root error for second token, got %v", err)
	}
}

func TestIsRootToken(t *testing.T) {
	if !auth.IsRootToken(&models.Token{Role: models.RoleAdmin}) {
		t.Fatal("admin should be root")
	}
	if !auth.IsRootToken(&models.Token{Metadata: map[string]string{"type": "root"}}) {
		t.Fatal("metadata type=root should be root")
	}
	if auth.IsRootToken(&models.Token{Role: models.RoleViewer}) {
		t.Fatal("viewer should not be root")
	}
}
