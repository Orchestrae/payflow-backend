package middleware

import (
	"net/http"

	"payflow/internal/repository"
)

// SuspensionMiddleware checks if the business is suspended.
// Returns 403 if the org is suspended.
func SuspensionMiddleware(businessRepo repository.BusinessRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaimsFromContext(r.Context())
			if !ok || claims.BusinessID == 0 {
				next.ServeHTTP(w, r)
				return
			}

			business, err := businessRepo.FindByID(r.Context(), claims.BusinessID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if business.IsSuspended {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"Account suspended. Please contact support or update your subscription."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
