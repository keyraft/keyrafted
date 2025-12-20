package api

import (
	"context"
	"encoding/json"
	"keyrafted/internal/audit"
	"keyrafted/internal/auth"
	"keyrafted/internal/engine"
	"keyrafted/internal/watch"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Server represents the HTTP API server
type Server struct {
	engine     *engine.Engine
	auth       *auth.Service
	watch      *watch.Manager
	audit      *audit.Service
	router     *mux.Router
	server     *http.Server
	listenAddr string
}

// NewServer creates a new API server
func NewServer(listenAddr string, engine *engine.Engine, authSvc *auth.Service, watchMgr *watch.Manager, auditSvc *audit.Service) *Server {
	s := &Server{
		engine:     engine,
		auth:       authSvc,
		watch:      watchMgr,
		audit:      auditSvc,
		listenAddr: listenAddr,
	}

	s.setupRouter()
	return s
}

// setupRouter configures the HTTP routes
func (s *Server) setupRouter() {
	r := mux.NewRouter()

	// Public endpoints (no auth required)
	r.HandleFunc("/v1/health", s.handleHealth).Methods("GET")
	r.HandleFunc("/v1/metrics", s.handleMetrics).Methods("GET")

	// Protected endpoints (auth required)
	api := r.PathPrefix("/v1").Subrouter()
	api.Use(s.loggingMiddleware)
	api.Use(s.auth.Middleware)

	// KV endpoints
	api.HandleFunc("/kv/{namespace:.+}/{key:[^/]+}", s.handleSetKey).Methods("PUT")
	api.HandleFunc("/kv/{namespace:.+}/{key:[^/]+}", s.handleDeleteKey).Methods("DELETE")
	api.HandleFunc("/kv/{namespace:.+}/{key:[^/]+}", s.handleGetKey).Methods("GET")
	api.HandleFunc("/kv/{namespace:.+}", s.handleListKeys).Methods("GET")

	// Watch endpoint
	api.HandleFunc("/watch/{namespace:.+}", s.handleWatch).Methods("GET")

	// Namespace endpoints
	api.HandleFunc("/namespaces", s.handleListNamespaces).Methods("GET")
	api.HandleFunc("/namespaces/{namespace:.+}", s.handleGetNamespace).Methods("GET")

	// Auth endpoints
	api.HandleFunc("/auth/token", s.handleCreateToken).Methods("POST")
	api.HandleFunc("/auth/tokens", s.handleListTokens).Methods("GET")
	api.HandleFunc("/auth/token/{token}", s.handleRevokeToken).Methods("DELETE")

	// Audit log endpoints
	api.HandleFunc("/audit", s.handleGetAuditLogs).Methods("GET")

	s.router = r
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         s.listenAddr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting Keyraft server on %s", s.listenAddr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.server.Shutdown(ctx)
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher to support SSE
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Helper functions
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Error encoding JSON: %v", err)
		}
	}
}

func respondError(w http.ResponseWriter, statusCode int, message string) {
	respondJSON(w, statusCode, map[string]string{"error": message})
}
