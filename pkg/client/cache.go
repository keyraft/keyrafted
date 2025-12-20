package client

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// CachedClient wraps the client with caching and auto-reload
type CachedClient struct {
	client       *Client
	namespace    string
	cache        map[string]*Entry
	cacheMu      sync.RWMutex
	callbacks    []func(key, value string)
	callbacksMu  sync.RWMutex
	watchCtx     context.Context
	watchCancel  context.CancelFunc
	pollInterval time.Duration
}

// CacheConfig holds cached client configuration
type CacheConfig struct {
	Client       *Client
	Namespace    string
	PollInterval time.Duration // How often to poll for changes
}

// NewCachedClient creates a new cached client with auto-reload
func NewCachedClient(config CacheConfig) (*CachedClient, error) {
	if config.PollInterval == 0 {
		config.PollInterval = 10 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	cc := &CachedClient{
		client:       config.Client,
		namespace:    config.Namespace,
		cache:        make(map[string]*Entry),
		watchCtx:     ctx,
		watchCancel:  cancel,
		pollInterval: config.PollInterval,
	}

	// Initial load
	if err := cc.reload(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load initial data: %w", err)
	}

	// Start watch goroutine
	go cc.watchLoop()

	return cc, nil
}

// Get retrieves a value from cache
func (cc *CachedClient) Get(key string) (string, bool) {
	cc.cacheMu.RLock()
	defer cc.cacheMu.RUnlock()

	entry, exists := cc.cache[key]
	if !exists {
		return "", false
	}

	return entry.Value, true
}

// GetEntry retrieves the full entry from cache
func (cc *CachedClient) GetEntry(key string) (*Entry, bool) {
	cc.cacheMu.RLock()
	defer cc.cacheMu.RUnlock()

	entry, exists := cc.cache[key]
	return entry, exists
}

// GetAll returns all cached entries
func (cc *CachedClient) GetAll() map[string]string {
	cc.cacheMu.RLock()
	defer cc.cacheMu.RUnlock()

	result := make(map[string]string, len(cc.cache))
	for key, entry := range cc.cache {
		result[key] = entry.Value
	}

	return result
}

// Set updates a value and refreshes cache
func (cc *CachedClient) Set(key, value string, metadata map[string]string) error {
	_, err := cc.client.Set(cc.namespace, key, value, metadata)
	if err != nil {
		return err
	}

	// Update cache immediately
	return cc.reload()
}

// SetSecret updates a secret and refreshes cache
func (cc *CachedClient) SetSecret(key, value string, metadata map[string]string) error {
	_, err := cc.client.SetSecret(cc.namespace, key, value, metadata)
	if err != nil {
		return err
	}

	// Update cache immediately
	return cc.reload()
}

// Delete removes a key and refreshes cache
func (cc *CachedClient) Delete(key string) error {
	if err := cc.client.Delete(cc.namespace, key); err != nil {
		return err
	}

	// Update cache immediately
	return cc.reload()
}

// OnChange registers a callback to be called when a key changes
func (cc *CachedClient) OnChange(callback func(key, value string)) {
	cc.callbacksMu.Lock()
	defer cc.callbacksMu.Unlock()

	cc.callbacks = append(cc.callbacks, callback)
}

// Close stops the watch loop
func (cc *CachedClient) Close() {
	cc.watchCancel()
}

// reload fetches all keys and updates the cache
func (cc *CachedClient) reload() error {
	entries, err := cc.client.List(cc.namespace)
	if err != nil {
		return err
	}

	cc.cacheMu.Lock()
	oldCache := cc.cache
	newCache := make(map[string]*Entry)

	for _, entry := range entries {
		newCache[entry.Key] = entry
	}

	cc.cache = newCache
	cc.cacheMu.Unlock()

	// Notify callbacks of changes
	cc.notifyChanges(oldCache, newCache)

	return nil
}

// watchLoop continuously watches for changes
func (cc *CachedClient) watchLoop() {
	ticker := time.NewTicker(cc.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cc.watchCtx.Done():
			return
		case <-ticker.C:
			// Poll for changes using watch API
			event, err := cc.client.Watch(cc.namespace, 5*time.Second)
			if err != nil {
				log.Printf("Watch error: %v", err)
				continue
			}

			// If there was a change, reload the cache
			if !event.Timeout {
				if err := cc.reload(); err != nil {
					log.Printf("Reload error: %v", err)
				}
			}
		}
	}
}

// notifyChanges calls registered callbacks for changed keys
func (cc *CachedClient) notifyChanges(oldCache, newCache map[string]*Entry) {
	cc.callbacksMu.RLock()
	callbacks := cc.callbacks
	cc.callbacksMu.RUnlock()

	if len(callbacks) == 0 {
		return
	}

	// Check for new or updated keys
	for key, newEntry := range newCache {
		oldEntry, existed := oldCache[key]
		if !existed || oldEntry.Version != newEntry.Version {
			// Key was added or updated
			for _, callback := range callbacks {
				go callback(key, newEntry.Value)
			}
		}
	}

	// Check for deleted keys
	for key := range oldCache {
		if _, exists := newCache[key]; !exists {
			// Key was deleted
			for _, callback := range callbacks {
				go callback(key, "")
			}
		}
	}
}
