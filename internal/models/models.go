package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// EntryType represents whether an entry is a config or secret
type EntryType string

const (
	TypeConfig EntryType = "config"
	TypeSecret EntryType = "secret"
)

// KVEntry represents a key-value entry in the store
type KVEntry struct {
	Namespace string            `json:"namespace"`
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Type      EntryType         `json:"type"`
	Version   int64             `json:"version"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	IsDeleted bool              `json:"is_deleted,omitempty"`
}

// Version represents a historical version of a KV entry
type Version struct {
	Namespace string            `json:"namespace"`
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Type      EntryType         `json:"type"`
	Version   int64             `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Namespace represents namespace metadata
type Namespace struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Token represents an authentication token
type Token struct {
	ID        string            `json:"id"`
	Token     string            `json:"token"`
	Scopes    []TokenScope      `json:"scopes,omitempty"` // Legacy: for backward compatibility
	Role      string            `json:"role,omitempty"`     // RBAC: role name (admin, developer, viewer, operator)
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// TokenScope represents access control for a token
type TokenScope struct {
	Namespace string `json:"namespace"`
	Read      bool   `json:"read"`
	Write     bool   `json:"write"`
}

// AuditLogEntry represents an audit log entry
type AuditLogEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	TokenID   string    `json:"token_id"`
	Action    string    `json:"action"`
	Namespace string    `json:"namespace"`
	Key       string    `json:"key,omitempty"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// Role represents a role in the RBAC system
type Role struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	Description string   `json:"description,omitempty"`
}

// Permission constants
const (
	PermissionRead        = "read"
	PermissionWrite       = "write"
	PermissionDelete      = "delete"
	PermissionManageTokens = "manage_tokens"
	PermissionManageRoles  = "manage_roles"
	PermissionViewAudit    = "view_audit"
	PermissionManageNamespaces = "manage_namespaces"
)

// Role constants
const (
	RoleAdmin     = "admin"
	RoleDeveloper = "developer"
	RoleViewer    = "viewer"
	RoleOperator  = "operator"
)

// GetDefaultRoles returns the default roles with their permissions
func GetDefaultRoles() map[string]*Role {
	return map[string]*Role{
		RoleAdmin: {
			Name:        RoleAdmin,
			Description: "Full access to all resources",
			Permissions: []string{
				PermissionRead,
				PermissionWrite,
				PermissionDelete,
				PermissionManageTokens,
				PermissionManageRoles,
				PermissionViewAudit,
				PermissionManageNamespaces,
			},
		},
		RoleDeveloper: {
			Name:        RoleDeveloper,
			Description: "Can read and write config/secrets in assigned namespaces",
			Permissions: []string{
				PermissionRead,
				PermissionWrite,
				PermissionDelete,
			},
		},
		RoleViewer: {
			Name:        RoleViewer,
			Description: "Read-only access to assigned namespaces",
			Permissions: []string{
				PermissionRead,
			},
		},
		RoleOperator: {
			Name:        RoleOperator,
			Description: "Can read, write, and view audit logs in assigned namespaces",
			Permissions: []string{
				PermissionRead,
				PermissionWrite,
				PermissionDelete,
				PermissionViewAudit,
			},
		},
	}
}

// ValidateNamespace validates namespace format: project/environment/service
var namespaceRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+(/[a-zA-Z0-9_-]+){0,2}$`)

func ValidateNamespace(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if len(namespace) > 256 {
		return fmt.Errorf("namespace too long (max 256 characters)")
	}
	if !namespaceRegex.MatchString(namespace) {
		return fmt.Errorf("invalid namespace format: must be alphanumeric with forward slashes (e.g., project/environment/service)")
	}
	return nil
}

// ValidateKey validates key name
func ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > 256 {
		return fmt.Errorf("key too long (max 256 characters)")
	}
	// Allow alphanumeric, underscore, hyphen, dot
	keyRegex := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !keyRegex.MatchString(key) {
		return fmt.Errorf("invalid key format: must be alphanumeric with ._- characters")
	}
	return nil
}

// ToJSON converts a struct to JSON
func ToJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// FromJSON parses JSON into a struct
func FromJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
