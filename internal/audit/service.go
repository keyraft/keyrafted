package audit

import (
	"crypto/rand"
	"encoding/hex"
	"keyrafted/internal/models"
	"keyrafted/internal/storage"
	"time"
)

// Service provides audit logging functionality
type Service struct {
	store storage.Storage
}

// NewService creates a new audit service
func NewService(store storage.Storage) *Service {
	return &Service{
		store: store,
	}
}

// LogOperation logs an operation to the audit log
func (s *Service) LogOperation(tokenID, action, namespace, key string, success bool, errorMsg string) error {
	entry := &models.AuditLogEntry{
		ID:        generateID(),
		Timestamp: time.Now(),
		TokenID:   tokenID,
		Action:    action,
		Namespace: namespace,
		Key:       key,
		Success:   success,
		Error:     errorMsg,
	}

	return s.store.LogAudit(entry)
}

// GetLogs retrieves audit logs for a namespace
func (s *Service) GetLogs(namespace string, limit int) ([]*models.AuditLogEntry, error) {
	return s.store.GetAuditLogs(namespace, limit)
}

// generateID generates a unique ID for audit log entries
func generateID() string {
	// Simple ID generation using timestamp and random string
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of given length
func randomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based if crypto/rand fails
		now := time.Now().UnixNano()
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		b := make([]byte, length)
		for i := range b {
			b[i] = charset[now%int64(len(charset))]
			now = now / int64(len(charset))
		}
		return string(b)
	}
	return hex.EncodeToString(bytes)[:length]
}
