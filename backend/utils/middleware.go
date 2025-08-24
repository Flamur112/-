package utils

import (
	"context"
	"net/http"
	"strings"
)

// AuthMiddleware creates middleware for JWT authentication
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "No authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Extract user info
		userID, username, role, err := ExtractUserFromJWT(claims)
		if err != nil {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Add user info to request context
		ctx := context.WithValue(r.Context(), "user_id", userID)
		ctx = context.WithValue(ctx, "username", username)
		ctx = context.WithValue(ctx, "role", role)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RoleMiddleware creates middleware to check user roles
func RoleMiddleware(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole := r.Context().Value("role").(string)

			// Check if user role is allowed
			allowed := false
			for _, role := range allowedRoles {
				if role == userRole {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
