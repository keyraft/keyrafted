package integration

import (
	"context"
	"fmt"
	"keyrafted/internal/auth"
	"keyrafted/internal/crypto"
	"keyrafted/internal/engine"
	"keyrafted/internal/models"
	"keyrafted/internal/storage"
	"keyrafted/internal/watch"
	"keyrafted/pkg/client"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStorageOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "keyraft-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	dbPath := filepath.Join(tempDir, "test.db")
	store := storage.NewBoltDBStorage(dbPath)

	if err := store.Open(); err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	// Test KV operations
	entry := &models.KVEntry{
		Namespace: "test/prod",
		Key:       "DB_HOST",
		Value:     "localhost",
		Type:      models.TypeConfig,
		Metadata:  map[string]string{"env": "prod"},
	}

	// Set
	if err := store.Set(entry); err != nil {
		t.Fatalf("Failed to set entry: %v", err)
	}

	// Get
	retrieved, err := store.Get("test/prod", "DB_HOST")
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}

	if retrieved.Value != "localhost" {
		t.Errorf("Expected value 'localhost', got '%s'", retrieved.Value)
	}

	if retrieved.Version != 1 {
		t.Errorf("Expected version 1, got %d", retrieved.Version)
	}

	// Update
	entry.Value = "db.example.com"
	if err := store.Set(entry); err != nil {
		t.Fatalf("Failed to update entry: %v", err)
	}

	updated, err := store.Get("test/prod", "DB_HOST")
	if err != nil {
		t.Fatalf("Failed to get updated entry: %v", err)
	}

	if updated.Version != 2 {
		t.Errorf("Expected version 2, got %d", updated.Version)
	}

	// List
	entries, err := store.List("test/prod")
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	// Delete
	if err := store.Delete("test/prod", "DB_HOST"); err != nil {
		t.Fatalf("Failed to delete entry: %v", err)
	}

	// Verify deleted
	_, err = store.Get("test/prod", "DB_HOST")
	if err == nil {
		t.Error("Expected error when getting deleted entry")
	}
}

func TestEncryptionIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "keyraft-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	dbPath := filepath.Join(tempDir, "test.db")
	store := storage.NewBoltDBStorage(dbPath)

	if err := store.Open(); err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	encryptor, err := crypto.NewEncryptor([]byte("test-master-key-32-bytes-long!!!"))
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	eng := engine.NewEngine(store, encryptor)

	// Store a secret
	secretValue := "super-secret-password"
	entry, err := eng.Set("app/prod", "API_KEY", secretValue, models.TypeSecret, nil)
	if err != nil {
		t.Fatalf("Failed to set secret: %v", err)
	}

	// Value in response should be decrypted
	if entry.Value != secretValue {
		t.Errorf("Expected decrypted value in response")
	}

	// Retrieve the secret
	retrieved, err := eng.Get("app/prod", "API_KEY")
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if retrieved.Value != secretValue {
		t.Errorf("Expected '%s', got '%s'", secretValue, retrieved.Value)
	}

	// Verify it's encrypted in storage
	storedEntry, err := store.Get("app/prod", "API_KEY")
	if err != nil {
		t.Fatalf("Failed to get from storage: %v", err)
	}

	if storedEntry.Value == secretValue {
		t.Error("Secret should be encrypted in storage")
	}
}

func TestAuthenticationFlow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "keyraft-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	dbPath := filepath.Join(tempDir, "test.db")
	store := storage.NewBoltDBStorage(dbPath)

	if err := store.Open(); err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	authSvc := auth.NewService(store)

	// Create root token
	rootToken, err := authSvc.InitializeRootToken()
	if err != nil {
		t.Fatalf("Failed to create root token: %v", err)
	}

	// Validate root token
	validated, err := authSvc.ValidateToken(rootToken.Token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// Root token should have full access
	if !authSvc.HasAccess(validated, "any/namespace", true) {
		t.Error("Root token should have full access")
	}

	// Create scoped token
	scopes := []models.TokenScope{
		{Namespace: "app/prod", Read: true, Write: false},
	}

	scopedToken, err := authSvc.GenerateToken(scopes, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create scoped token: %v", err)
	}

	// Validate scoped token
	validatedScoped, err := authSvc.ValidateToken(scopedToken.Token)
	if err != nil {
		t.Fatalf("Failed to validate scoped token: %v", err)
	}

	// Should have read access
	if !authSvc.HasAccess(validatedScoped, "app/prod", false) {
		t.Error("Should have read access to app/prod")
	}

	// Should not have write access
	if authSvc.HasAccess(validatedScoped, "app/prod", true) {
		t.Error("Should not have write access to app/prod")
	}

	// Should not have access to other namespaces
	if authSvc.HasAccess(validatedScoped, "other/namespace", false) {
		t.Error("Should not have access to other namespaces")
	}
}

func TestVersioning(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "keyraft-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	dbPath := filepath.Join(tempDir, "test.db")
	store := storage.NewBoltDBStorage(dbPath)

	if err := store.Open(); err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	encryptor, err := crypto.NewEncryptor([]byte("test-master-key-32-bytes-long!!!"))
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	eng := engine.NewEngine(store, encryptor)

	// Create multiple versions
	for i := 1; i <= 5; i++ {
		value := fmt.Sprintf("value-%d", i)
		_, err := eng.Set("app/prod", "CONFIG", value, models.TypeConfig, nil)
		if err != nil {
			t.Fatalf("Failed to set version %d: %v", i, err)
		}
	}

	// Get latest version
	latest, err := eng.Get("app/prod", "CONFIG")
	if err != nil {
		t.Fatalf("Failed to get latest: %v", err)
	}

	if latest.Value != "value-5" || latest.Version != 5 {
		t.Errorf("Expected value-5 version 5, got %s version %d", latest.Value, latest.Version)
	}

	// Get specific version
	v2, err := eng.GetVersion("app/prod", "CONFIG", 2)
	if err != nil {
		t.Fatalf("Failed to get version 2: %v", err)
	}

	if v2.Value != "value-2" || v2.Version != 2 {
		t.Errorf("Expected value-2 version 2, got %s version %d", v2.Value, v2.Version)
	}

	// Get all versions
	versions, err := eng.GetVersions("app/prod", "CONFIG")
	if err != nil {
		t.Fatalf("Failed to get versions: %v", err)
	}

	if len(versions) != 5 {
		t.Errorf("Expected 5 versions, got %d", len(versions))
	}
}

func TestWatchManager(t *testing.T) {
	watchMgr := watch.NewManager()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create watcher
	watcher := watchMgr.Watch(ctx, "app/prod", 10)

	// Notify of change
	go func() {
		time.Sleep(100 * time.Millisecond)
		watchMgr.Notify(watch.Event{
			Action:    "set",
			Namespace: "app/prod",
			Key:       "TEST_KEY",
		})
	}()

	// Wait for event
	select {
	case event := <-watcher.Events:
		if event.Action != "set" || event.Key != "TEST_KEY" {
			t.Errorf("Unexpected event: %+v", event)
		}
	case <-ctx.Done():
		t.Error("Timeout waiting for event")
	}

	// Cleanup
	watchMgr.Unwatch(watcher.ID)
}

func TestClientSDK(t *testing.T) {
	// Note: This would require a running server
	// For now, we'll just test the client can be created
	config := client.Config{
		BaseURL: "http://localhost:7200",
		Token:   "test-token",
		Timeout: 5 * time.Second,
	}

	c := client.NewClient(config)
	if c == nil {
		t.Error("Failed to create client")
	}
}
