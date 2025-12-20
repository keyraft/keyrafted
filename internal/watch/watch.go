package watch

import (
	"context"
	"keyrafted/internal/models"
	"sync"
	"time"
)

// Event represents a change event
type Event struct {
	Action    string          `json:"action"` // "set", "delete"
	Namespace string          `json:"namespace"`
	Key       string          `json:"key"`
	Entry     *models.KVEntry `json:"entry,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// Watcher represents a client watching for changes
type Watcher struct {
	ID        string
	Namespace string
	Events    chan Event
	ctx       context.Context
	cancel    context.CancelFunc
}

// Manager manages watchers and distributes events
type Manager struct {
	watchers map[string]*Watcher
	mu       sync.RWMutex
}

// NewManager creates a new watch manager
func NewManager() *Manager {
	return &Manager{
		watchers: make(map[string]*Watcher),
	}
}

// Watch creates a new watcher for a namespace
func (m *Manager) Watch(ctx context.Context, namespace string, bufferSize int) *Watcher {
	m.mu.Lock()
	defer m.mu.Unlock()

	watcherCtx, cancel := context.WithCancel(ctx)

	watcher := &Watcher{
		ID:        generateWatcherID(),
		Namespace: namespace,
		Events:    make(chan Event, bufferSize),
		ctx:       watcherCtx,
		cancel:    cancel,
	}

	m.watchers[watcher.ID] = watcher

	// Start cleanup goroutine
	go m.cleanupWatcher(watcher)

	return watcher
}

// Unwatch removes a watcher
func (m *Manager) Unwatch(watcherID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if watcher, exists := m.watchers[watcherID]; exists {
		watcher.cancel()
		close(watcher.Events)
		delete(m.watchers, watcherID)
	}
}

// Notify sends an event to all matching watchers
func (m *Manager) Notify(event Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	event.Timestamp = time.Now()

	for _, watcher := range m.watchers {
		// Check if watcher is interested in this namespace
		if matchesNamespace(watcher.Namespace, event.Namespace) {
			select {
			case watcher.Events <- event:
				// Event sent
			default:
				// Channel full, skip (prevents blocking)
			}
		}
	}
}

// NotifySet notifies watchers of a set operation
func (m *Manager) NotifySet(entry *models.KVEntry) {
	m.Notify(Event{
		Action:    "set",
		Namespace: entry.Namespace,
		Key:       entry.Key,
		Entry:     entry,
		Timestamp: time.Now(),
	})
}

// NotifyDelete notifies watchers of a delete operation
func (m *Manager) NotifyDelete(namespace, key string) {
	m.Notify(Event{
		Action:    "delete",
		Namespace: namespace,
		Key:       key,
		Timestamp: time.Now(),
	})
}

// GetWatcher retrieves a watcher by ID
func (m *Manager) GetWatcher(watcherID string) (*Watcher, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	watcher, exists := m.watchers[watcherID]
	return watcher, exists
}

// ActiveWatchers returns the number of active watchers
func (m *Manager) ActiveWatchers() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.watchers)
}

// cleanupWatcher removes watcher when context is cancelled
func (m *Manager) cleanupWatcher(watcher *Watcher) {
	<-watcher.ctx.Done()

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.watchers[watcher.ID]; exists {
		close(watcher.Events)
		delete(m.watchers, watcher.ID)
	}
}

// matchesNamespace checks if a watcher namespace matches an event namespace
func matchesNamespace(watcherNS, eventNS string) bool {
	// Exact match
	if watcherNS == eventNS {
		return true
	}

	// Wildcard match
	if watcherNS == "*" {
		return true
	}

	// Prefix match (e.g., "billing" matches "billing/prod")
	if len(watcherNS) < len(eventNS) && eventNS[:len(watcherNS)] == watcherNS && eventNS[len(watcherNS)] == '/' {
		return true
	}

	return false
}

// generateWatcherID generates a unique ID for a watcher
var watcherCounter uint64
var watcherMu sync.Mutex

func generateWatcherID() string {
	watcherMu.Lock()
	defer watcherMu.Unlock()
	watcherCounter++
	return time.Now().Format("20060102150405") + "-" + string(rune(watcherCounter))
}
