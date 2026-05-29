package middleware

import (
	"encoding/json"
	"net/http"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// PlanLimitMiddleware checks if the business has exceeded its plan limits.
// Applies to employee creation and payroll run creation endpoints.
func PlanLimitMiddleware(
	businessRepo repository.BusinessRepository,
	employeeRepo repository.EmployeeRepository,
	planRepo repository.SubscriptionPlanRepository,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check on creation endpoints
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

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

			plan, err := planRepo.FindByTier(r.Context(), domain.PlanTier(business.SubscriptionTier))
			if err != nil || plan == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Check employee limit
			if plan.MaxEmployees > 0 {
				employees, err := employeeRepo.FindByBusinessID(r.Context(), claims.BusinessID)
				if err == nil && len(employees) >= plan.MaxEmployees {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":         "Plan limit reached",
						"message":       "You have reached the maximum number of employees for your plan. Upgrade to add more.",
						"current_count": len(employees),
						"max_allowed":   plan.MaxEmployees,
						"current_plan":  plan.Tier,
					})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
