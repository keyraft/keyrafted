package storage

import (
	"keyrafted/internal/models"
)

// Storage defines the interface for the storage backend
type Storage interface {
	// Initialize the storage
	Open() error
	Close() error

	// KV operations
	Set(entry *models.KVEntry) error
	Get(namespace, key string) (*models.KVEntry, error)
	GetVersion(namespace, key string, version int64) (*models.Version, error)
	List(namespace string) ([]*models.KVEntry, error)
	Delete(namespace, key string) error
	GetVersions(namespace, key string) ([]*models.Version, error)

	// Namespace operations
	CreateNamespace(ns *models.Namespace) error
	GetNamespace(name string) (*models.Namespace, error)
	ListNamespaces() ([]*models.Namespace, error)

	// Token operations
	SaveToken(token *models.Token) error
	GetToken(tokenStr string) (*models.Token, error)
	DeleteToken(tokenStr string) error
	ListTokens() ([]*models.Token, error)

	// Audit log operations
	LogAudit(entry *models.AuditLogEntry) error
	GetAuditLogs(namespace string, limit int) ([]*models.AuditLogEntry, error)

	// Version management
	GetNextVersion(namespace, key string) (int64, error)
}

