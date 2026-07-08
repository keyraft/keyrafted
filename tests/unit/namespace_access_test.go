package unit

import (
	"keyrafted/internal/auth"
	"keyrafted/internal/models"
	"testing"
)

func TestScopedTokenNamespaceAccess(t *testing.T) {
	svc := auth.NewService(nil)

	scoped := &models.Token{
		Scopes: []models.TokenScope{
			{Namespace: "demo/*", Read: true, Write: false},
		},
	}

	if !svc.HasAccess(scoped, "demo/staging", false) {
		t.Fatal("demo/staging should be readable")
	}
	if !svc.HasAccess(scoped, "demo/staging/test", false) {
		t.Fatal("demo/staging/test should be readable under demo/*")
	}
	if svc.HasAccess(scoped, "test", false) {
		t.Fatal("test should not be readable")
	}
	if svc.HasAccess(scoped, "demo/staging", true) {
		t.Fatal("write should be denied")
	}

	exact := &models.Token{
		Scopes: []models.TokenScope{
			{Namespace: "abc/random", Read: true, Write: true},
		},
	}
	if !svc.HasAccess(exact, "abc/random", true) {
		t.Fatal("exact scope should allow")
	}
	if svc.HasAccess(exact, "abc/other", false) {
		t.Fatal("exact scope should not leak")
	}
}
