package engine

import (
	"fmt"
	"keyrafted/internal/crypto"
	"keyrafted/internal/models"
	"keyrafted/internal/storage"
	"time"
)

// Engine handles config and secrets operations
type Engine struct {
	storage   storage.Storage
	encryptor *crypto.Encryptor
}

// NewEngine creates a new engine
func NewEngine(storage storage.Storage, encryptor *crypto.Encryptor) *Engine {
	return &Engine{
		storage:   storage,
		encryptor: encryptor,
	}
}

// Set stores or updates a key-value entry
func (e *Engine) Set(namespace, key, value string, entryType models.EntryType, metadata map[string]string) (*models.KVEntry, error) {
	if err := models.ValidateNamespace(namespace); err != nil {
		return nil, err
	}
	if err := models.ValidateKey(key); err != nil {
		return nil, err
	}

	// Encrypt value if it's a secret
	storedValue := value
	if entryType == models.TypeSecret {
		encrypted, err := e.encryptor.Encrypt(value)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret: %w", err)
		}
		storedValue = encrypted
	}

	entry := &models.KVEntry{
		Namespace: namespace,
		Key:       key,
		Value:     storedValue,
		Type:      entryType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  metadata,
	}

	if err := e.storage.Set(entry); err != nil {
		return nil, err
	}

	// Return entry with decrypted value for display
	entry.Value = value

	return entry, nil
}

// Get retrieves the latest version of a key
func (e *Engine) Get(namespace, key string) (*models.KVEntry, error) {
	entry, err := e.storage.Get(namespace, key)
	if err != nil {
		return nil, err
	}

	// Decrypt if it's a secret
	if entry.Type == models.TypeSecret {
		decrypted, err := e.encryptor.Decrypt(entry.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt secret: %w", err)
		}
		entry.Value = decrypted
	}

	return entry, nil
}

// GetVersion retrieves a specific version of a key
func (e *Engine) GetVersion(namespace, key string, version int64) (*models.Version, error) {
	ver, err := e.storage.GetVersion(namespace, key, version)
	if err != nil {
		return nil, err
	}

	// Decrypt if it's a secret
	if ver.Type == models.TypeSecret {
		decrypted, err := e.encryptor.Decrypt(ver.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt secret: %w", err)
		}
		ver.Value = decrypted
	}

	return ver, nil
}

// List retrieves all keys in a namespace
func (e *Engine) List(namespace string) ([]*models.KVEntry, error) {
	entries, err := e.storage.List(namespace)
	if err != nil {
		return nil, err
	}

	// Decrypt secrets
	for _, entry := range entries {
		if entry.Type == models.TypeSecret {
			decrypted, err := e.encryptor.Decrypt(entry.Value)
			if err != nil {
				// Skip entries that can't be decrypted
				continue
			}
			entry.Value = decrypted
		}
	}

	return entries, nil
}

// Delete marks a key as deleted
func (e *Engine) Delete(namespace, key string) error {
	return e.storage.Delete(namespace, key)
}

// GetVersions retrieves all versions of a key
func (e *Engine) GetVersions(namespace, key string) ([]*models.Version, error) {
	versions, err := e.storage.GetVersions(namespace, key)
	if err != nil {
		return nil, err
	}

	// Decrypt secrets
	for _, ver := range versions {
		if ver.Type == models.TypeSecret {
			decrypted, err := e.encryptor.Decrypt(ver.Value)
			if err != nil {
				// Skip versions that can't be decrypted
				continue
			}
			ver.Value = decrypted
		}
	}

	return versions, nil
}

// ListNamespaces retrieves all namespaces
func (e *Engine) ListNamespaces() ([]*models.Namespace, error) {
	return e.storage.ListNamespaces()
}

// GetNamespace retrieves namespace metadata
func (e *Engine) GetNamespace(name string) (*models.Namespace, error) {
	return e.storage.GetNamespace(name)
}

