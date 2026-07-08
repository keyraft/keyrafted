package unit

import (
	"keyrafted/internal/crypto"
	"keyrafted/internal/engine"
	"keyrafted/internal/models"
	"keyrafted/internal/storage"
	"path/filepath"
	"testing"
)

// Nested namespace listing must not 404 as "key not found" when a sibling-looking path segment exists.
func TestNestedNamespaceExistsAlongsideParent(t *testing.T) {
	store := storage.NewBoltDBStorage(filepath.Join(t.TempDir(), "ns.db"))
	if err := store.Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	enc, err := crypto.NewEncryptor([]byte("test-master-key-32-bytes-long!!!"))
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	eng := engine.NewEngine(store, enc)

	if _, err := eng.Set("demo/staging", "APP_ENV", "prod", models.TypeConfig, nil); err != nil {
		t.Fatalf("set parent: %v", err)
	}
	if _, err := eng.Set("demo/staging/test", "FOO", "bar", models.TypeConfig, nil); err != nil {
		t.Fatalf("set child: %v", err)
	}

	if _, err := eng.GetNamespace("demo/staging/test"); err != nil {
		t.Fatalf("child namespace missing: %v", err)
	}
	if _, err := eng.Get("demo/staging", "test"); err == nil {
		t.Fatal("demo/staging/test should not also be a key named test")
	}

	keys, err := eng.List("demo/staging/test")
	if err != nil {
		t.Fatalf("list child: %v", err)
	}
	found := false
	for _, k := range keys {
		if k.Namespace == "demo/staging/test" && k.Key == "FOO" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected FOO in demo/staging/test, got %+v", keys)
	}
}
