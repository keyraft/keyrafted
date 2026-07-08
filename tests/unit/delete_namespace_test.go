package unit

import (
	"keyrafted/internal/crypto"
	"keyrafted/internal/engine"
	"keyrafted/internal/models"
	"keyrafted/internal/storage"
	"path/filepath"
	"testing"
)

func TestDeleteNamespace(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewBoltDBStorage(filepath.Join(tempDir, "test.db"))
	if err := store.Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	enc, err := crypto.NewEncryptor([]byte("test-master-key-32-bytes-long!!!"))
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	eng := engine.NewEngine(store, enc)

	if _, err := eng.Set("test", "A", "one", models.TypeConfig, nil); err != nil {
		t.Fatalf("set test/A: %v", err)
	}
	if _, err := eng.Set("test/child", "B", "two", models.TypeConfig, nil); err != nil {
		t.Fatalf("set test/child/B: %v", err)
	}

	keys, err := eng.DeleteNamespace("test")
	if err != nil {
		t.Fatalf("delete test: %v", err)
	}
	if len(keys) != 1 || keys[0] != "A" {
		t.Fatalf("deleted keys = %v, want [A]", keys)
	}

	if _, err := eng.Get("test", "A"); err == nil {
		t.Fatal("test/A should be gone")
	}
	if _, err := eng.GetNamespace("test"); err == nil {
		t.Fatal("test namespace should be gone")
	}
	if _, err := eng.Get("test/child", "B"); err != nil {
		t.Fatalf("test/child/B should remain: %v", err)
	}
}
