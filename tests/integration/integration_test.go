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
	"path/filepath"
	"testing"
	"time"
)

func TestStorageOperations(t *testing.T) {
	tempDir := t.TempDir()
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
	tempDir := t.TempDir()
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
	tempDir := t.TempDir()
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

	scopedToken, err := authSvc.GenerateToken(scopes, "", nil, nil)
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
	tempDir := t.TempDir()
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

func TestSSEWatch(t *testing.T) {
	tempDir := t.TempDir()
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
	authSvc := auth.NewService(store)
	watchMgr := watch.NewManager()

	// Initialize root token
	_, err = authSvc.InitializeRootToken()
	if err != nil {
		t.Fatalf("Failed to create root token: %v", err)
	}

	// Create test server (would need actual HTTP server for full test)
	// This is a basic structure test
	if eng == nil || authSvc == nil || watchMgr == nil {
		t.Error("Failed to create required services")
	}

	// Test that watch manager can handle events
	entry := &models.KVEntry{
		Namespace: "test/prod",
		Key:       "TEST_KEY",
		Value:     "test-value",
		Type:      models.TypeConfig,
	}

	watchMgr.NotifySet(entry)
	if watchMgr.ActiveWatchers() != 0 {
		t.Log("Watch manager is working")
	}
}

func TestRBAC(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	store := storage.NewBoltDBStorage(dbPath)

	if err := store.Open(); err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	authSvc := auth.NewService(store)

	// Create root token (admin role)
	rootToken, err := authSvc.InitializeRootToken()
	if err != nil {
		t.Fatalf("Failed to create root token: %v", err)
	}

	validatedRoot, err := authSvc.ValidateToken(rootToken.Token)
	if err != nil {
		t.Fatalf("Failed to validate root token: %v", err)
	}

	// Root token should have admin role
	if validatedRoot.Role != models.RoleAdmin {
		t.Errorf("Expected root token to have admin role, got %s", validatedRoot.Role)
	}

	// Admin should have all permissions
	if !authSvc.HasPermission(validatedRoot, models.PermissionRead) {
		t.Error("Admin should have read permission")
	}
	if !authSvc.HasPermission(validatedRoot, models.PermissionWrite) {
		t.Error("Admin should have write permission")
	}
	if !authSvc.HasPermission(validatedRoot, models.PermissionManageTokens) {
		t.Error("Admin should have manage_tokens permission")
	}
	if !authSvc.HasPermission(validatedRoot, models.PermissionManageRoles) {
		t.Error("Admin should have manage_roles permission")
	}
	if !authSvc.HasPermission(validatedRoot, models.PermissionViewAudit) {
		t.Error("Admin should have view_audit permission")
	}

	// Test developer role
	developerToken, err := authSvc.GenerateToken([]models.TokenScope{}, models.RoleDeveloper, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create developer token: %v", err)
	}

	validatedDev, err := authSvc.ValidateToken(developerToken.Token)
	if err != nil {
		t.Fatalf("Failed to validate developer token: %v", err)
	}

	if validatedDev.Role != models.RoleDeveloper {
		t.Errorf("Expected developer role, got %s", validatedDev.Role)
	}

	// Developer should have read, write, delete
	if !authSvc.HasPermission(validatedDev, models.PermissionRead) {
		t.Error("Developer should have read permission")
	}
	if !authSvc.HasPermission(validatedDev, models.PermissionWrite) {
		t.Error("Developer should have write permission")
	}
	if !authSvc.HasPermission(validatedDev, models.PermissionDelete) {
		t.Error("Developer should have delete permission")
	}

	// Developer should NOT have admin permissions
	if authSvc.HasPermission(validatedDev, models.PermissionManageTokens) {
		t.Error("Developer should not have manage_tokens permission")
	}
	if authSvc.HasPermission(validatedDev, models.PermissionManageRoles) {
		t.Error("Developer should not have manage_roles permission")
	}

	// Test viewer role
	viewerToken, err := authSvc.GenerateToken([]models.TokenScope{}, models.RoleViewer, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create viewer token: %v", err)
	}

	validatedViewer, err := authSvc.ValidateToken(viewerToken.Token)
	if err != nil {
		t.Fatalf("Failed to validate viewer token: %v", err)
	}

	if validatedViewer.Role != models.RoleViewer {
		t.Errorf("Expected viewer role, got %s", validatedViewer.Role)
	}

	// Viewer should only have read permission
	if !authSvc.HasPermission(validatedViewer, models.PermissionRead) {
		t.Error("Viewer should have read permission")
	}
	if authSvc.HasPermission(validatedViewer, models.PermissionWrite) {
		t.Error("Viewer should not have write permission")
	}
	if authSvc.HasPermission(validatedViewer, models.PermissionDelete) {
		t.Error("Viewer should not have delete permission")
	}

	// Test operator role
	operatorToken, err := authSvc.GenerateToken([]models.TokenScope{}, models.RoleOperator, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create operator token: %v", err)
	}

	validatedOp, err := authSvc.ValidateToken(operatorToken.Token)
	if err != nil {
		t.Fatalf("Failed to validate operator token: %v", err)
	}

	if validatedOp.Role != models.RoleOperator {
		t.Errorf("Expected operator role, got %s", validatedOp.Role)
	}

	// Operator should have read, write, delete, view_audit
	if !authSvc.HasPermission(validatedOp, models.PermissionRead) {
		t.Error("Operator should have read permission")
	}
	if !authSvc.HasPermission(validatedOp, models.PermissionWrite) {
		t.Error("Operator should have write permission")
	}
	if !authSvc.HasPermission(validatedOp, models.PermissionDelete) {
		t.Error("Operator should have delete permission")
	}
	if !authSvc.HasPermission(validatedOp, models.PermissionViewAudit) {
		t.Error("Operator should have view_audit permission")
	}

	// Operator should NOT have manage permissions
	if authSvc.HasPermission(validatedOp, models.PermissionManageTokens) {
		t.Error("Operator should not have manage_tokens permission")
	}
	if authSvc.HasPermission(validatedOp, models.PermissionManageRoles) {
		t.Error("Operator should not have manage_roles permission")
	}

	// Test invalid role
	_, err = authSvc.GenerateToken([]models.TokenScope{}, "invalid_role", nil, nil)
	if err == nil {
		t.Error("Should fail when creating token with invalid role")
	}

	// Test backward compatibility: tokens with scopes should still work
	scopes := []models.TokenScope{
		{Namespace: "app/prod", Read: true, Write: false},
	}
	scopedToken, err := authSvc.GenerateToken(scopes, "", nil, nil)
	if err != nil {
		t.Fatalf("Failed to create scoped token: %v", err)
	}

	validatedScoped, err := authSvc.ValidateToken(scopedToken.Token)
	if err != nil {
		t.Fatalf("Failed to validate scoped token: %v", err)
	}

	// Scoped token should work with HasAccess (backward compatibility)
	if !authSvc.HasAccess(validatedScoped, "app/prod", false) {
		t.Error("Scoped token should have read access to app/prod")
	}
	if authSvc.HasAccess(validatedScoped, "app/prod", true) {
		t.Error("Scoped token should not have write access to app/prod")
	}

	// Test admin has full access regardless of namespace
	if !authSvc.HasAccess(validatedRoot, "any/namespace", true) {
		t.Error("Admin should have full access to any namespace")
	}
	if !authSvc.HasAccess(validatedRoot, "another/namespace", false) {
		t.Error("Admin should have read access to any namespace")
	}
}

func TestDefaultRoles(t *testing.T) {
	roles := models.GetDefaultRoles()

	// Check all expected roles exist
	expectedRoles := []string{models.RoleAdmin, models.RoleDeveloper, models.RoleViewer, models.RoleOperator}
	for _, roleName := range expectedRoles {
		if _, exists := roles[roleName]; !exists {
			t.Errorf("Expected role %s not found", roleName)
		}
	}

	// Verify admin has all permissions
	admin := roles[models.RoleAdmin]
	allPermissions := []string{
		models.PermissionRead,
		models.PermissionWrite,
		models.PermissionDelete,
		models.PermissionManageTokens,
		models.PermissionManageRoles,
		models.PermissionViewAudit,
		models.PermissionManageNamespaces,
	}

	for _, perm := range allPermissions {
		found := false
		for _, p := range admin.Permissions {
			if p == perm {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Admin role missing permission: %s", perm)
		}
	}

	// Verify viewer only has read
	viewer := roles[models.RoleViewer]
	if len(viewer.Permissions) != 1 || viewer.Permissions[0] != models.PermissionRead {
		t.Errorf("Viewer should only have read permission, got %v", viewer.Permissions)
	}
}

func TestWildcardScopeMatching(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	store := storage.NewBoltDBStorage(dbPath)

	if err := store.Open(); err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	authSvc := auth.NewService(store)

	// Initialize root token (required before creating other tokens)
	_, err := authSvc.InitializeRootToken()
	if err != nil {
		t.Fatalf("Failed to create root token: %v", err)
	}

	// Create token with wildcard scope: myapp/*
	scopes := []models.TokenScope{
		{Namespace: "myapp/*", Read: true, Write: false},
	}

	wildcardToken, err := authSvc.GenerateToken(scopes, "", nil, nil)
	if err != nil {
		t.Fatalf("Failed to create wildcard token: %v", err)
	}

	validated, err := authSvc.ValidateToken(wildcardToken.Token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// Test cases: myapp/* should match various myapp namespaces
	testCases := []struct {
		namespace string
		write     bool
		expected  bool
		desc      string
	}{
		{"myapp/prod", false, true, "read access to myapp/prod"},
		{"myapp/dev", false, true, "read access to myapp/dev"},
		{"myapp/prod/api", false, true, "read access to myapp/prod/api"},
		{"myapp", false, true, "read access to myapp"},
		{"myapp/prod", true, false, "write access to myapp/prod (should be false)"},
		{"myappx/prod", false, false, "read access to myappx/prod (should not match)"},
		{"other/prod", false, false, "read access to other/prod (should not match)"},
		{"myappx", false, false, "read access to myappx (should not match)"},
	}

	for _, tc := range testCases {
		result := authSvc.HasAccess(validated, tc.namespace, tc.write)
		if result != tc.expected {
			t.Errorf("%s: expected %v, got %v", tc.desc, tc.expected, result)
		}
	}

	// Test write access with wildcard
	writeScopes := []models.TokenScope{
		{Namespace: "myapp/*", Read: true, Write: true},
	}

	writeToken, err := authSvc.GenerateToken(writeScopes, "", nil, nil)
	if err != nil {
		t.Fatalf("Failed to create write wildcard token: %v", err)
	}

	validatedWrite, err := authSvc.ValidateToken(writeToken.Token)
	if err != nil {
		t.Fatalf("Failed to validate write token: %v", err)
	}

	// Should have write access
	if !authSvc.HasAccess(validatedWrite, "myapp/prod", true) {
		t.Error("Wildcard token with write=true should have write access")
	}

	// Test global wildcard
	globalScopes := []models.TokenScope{
		{Namespace: "*", Read: true, Write: false},
	}

	globalToken, err := authSvc.GenerateToken(globalScopes, "", nil, nil)
	if err != nil {
		t.Fatalf("Failed to create global wildcard token: %v", err)
	}

	validatedGlobal, err := authSvc.ValidateToken(globalToken.Token)
	if err != nil {
		t.Fatalf("Failed to validate global token: %v", err)
	}

	// Global wildcard should match everything
	if !authSvc.HasAccess(validatedGlobal, "any/namespace", false) {
		t.Error("Global wildcard (*) should match any namespace")
	}

	if !authSvc.HasAccess(validatedGlobal, "completely/different/namespace", false) {
		t.Error("Global wildcard (*) should match any namespace")
	}
}
