package storage

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"keyrafted/internal/models"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketNamespaces = []byte("namespaces")
	bucketKVData     = []byte("kv_data")
	bucketKVVersions = []byte("kv_versions")
	bucketTokens     = []byte("tokens")
	bucketAuditLog   = []byte("audit_log")
	bucketMeta       = []byte("meta")
)

// BoltDBStorage implements Storage interface using BoltDB
type BoltDBStorage struct {
	db   *bolt.DB
	path string
}

// NewBoltDBStorage creates a new BoltDB storage instance
func NewBoltDBStorage(path string) *BoltDBStorage {
	return &BoltDBStorage{
		path: path,
	}
}

// Open initializes the BoltDB database
func (s *BoltDBStorage) Open() error {
	db, err := bolt.Open(s.path, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Create buckets
	return s.db.Update(func(tx *bolt.Tx) error {
		buckets := [][]byte{
			bucketNamespaces,
			bucketKVData,
			bucketKVVersions,
			bucketTokens,
			bucketAuditLog,
			bucketMeta,
		}
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
}

// Close closes the database
func (s *BoltDBStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Set stores or updates a KV entry
func (s *BoltDBStorage) Set(entry *models.KVEntry) error {
	if err := models.ValidateNamespace(entry.Namespace); err != nil {
		return err
	}
	if err := models.ValidateKey(entry.Key); err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		// Ensure namespace exists
		ns := &models.Namespace{
			Name:      entry.Namespace,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.saveNamespace(tx, ns); err != nil {
			return err
		}

		// Get next version
		version, err := s.getNextVersionTx(tx, entry.Namespace, entry.Key)
		if err != nil {
			return err
		}
		entry.Version = version
		entry.UpdatedAt = time.Now()
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = entry.UpdatedAt
		}

		// Save current version to kv_data
		kvKey := makeKVKey(entry.Namespace, entry.Key)
		kvData, err := json.Marshal(entry)
		if err != nil {
			return err
		}

		bucket := tx.Bucket(bucketKVData)
		if err := bucket.Put([]byte(kvKey), kvData); err != nil {
			return err
		}

		// Save version history
		historyVersion := &models.Version{
			Namespace: entry.Namespace,
			Key:       entry.Key,
			Value:     entry.Value,
			Type:      entry.Type,
			Version:   version,
			Timestamp: entry.UpdatedAt,
			Metadata:  entry.Metadata,
		}
		versionKey := makeVersionKey(entry.Namespace, entry.Key, version)
		versionData, err := json.Marshal(historyVersion)
		if err != nil {
			return err
		}

		versionBucket := tx.Bucket(bucketKVVersions)
		return versionBucket.Put([]byte(versionKey), versionData)
	})
}

// Get retrieves the latest version of a KV entry
func (s *BoltDBStorage) Get(namespace, key string) (*models.KVEntry, error) {
	var entry *models.KVEntry

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKVData)
		kvKey := makeKVKey(namespace, key)
		data := bucket.Get([]byte(kvKey))
		if data == nil {
			return fmt.Errorf("key not found")
		}

		entry = &models.KVEntry{}
		if err := json.Unmarshal(data, entry); err != nil {
			return err
		}

		// Check if deleted
		if entry.IsDeleted {
			return fmt.Errorf("key not found")
		}

		return nil
	})

	return entry, err
}

// GetVersion retrieves a specific version of a KV entry
func (s *BoltDBStorage) GetVersion(namespace, key string, version int64) (*models.Version, error) {
	var ver *models.Version

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKVVersions)
		versionKey := makeVersionKey(namespace, key, version)
		data := bucket.Get([]byte(versionKey))
		if data == nil {
			return fmt.Errorf("version not found")
		}

		ver = &models.Version{}
		return json.Unmarshal(data, ver)
	})

	return ver, err
}

// List retrieves all non-deleted keys in a namespace
func (s *BoltDBStorage) List(namespace string) ([]*models.KVEntry, error) {
	var entries []*models.KVEntry

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKVData)
		prefix := []byte(namespace + "/")

		cursor := bucket.Cursor()
		for k, v := cursor.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = cursor.Next() {
			entry := &models.KVEntry{}
			if err := json.Unmarshal(v, entry); err != nil {
				continue
			}
			if !entry.IsDeleted {
				entries = append(entries, entry)
			}
		}

		return nil
	})

	return entries, err
}

// Delete marks a KV entry as deleted (soft delete)
func (s *BoltDBStorage) Delete(namespace, key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKVData)
		kvKey := makeKVKey(namespace, key)
		data := bucket.Get([]byte(kvKey))
		if data == nil {
			return fmt.Errorf("key not found")
		}

		entry := &models.KVEntry{}
		if err := json.Unmarshal(data, entry); err != nil {
			return err
		}

		entry.IsDeleted = true
		entry.UpdatedAt = time.Now()

		updatedData, err := json.Marshal(entry)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(kvKey), updatedData)
	})
}

// GetVersions retrieves all versions of a key
func (s *BoltDBStorage) GetVersions(namespace, key string) ([]*models.Version, error) {
	var versions []*models.Version

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketKVVersions)
		prefix := []byte(makeKVKey(namespace, key) + "#")

		cursor := bucket.Cursor()
		for k, v := cursor.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = cursor.Next() {
			version := &models.Version{}
			if err := json.Unmarshal(v, version); err != nil {
				continue
			}
			versions = append(versions, version)
		}

		return nil
	})

	return versions, err
}

