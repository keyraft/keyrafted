package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"keyrafted/internal/auth"
	"keyrafted/internal/models"
	"keyrafted/internal/version"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"version": version.Version,
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

// handleListVersions lists all versions of a key
func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	key := vars["key"]

	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if !s.auth.HasAccess(token, namespace, false) {
		respondError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	versions, err := s.engine.GetVersions(namespace, key)
	if err != nil {
		_ = s.audit.LogOperation(token.ID, "list_versions", namespace, key, false, err.Error())
		respondError(w, http.StatusNotFound, "key not found")
		return
	}

	_ = s.audit.LogOperation(token.ID, "list_versions", namespace, key, true, "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"namespace": namespace,
		"key":       key,
		"versions":  versions,
		"count":     len(versions),
	})
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

// looksLikeKeyName guesses whether a path segment is a key vs namespace part (ponytail: underscore/uppercase heuristic)
func looksLikeKeyName(s string) bool {
	if strings.Contains(s, "_") {
		return true
	}
	return s != strings.ToLower(s)
}

// handleListKeys lists keys in a namespace, or returns a single key when the path is unambiguous
func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	path := r.URL.Path
	kvPrefix := "/v1/kv/"
	if !strings.HasPrefix(path, kvPrefix) {
		respondError(w, http.StatusBadRequest, "invalid path")
		return
	}

	relativePath := strings.TrimPrefix(path, kvPrefix)
	pathSegments := strings.Split(relativePath, "/")

	// 2+ segments: try get (namespace = prefix, key = last segment); fall back to list full path
	if len(pathSegments) >= 2 {
		potentialNamespace := strings.Join(pathSegments[:len(pathSegments)-1], "/")
		potentialKey := pathSegments[len(pathSegments)-1]

		if s.auth.HasAccess(token, potentialNamespace, false) {
			// Specific version requested (e.g. ?version=2)
			if versionStr := r.URL.Query().Get("version"); versionStr != "" {
				version, err := strconv.ParseInt(versionStr, 10, 64)
				if err != nil {
					respondError(w, http.StatusBadRequest, "invalid version parameter")
					return
				}
				ver, err := s.engine.GetVersion(potentialNamespace, potentialKey, version)
				if err != nil {
					_ = s.audit.LogOperation(token.ID, "get", potentialNamespace, potentialKey, false, err.Error())
					respondError(w, http.StatusNotFound, "version not found")
					return
				}
				_ = s.audit.LogOperation(token.ID, "get", potentialNamespace, potentialKey, true, "")
				respondJSON(w, http.StatusOK, ver)
				return
			}

			entry, err := s.engine.Get(potentialNamespace, potentialKey)
			if err == nil && entry != nil {
				_ = s.audit.LogOperation(token.ID, "get", potentialNamespace, potentialKey, true, "")
				respondJSON(w, http.StatusOK, entry)
				return
			}

			// Full path is itself a namespace → list, don't 404 as missing key
			if _, nsErr := s.engine.GetNamespace(relativePath); nsErr == nil {
				// fall through to list below
			} else if looksLikeKeyName(potentialKey) {
				_ = s.audit.LogOperation(token.ID, "get", potentialNamespace, potentialKey, false, "key not found")
				respondError(w, http.StatusNotFound, "key not found")
				return
			}
		}
	}

	namespace := relativePath
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

