package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"keyrafted/internal/auth"
	"keyrafted/internal/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"version": "0.1.0",
		"time":    time.Now(),
	})
}

// handleMetrics returns Prometheus-compatible metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	// Basic metrics for now
	metrics := fmt.Sprintf(`# HELP keyraft_active_watches Number of active watch connections
# TYPE keyraft_active_watches gauge
keyraft_active_watches %d
`, s.watch.ActiveWatchers())

	if _, err := w.Write([]byte(metrics)); err != nil {
		log.Printf("Error writing metrics: %v", err)
	}
}

// handleSetKey stores or updates a key
func (s *Server) handleSetKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	key := vars["key"]

	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Check write permission
	if !s.auth.HasAccess(token, namespace, true) {
		respondError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	// Parse request body
	var req struct {
		Value    string            `json:"value"`
		Type     string            `json:"type"`
		Metadata map[string]string `json:"metadata,omitempty"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Default to config type
	entryType := models.TypeConfig
	if req.Type == "secret" {
		entryType = models.TypeSecret
	}

	// Store the entry
	entry, err := s.engine.Set(namespace, key, req.Value, entryType, req.Metadata)
	if err != nil {
		// Log failed operation
		_ = s.audit.LogOperation(token.ID, "set", namespace, key, false, err.Error())
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Notify watchers
	s.watch.NotifySet(entry)

	// Log successful operation
	_ = s.audit.LogOperation(token.ID, "set", namespace, key, true, "")

	respondJSON(w, http.StatusOK, entry)
}

// handleGetKey retrieves a key or lists keys if no key specified
func (s *Server) handleGetKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	key := vars["key"]

	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Check read permission
	if !s.auth.HasAccess(token, namespace, false) {
		respondError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	// Check for version parameter
	versionStr := r.URL.Query().Get("version")
	if versionStr != "" {
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid version parameter")
			return
		}

		ver, err := s.engine.GetVersion(namespace, key, version)
		if err != nil {
			respondError(w, http.StatusNotFound, "version not found")
			return
		}

		respondJSON(w, http.StatusOK, ver)
		return
	}

	// Get latest version
	entry, err := s.engine.Get(namespace, key)
	if err != nil {
		// If key not found, try treating the whole path as namespace for listing
		fullNamespace := namespace
		if key != "" {
			fullNamespace = namespace + "/" + key
		}
		entries, listErr := s.engine.List(fullNamespace)
		if listErr == nil && len(entries) >= 0 {
			_ = s.audit.LogOperation(token.ID, "list", fullNamespace, "", true, "")
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"namespace": fullNamespace,
				"keys":      entries,
				"count":     len(entries),
			})
			return
		}
		_ = s.audit.LogOperation(token.ID, "get", namespace, key, false, "key not found")
		respondError(w, http.StatusNotFound, "key not found")
		return
	}

	// Log successful operation
	_ = s.audit.LogOperation(token.ID, "get", namespace, key, true, "")

	respondJSON(w, http.StatusOK, entry)
}

// handleDeleteKey deletes a key
func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	key := vars["key"]

	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Check write permission
	if !s.auth.HasAccess(token, namespace, true) {
		respondError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	if err := s.engine.Delete(namespace, key); err != nil {
		_ = s.audit.LogOperation(token.ID, "delete", namespace, key, false, err.Error())
		respondError(w, http.StatusNotFound, "key not found")
		return
	}

	// Notify watchers
	s.watch.NotifyDelete(namespace, key)

	// Log successful operation
	_ = s.audit.LogOperation(token.ID, "delete", namespace, key, true, "")

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleListKeys lists all keys in a namespace
func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Check read permission
	if !s.auth.HasAccess(token, namespace, false) {
		respondError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	entries, err := s.engine.List(namespace)
	if err != nil {
		_ = s.audit.LogOperation(token.ID, "list", namespace, "", false, err.Error())
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Log successful operation
	_ = s.audit.LogOperation(token.ID, "list", namespace, "", true, "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"namespace": namespace,
		"keys":      entries,
		"count":     len(entries),
	})
}

// handleWatch implements long-polling and SSE for watching changes
func (s *Server) handleWatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Check read permission
	if !s.auth.HasAccess(token, namespace, false) {
		respondError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	// Log watch operation
	_ = s.audit.LogOperation(token.ID, "watch", namespace, "", true, "")

	// Check if SSE mode is requested
	if r.URL.Query().Get("stream") == "true" || r.Header.Get("Accept") == "text/event-stream" {
		s.handleWatchSSE(w, r, namespace)
		return
	}

	// Default: long-polling mode (backward compatible)
	s.handleWatchLongPoll(w, r, namespace)
}

// handleWatchLongPoll implements long-polling for watching changes
func (s *Server) handleWatchLongPoll(w http.ResponseWriter, r *http.Request, namespace string) {
	// Parse timeout parameter (default 30s)
	timeoutStr := r.URL.Query().Get("timeout")
	timeout := 30 * time.Second
	if timeoutStr != "" {
		if t, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = t
		}
	}

	// Create watcher
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	watcher := s.watch.Watch(ctx, namespace, 10)
	defer s.watch.Unwatch(watcher.ID)

	// Wait for event or timeout
	select {
	case event := <-watcher.Events:
		respondJSON(w, http.StatusOK, event)
	case <-ctx.Done():
		// Timeout - return empty response
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"timeout": true,
		})
	}
}

// handleWatchSSE implements Server-Sent Events for streaming watch updates
func (s *Server) handleWatchSSE(w http.ResponseWriter, r *http.Request, namespace string) {
	// Set SSE headers BEFORE writing any response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")          // Disable nginx buffering
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow CORS for SSE

	// Write status code before flushing
	w.WriteHeader(http.StatusOK)

	// Flush headers immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Create watcher with larger buffer for streaming
	ctx := r.Context()
	watcher := s.watch.Watch(ctx, namespace, 100)
	defer s.watch.Unwatch(watcher.ID)

	// Send initial connection event
	s.writeSSEEvent(w, "connected", map[string]interface{}{
		"namespace": namespace,
		"timestamp": time.Now(),
	})
	// Flush the connection event immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Stream events
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				// Channel closed
				return
			}
			// Send event as SSE
			s.writeSSEEvent(w, "change", event)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-ctx.Done():
			// Client disconnected or request cancelled
			return
		}
	}
}

// writeSSEEvent writes a Server-Sent Event
func (s *Server) writeSSEEvent(w http.ResponseWriter, eventType string, data interface{}) {
	// Format: event: <type>\ndata: <json>\n\n
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling SSE event: %v", err)
		return
	}

	// Ensure we're writing SSE format, not JSON
	event := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(jsonData))
	if _, err := w.Write([]byte(event)); err != nil {
		log.Printf("Error writing SSE event: %v", err)
	}
}

// handleListNamespaces lists all namespaces
func (s *Server) handleListNamespaces(w http.ResponseWriter, r *http.Request) {
	// Get token from context
	_, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	namespaces, err := s.engine.ListNamespaces()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"namespaces": namespaces,
		"count":      len(namespaces),
	})
}

// handleGetNamespace retrieves namespace metadata
func (s *Server) handleGetNamespace(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	// Get token from context
	_, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ns, err := s.engine.GetNamespace(namespace)
	if err != nil {
		respondError(w, http.StatusNotFound, "namespace not found")
		return
	}

	respondJSON(w, http.StatusOK, ns)
}

// handleCreateToken creates a new authentication token
func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Only root tokens can create new tokens
	if len(token.Scopes) > 0 {
		respondError(w, http.StatusForbidden, "only root tokens can create new tokens")
		return
	}

	// Parse request body
	var req struct {
		Scopes    []models.TokenScope `json:"scopes"`
		ExpiresIn *int64              `json:"expires_in,omitempty"` // seconds
		Metadata  map[string]string   `json:"metadata,omitempty"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	var expiresIn *time.Duration
	if req.ExpiresIn != nil {
		duration := time.Duration(*req.ExpiresIn) * time.Second
		expiresIn = &duration
	}

	newToken, err := s.auth.GenerateToken(req.Scopes, expiresIn, req.Metadata)
	if err != nil {
		_ = s.audit.LogOperation(token.ID, "token_create", "", "", false, err.Error())
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Log successful operation
	_ = s.audit.LogOperation(token.ID, "token_create", "", newToken.ID, true, "")

	respondJSON(w, http.StatusOK, newToken)
}

// handleListTokens lists all tokens
func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Only root tokens can list tokens
	if len(token.Scopes) > 0 {
		respondError(w, http.StatusForbidden, "only root tokens can list tokens")
		return
	}

	tokens, err := s.auth.ListTokens()
	if err != nil {
		_ = s.audit.LogOperation(token.ID, "token_list", "", "", false, err.Error())
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Log successful operation
	_ = s.audit.LogOperation(token.ID, "token_list", "", "", true, "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tokens": tokens,
		"count":  len(tokens),
	})
}

// handleRevokeToken revokes a token
func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tokenToRevoke := vars["token"]

	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Only root tokens can revoke tokens
	if len(token.Scopes) > 0 {
		respondError(w, http.StatusForbidden, "only root tokens can revoke tokens")
		return
	}

	if err := s.auth.RevokeToken(tokenToRevoke); err != nil {
		_ = s.audit.LogOperation(token.ID, "token_revoke", "", tokenToRevoke, false, err.Error())
		respondError(w, http.StatusNotFound, "token not found")
		return
	}

	// Log successful operation
	_ = s.audit.LogOperation(token.ID, "token_revoke", "", tokenToRevoke, true, "")

	respondJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// handleGetAuditLogs retrieves audit logs
func (s *Server) handleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Only root tokens can view audit logs
	if !s.auth.HasAccess(token, "*", true) {
		respondError(w, http.StatusForbidden, "insufficient permissions: audit logs require root access")
		return
	}

	// Parse query parameters
	namespace := r.URL.Query().Get("namespace")
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Get audit logs
	logs, err := s.audit.GetLogs(namespace, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}