// CreateNamespace creates or updates a namespace
func (s *BoltDBStorage) CreateNamespace(ns *models.Namespace) error {
	if err := models.ValidateNamespace(ns.Name); err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		return s.saveNamespace(tx, ns)
	})
}

// GetNamespace retrieves a namespace
func (s *BoltDBStorage) GetNamespace(name string) (*models.Namespace, error) {
	var ns *models.Namespace

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketNamespaces)
		data := bucket.Get([]byte(name))
		if data == nil {
			return fmt.Errorf("namespace not found")
		}

		ns = &models.Namespace{}
		return json.Unmarshal(data, ns)
	})

	return ns, err
}

// ListNamespaces retrieves all namespaces
func (s *BoltDBStorage) ListNamespaces() ([]*models.Namespace, error) {
	var namespaces []*models.Namespace

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketNamespaces)
		return bucket.ForEach(func(k, v []byte) error {
			ns := &models.Namespace{}
			if err := json.Unmarshal(v, ns); err != nil {
				return err
			}
			namespaces = append(namespaces, ns)
			return nil
		})
	})

	return namespaces, err
}

// SaveToken stores a token
func (s *BoltDBStorage) SaveToken(token *models.Token) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketTokens)
		data, err := json.Marshal(token)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(token.Token), data)
	})
}

// GetToken retrieves a token
func (s *BoltDBStorage) GetToken(tokenStr string) (*models.Token, error) {
	var token *models.Token

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketTokens)
		data := bucket.Get([]byte(tokenStr))
		if data == nil {
			return fmt.Errorf("token not found")
		}

		token = &models.Token{}
		return json.Unmarshal(data, token)
	})

	return token, err
}

// DeleteToken removes a token
func (s *BoltDBStorage) DeleteToken(tokenStr string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketTokens)
		return bucket.Delete([]byte(tokenStr))
	})
}

// ListTokens retrieves all tokens
func (s *BoltDBStorage) ListTokens() ([]*models.Token, error) {
	var tokens []*models.Token

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketTokens)
		return bucket.ForEach(func(k, v []byte) error {
			token := &models.Token{}
			if err := json.Unmarshal(v, token); err != nil {
				return err
			}
			tokens = append(tokens, token)
			return nil
		})
	})

	return tokens, err
}

// LogAudit stores an audit log entry
func (s *BoltDBStorage) LogAudit(entry *models.AuditLogEntry) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketAuditLog)
		
		// Use timestamp + ID as key for ordering
		key := fmt.Sprintf("%d_%s", entry.Timestamp.UnixNano(), entry.ID)
		
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(key), data)
	})
}

// GetAuditLogs retrieves audit logs for a namespace
func (s *BoltDBStorage) GetAuditLogs(namespace string, limit int) ([]*models.AuditLogEntry, error) {
	var logs []*models.AuditLogEntry

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketAuditLog)
		cursor := bucket.Cursor()

		count := 0
		// Iterate in reverse order (newest first)
		for k, v := cursor.Last(); k != nil && (limit <= 0 || count < limit); k, v = cursor.Prev() {
			entry := &models.AuditLogEntry{}
			if err := json.Unmarshal(v, entry); err != nil {
				continue
			}
			if namespace == "" || entry.Namespace == namespace {
				logs = append(logs, entry)
				count++
			}
		}

		return nil
	})

	return logs, err
}

// GetNextVersion gets the next version number for a key
func (s *BoltDBStorage) GetNextVersion(namespace, key string) (int64, error) {
	var version int64
	err := s.db.Update(func(tx *bolt.Tx) error {
		var err error
		version, err = s.getNextVersionTx(tx, namespace, key)
		return err
	})
	return version, err
}

// Helper functions

func (s *BoltDBStorage) getNextVersionTx(tx *bolt.Tx, namespace, key string) (int64, error) {
	bucket := tx.Bucket(bucketMeta)
	versionKey := []byte("version_" + makeKVKey(namespace, key))
	
	data := bucket.Get(versionKey)
	var version int64 = 1
	if data != nil {
		version = int64(binary.BigEndian.Uint64(data)) + 1
	}

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(version))
	if err := bucket.Put(versionKey, buf); err != nil {
		return 0, err
	}

	return version, nil
}

func (s *BoltDBStorage) saveNamespace(tx *bolt.Tx, ns *models.Namespace) error {
	bucket := tx.Bucket(bucketNamespaces)
	
	// Check if exists
	existing := bucket.Get([]byte(ns.Name))
	if existing != nil {
		existingNs := &models.Namespace{}
		if err := json.Unmarshal(existing, existingNs); err == nil {
			ns.CreatedAt = existingNs.CreatedAt
		}
	} else {
		ns.CreatedAt = time.Now()
	}
	ns.UpdatedAt = time.Now()

	data, err := json.Marshal(ns)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(ns.Name), data)
}

func makeKVKey(namespace, key string) string {
	return namespace + "/" + key
}

func makeVersionKey(namespace, key string, version int64) string {
	return fmt.Sprintf("%s#%020d", makeKVKey(namespace, key), version)
}