// handleListNamespaces lists namespaces the caller can read
func (s *Server) handleListNamespaces(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	namespaces, err := s.engine.ListNamespaces()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	allowed := make([]*models.Namespace, 0, len(namespaces))
	for _, ns := range namespaces {
		if s.auth.HasAccess(token, ns.Name, false) {
			allowed = append(allowed, ns)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"namespaces": allowed,
		"count":      len(allowed),
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

// handleDeleteNamespace deletes a namespace and all secrets in it
func (s *Server) handleDeleteNamespace(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if !s.auth.HasAccess(token, namespace, true) {
		respondError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	keys, err := s.engine.DeleteNamespace(namespace)
	if err != nil {
		if err.Error() == "namespace not found" {
			respondError(w, http.StatusNotFound, "namespace not found")
			return
		}
		_ = s.audit.LogOperation(token.ID, "delete_namespace", namespace, "", false, err.Error())
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, key := range keys {
		s.watch.NotifyDelete(namespace, key)
	}
	_ = s.audit.LogOperation(token.ID, "delete_namespace", namespace, "", true, "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "deleted",
		"namespace": namespace,
		"keys":      keys,
	})
}

// handleAuthMe returns the current token's identity and capability flags for the UI
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":                token.ID,
		"role":              token.Role,
		"scopes":            token.Scopes,
		"metadata":          token.Metadata,
		"expires_at":        token.ExpiresAt,
		"is_root":           token.Role == models.RoleAdmin || (len(token.Scopes) == 0 && token.Role == ""),
		"can_manage_tokens": s.auth.HasPermission(token, models.PermissionManageTokens),
		"can_view_audit":    s.auth.HasPermission(token, models.PermissionViewAudit),
		"can_manage_roles":  s.auth.HasPermission(token, models.PermissionManageRoles),
	})
}

// handleCreateToken creates a new authentication token
func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Only admin/root tokens can create new tokens
	if !s.auth.HasPermission(token, models.PermissionManageTokens) {
		respondError(w, http.StatusForbidden, "insufficient permissions: manage_tokens required")
		return
	}

	// Parse request body
	var req struct {
		Scopes    []models.TokenScope `json:"scopes,omitempty"`     // Legacy: for backward compatibility
		Role      string              `json:"role,omitempty"`       // RBAC: role name
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

	// Validate: must provide either role or scopes (not both, not neither)
	if req.Role != "" && len(req.Scopes) > 0 {
		respondError(w, http.StatusBadRequest, "cannot specify both role and scopes")
		return
	}
	if req.Role == "" && len(req.Scopes) == 0 {
		respondError(w, http.StatusBadRequest, "must specify either role or scopes")
		return
	}

	var expiresIn *time.Duration
	if req.ExpiresIn != nil {
		duration := time.Duration(*req.ExpiresIn) * time.Second
		expiresIn = &duration
	}

	newToken, err := s.auth.GenerateToken(req.Scopes, req.Role, expiresIn, req.Metadata)
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

	// Check permission
	if !s.auth.HasPermission(token, models.PermissionManageTokens) {
		respondError(w, http.StatusForbidden, "insufficient permissions: manage_tokens required")
		return
	}

	// Check permission
	if !s.auth.HasPermission(token, models.PermissionManageTokens) {
		respondError(w, http.StatusForbidden, "insufficient permissions: manage_tokens required")
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

	// Check permission
	if !s.auth.HasPermission(token, models.PermissionManageTokens) {
		respondError(w, http.StatusForbidden, "insufficient permissions: manage_tokens required")
		return
	}

	if err := s.auth.RevokeToken(tokenToRevoke); err != nil {
		_ = s.audit.LogOperation(token.ID, "token_revoke", "", tokenToRevoke, false, err.Error())
		msg := err.Error()
		status := http.StatusNotFound
		if strings.Contains(msg, "last root token") {
			status = http.StatusForbidden
		} else if msg != "token not found" {
			status = http.StatusInternalServerError
		}
		respondError(w, status, msg)
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

	// Check permission
	if !s.auth.HasPermission(token, models.PermissionViewAudit) {
		respondError(w, http.StatusForbidden, "insufficient permissions: view_audit required")
		return
	}

	// Parse query parameters
	namespace := r.URL.Query().Get("namespace")
	limit := 25
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > 100 {
		limit = 100
	}
	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	total, err := s.audit.CountLogs(namespace)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	logs, err := s.audit.GetLogs(namespace, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"logs":     logs,
		"count":    len(logs),
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"has_more": offset+len(logs) < total,
	})
}

// handleListRoles lists all available roles
func (s *Server) handleListRoles(w http.ResponseWriter, r *http.Request) {
	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Check permission
	if !s.auth.HasPermission(token, models.PermissionManageRoles) {
		respondError(w, http.StatusForbidden, "insufficient permissions: manage_roles required")
		return
	}

	roles := models.GetDefaultRoles()
	roleList := make([]*models.Role, 0, len(roles))
	for _, role := range roles {
		roleList = append(roleList, role)
	}
	// Map iteration order is random — keep a stable display order
	order := map[string]int{
		models.RoleAdmin:     0,
		models.RoleDeveloper: 1,
		models.RoleOperator:  2,
		models.RoleViewer:    3,
	}
	sort.Slice(roleList, func(i, j int) bool {
		ai, aOk := order[roleList[i].Name]
		aj, bOk := order[roleList[j].Name]
		if aOk && bOk {
			return ai < aj
		}
		return roleList[i].Name < roleList[j].Name
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"roles": roleList,
		"count": len(roleList),
	})
}

// handleGetRole retrieves details of a specific role
func (s *Server) handleGetRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roleName := vars["role"]

	// Get token from context
	token, err := auth.GetTokenFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Check permission
	if !s.auth.HasPermission(token, models.PermissionManageRoles) {
		respondError(w, http.StatusForbidden, "insufficient permissions: manage_roles required")
		return
	}

	roles := models.GetDefaultRoles()
	role, exists := roles[roleName]
	if !exists {
		respondError(w, http.StatusNotFound, "role not found")
		return
	}

	respondJSON(w, http.StatusOK, role)
}
