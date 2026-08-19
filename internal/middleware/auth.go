package middleware

import (
	"context"
	"net/http"
	"strings"

	"helpdesk/internal/auth"
)

type contextKey string

const (
	UserContextKey contextKey = "user_claims"
)

// Authenticate extracts and validates JWT from Authorization header or cookie
func Authenticate(jwtService *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			} else if cookie, err := r.Cookie("helpdesk_session"); err == nil {
				tokenStr = cookie.Value
			}

			if tokenStr == "" {
				http.Error(w, `{"error":"unauthorized","message":"missing or invalid authentication token"}`, http.StatusUnauthorized)
				return
			}

			claims, err := jwtService.ValidateToken(tokenStr)
			if err != nil {
				http.Error(w, `{"error":"unauthorized","message":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthenticate adds claims if token exists, but doesn't block anonymous requests
func OptionalAuthenticate(jwtService *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			} else if cookie, err := r.Cookie("helpdesk_session"); err == nil {
				tokenStr = cookie.Value
			}

			if tokenStr != "" {
				if claims, err := jwtService.ValidateToken(tokenStr); err == nil {
					ctx := context.WithValue(r.Context(), UserContextKey, claims)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission checks whether the authenticated user has a specific permission
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*auth.Claims)
			if !ok || claims == nil {
				http.Error(w, `{"error":"unauthorized","message":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			// Admin has wildcard access
			if claims.Role == "admin" {
				next.ServeHTTP(w, r)
				return
			}

			hasPerm := false
			for _, p := range claims.Permissions {
				if p == perm || p == "*" {
					hasPerm = true
					break
				}
			}

			if !hasPerm {
				http.Error(w, `{"error":"forbidden","message":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole checks whether the user has one of the allowed roles
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*auth.Claims)
			if !ok || claims == nil {
				http.Error(w, `{"error":"unauthorized","message":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			matched := false
			for _, r := range roles {
				if claims.Role == r {
					matched = true
					break
				}
			}

			if !matched {
				http.Error(w, `{"error":"forbidden","message":"access restricted by role"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserClaims helper retrieves claims from context
func GetUserClaims(ctx context.Context) *auth.Claims {
	if claims, ok := ctx.Value(UserContextKey).(*auth.Claims); ok {
		return claims
	}
	return nil
}
