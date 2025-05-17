package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ringo380/lessoncraft/api/middleware"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserContextKey is the key for storing user information in the request context
	UserContextKey contextKey = "user"
	// RolesContextKey is the key for storing user roles in the request context
	RolesContextKey contextKey = "roles"
)

// AuthMiddleware creates a middleware that validates JWT tokens and extracts user information
func AuthMiddleware(jwtService *JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(middleware.ErrorResponse{
					Error:     "Unauthorized",
					Code:      http.StatusUnauthorized,
					Message:   "Missing authorization token",
					TimeStamp: time.Now(),
				})
				return
			}

			// Check if the Authorization header has the correct format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(middleware.ErrorResponse{
					Error:     "Unauthorized",
					Code:      http.StatusUnauthorized,
					Message:   "Invalid authorization header format",
					TimeStamp: time.Now(),
				})
				return
			}

			// Extract the token
			tokenString := parts[1]

			// Validate the token
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				var status int
				var message string

				switch err {
				case ErrExpiredToken:
					status = http.StatusUnauthorized
					message = "Token has expired"
				case ErrInvalidToken:
					status = http.StatusUnauthorized
					message = "Invalid token"
				case ErrInvalidClaims:
					status = http.StatusUnauthorized
					message = "Invalid token claims"
				default:
					status = http.StatusInternalServerError
					message = "Error validating token"
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				json.NewEncoder(w).Encode(middleware.ErrorResponse{
					Error:     "Unauthorized",
					Code:      status,
					Message:   message,
					Details:   err.Error(),
					TimeStamp: time.Now(),
				})
				return
			}

			// Add user information to the request context
			ctx := context.WithValue(r.Context(), UserContextKey, claims.UserID)
			ctx = context.WithValue(ctx, RolesContextKey, claims.Roles)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RoleMiddleware creates a middleware that checks if the user has the required role
func RoleMiddleware(requiredRole Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get roles from context
			rolesInterface := r.Context().Value(RolesContextKey)
			if rolesInterface == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(middleware.ErrorResponse{
					Error:     "Unauthorized",
					Code:      http.StatusUnauthorized,
					Message:   "User not authenticated",
					TimeStamp: time.Now(),
				})
				return
			}

			// Convert to []Role
			roles, ok := rolesInterface.([]Role)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(middleware.ErrorResponse{
					Error:     "InternalServerError",
					Code:      http.StatusInternalServerError,
					Message:   "Invalid role format in context",
					TimeStamp: time.Now(),
				})
				return
			}

			// Check if the user has the required role
			hasRole := false
			for _, role := range roles {
				if role == requiredRole {
					hasRole = true
					break
				}
			}

			if !hasRole {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(middleware.ErrorResponse{
					Error:     "Forbidden",
					Code:      http.StatusForbidden,
					Message:   "Insufficient permissions",
					TimeStamp: time.Now(),
				})
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID extracts the user ID from the request context
func GetUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(UserContextKey).(string)
	return userID, ok
}

// GetUserRoles extracts the user roles from the request context
func GetUserRoles(r *http.Request) ([]Role, bool) {
	roles, ok := r.Context().Value(RolesContextKey).([]Role)
	return roles, ok
}

// HasRole checks if the user has a specific role
func HasRole(r *http.Request, role Role) bool {
	roles, ok := GetUserRoles(r)
	if !ok {
		return false
	}

	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin checks if the user has the admin role
func IsAdmin(r *http.Request) bool {
	return HasRole(r, RoleAdmin)
}

// IsEducator checks if the user has the educator role
func IsEducator(r *http.Request) bool {
	return HasRole(r, RoleEducator)
}

// IsLearner checks if the user has the learner role
func IsLearner(r *http.Request) bool {
	return HasRole(r, RoleLearner)
}
