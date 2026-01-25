// internal/api/middleware/auth.go
package middleware

import (
	"context"
	"log"
	"net/http"
	"payflow/internal/api/response"
	"payflow/internal/domain"
	"payflow/pkg/utils"
	"strconv"
	"strings"
)

// ContextKey is a custom type to avoid key collisions in the request context.
type ContextKey string

const (
	UserClaimsKey ContextKey = "userClaims"
)

// Claims represents the user data stored in the JWT.
type Claims struct {
	UserID     uint
	BusinessID uint
	Role       domain.UserRole
}

// AuthMiddleware validates the JWT and injects user claims into the request context.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.RespondWithError(w, domain.ErrUnauthorized)
				return
			}
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				response.RespondWithError(w, domain.ErrUnauthorized)
				return
			}

			// Debug logging
			//log.Printf("Validating token: %s", tokenString[:20]+"...")
			//log.Printf("Using JWT secret: %s", jwtSecret[:20]+"...")

			// Use the JWT util to validate
			validatedClaims, err := utils.ValidateToken(tokenString, jwtSecret)
			if err != nil {
				log.Printf("Token validation failed: %v", err)
				response.RespondWithError(w, domain.ErrUnauthorized)
				return
			}

			log.Printf("Token validation successful for user: %s", validatedClaims.UserID)

			// Parse claims into a structured, typed object for the context.
			userID, _ := strconv.ParseUint(validatedClaims.UserID, 10, 32)
			businessID, _ := strconv.ParseUint(validatedClaims.BusinessID, 10, 32)

			claims := &Claims{
				UserID:     uint(userID),
				BusinessID: uint(businessID),
				Role:       domain.UserRole(validatedClaims.Role),
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaimsFromContext is a helper function to safely retrieve claims from the context.
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(UserClaimsKey).(*Claims)
	return claims, ok
}

// RoleMiddleware checks if the user has one of the required roles.
func RoleMiddleware(allowedRoles ...domain.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserClaimsKey).(*Claims)
			if !ok {
				// This should not happen if AuthMiddleware is used before it.
				response.RespondWithError(w, domain.ErrInternalServer)
				return
			}

			for _, allowedRole := range allowedRoles {
				if claims.Role == allowedRole {
					next.ServeHTTP(w, r)
					return
				}
			}

			// User does not have the required role.
			response.RespondWithError(w, domain.ErrForbidden)
		})
	}
}
