package unit

import (
	"keyrafted/internal/models"
	"keyrafted/internal/storage"
	"path/filepath"
	"testing"
	"time"
)

func TestAuditLogPagination(t *testing.T) {
	store := storage.NewBoltDBStorage(filepath.Join(t.TempDir(), "audit.db"))
	if err := store.Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	for i := 0; i < 5; i++ {
		if err := store.LogAudit(&models.AuditLogEntry{
			ID:        "id",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			TokenID:   "tok",
			Action:    "set",
			Namespace: "test",
			Key:       "K",
			Success:   true,
		}); err != nil {
			t.Fatalf("log %d: %v", i, err)
		}
	}

	total, err := store.CountAuditLogs("test")
	if err != nil || total != 5 {
		t.Fatalf("count = %d, err = %v", total, err)
	}

	page1, err := store.GetAuditLogs("test", 2, 0)
	if err != nil || len(page1) != 2 {
		t.Fatalf("page1 len = %d, err = %v", len(page1), err)
	}
	page2, err := store.GetAuditLogs("test", 2, 2)
	if err != nil || len(page2) != 2 {
		t.Fatalf("page2 len = %d, err = %v", len(page2), err)
	}
	if page1[0].Timestamp.Before(page1[1].Timestamp) {
		t.Fatal("expected newest first on page 1")
	}
}
