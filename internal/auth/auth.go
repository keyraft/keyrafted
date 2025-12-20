package auth

import (
	"context"
	"fmt"
	"keyrafted/internal/crypto"
	"keyrafted/internal/models"
	"keyrafted/internal/storage"
	"net/http"
	"strings"
	"time"
)

// contextKey is a custom type for context keys
type contextKey string

const (
	tokenContextKey contextKey = "token"
)

// Service handles authentication operations
type Service struct {
	storage storage.Storage
}

// NewService creates a new authentication service
func NewService(storage storage.Storage) *Service {
	return &Service{
		storage: storage,
	}
}

// GenerateToken creates a new authentication token
func (s *Service) GenerateToken(scopes []models.TokenScope, expiresIn *time.Duration, metadata map[string]string) (*models.Token, error) {
	tokenStr, err := crypto.GenerateToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	id, err := crypto.GenerateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	token := &models.Token{
		ID:        id,
		Token:     tokenStr,
		Scopes:    scopes,
		CreatedAt: time.Now(),
		Metadata:  metadata,
	}

	if expiresIn != nil {
		expiresAt := time.Now().Add(*expiresIn)
		token.ExpiresAt = &expiresAt
	}

	if err := s.storage.SaveToken(token); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	return token, nil
}

// ValidateToken validates a token string and returns the token object
func (s *Service) ValidateToken(tokenStr string) (*models.Token, error) {
	token, err := s.storage.GetToken(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	// Check expiration
	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return token, nil
}

// HasAccess checks if a token has the required access to a namespace
func (s *Service) HasAccess(token *models.Token, namespace string, write bool) bool {
	// Root token (empty scopes) has access to everything
	if len(token.Scopes) == 0 {
		return true
	}

	// Check each scope
	for _, scope := range token.Scopes {
		// Wildcard match
		if scope.Namespace == "*" {
			if write {
				return scope.Write
			}
			return scope.Read
		}

		// Exact match
		if scope.Namespace == namespace {
			if write {
				return scope.Write
			}
			return scope.Read
		}

		// Prefix match (e.g., "billing/*" matches "billing/prod/api")
		if strings.HasSuffix(scope.Namespace, "/*") {
			prefix := strings.TrimSuffix(scope.Namespace, "/*")
			if strings.HasPrefix(namespace, prefix+"/") || namespace == prefix {
				if write {
					return scope.Write
				}
				return scope.Read
			}
		}
	}

	return false
}

// RevokeToken removes a token
func (s *Service) RevokeToken(tokenStr string) error {
	return s.storage.DeleteToken(tokenStr)
}

// ListTokens returns all tokens
func (s *Service) ListTokens() ([]*models.Token, error) {
	return s.storage.ListTokens()
}

// Middleware returns an HTTP middleware for authentication
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		token, err := s.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Add token to context
		ctx := context.WithValue(r.Context(), tokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTokenFromContext retrieves the token from the request context
func GetTokenFromContext(ctx context.Context) (*models.Token, error) {
	token, ok := ctx.Value(tokenContextKey).(*models.Token)
	if !ok {
		return nil, fmt.Errorf("no token in context")
	}
	return token, nil
}

// InitializeRootToken creates the initial root token if no tokens exist
func (s *Service) InitializeRootToken() (*models.Token, error) {
	// Check if any tokens exist
	tokens, err := s.storage.ListTokens()
	if err != nil {
		return nil, err
	}

	if len(tokens) > 0 {
		return nil, fmt.Errorf("tokens already exist")
	}

	// Create root token with full access (empty scopes = full access)
	rootToken, err := s.GenerateToken([]models.TokenScope{}, nil, map[string]string{
		"name": "root",
		"type": "root",
	})
	if err != nil {
		return nil, err
	}

	return rootToken, nil
}

