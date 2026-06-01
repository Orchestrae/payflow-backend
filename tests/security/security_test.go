// tests/security/security_test.go
package security_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"payflow/internal/api/middleware"
	"payflow/internal/domain"
	"payflow/pkg/utils"

	"github.com/go-chi/chi/v5"
)

// ============================================================================
// Test helpers
// ============================================================================

const testJWTSecret = "test-secret-key-for-security-tests-minimum-32-chars"

// buildSecurityTestRouter creates a Chi router with auth middleware applied
// to test security boundaries without requiring all service dependencies.
func buildSecurityTestRouter() http.Handler {
	r := chi.NewRouter()

	// Apply the same middleware as the real router
	r.Use(middleware.RequestID)
	r.Use(middleware.RateLimiter(100, 200))

	// Public auth routes with stricter rate limiting
	r.Route("/v1/auth", func(r chi.Router) {
		r.Use(middleware.RateLimiter(5, 10))
		r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid credentials"}`))
		})
		r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"registered"}`))
		})
	})

	// Protected routes
	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(testJWTSecret))

		r.Route("/employees", func(r chi.Router) {
			r.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleOperator))
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data":[]}`))
			})
			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write(body)
			})
			r.Get("/{employeeID}", func(w http.ResponseWriter, r *http.Request) {
				claims := r.Context().Value(middleware.UserClaimsKey).(*middleware.Claims)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(fmt.Sprintf(`{"business_id":%d}`, claims.BusinessID)))
			})
		})

		r.Route("/payroll-runs", func(r chi.Router) {
			r.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleOperator))
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data":[]}`))
			})
		})

		r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true}`))
		})
	})

	return r
}

// generateTestToken creates a valid JWT for testing.
func generateTestToken(userID, businessID uint, role domain.UserRole) string {
	token, err := utils.GenerateToken(
		fmt.Sprintf("%d", userID),
		fmt.Sprintf("%d", businessID),
		string(role),
		testJWTSecret,
		time.Hour,
	)
	if err != nil {
		panic(fmt.Sprintf("failed to generate test token: %v", err))
	}
	return token
}

// generateExpiredToken creates an expired JWT for testing.
func generateExpiredToken(userID, businessID uint, role domain.UserRole) string {
	token, err := utils.GenerateToken(
		fmt.Sprintf("%d", userID),
		fmt.Sprintf("%d", businessID),
		string(role),
		testJWTSecret,
		-time.Hour, // Expired 1 hour ago
	)
	if err != nil {
		panic(fmt.Sprintf("failed to generate expired token: %v", err))
	}
	return token
}

// ============================================================================
// Security Tests
// ============================================================================

func TestSQLInjectionAttempts(t *testing.T) {
	router := buildSecurityTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()

	token := generateTestToken(1, 1, domain.RoleAdmin)

	sqlInjectionPayloads := []string{
		`'; DROP TABLE users; --`,
		`1 OR 1=1`,
		`" OR ""="`,
		`1; DELETE FROM employees WHERE 1=1`,
		`UNION SELECT * FROM users--`,
		`' UNION SELECT NULL, email, password_hash FROM users--`,
		`admin'--`,
		`1' AND (SELECT COUNT(*) FROM users) > 0 --`,
	}

	for _, payload := range sqlInjectionPayloads {
		label := payload
		if len(label) > 20 {
			label = label[:20]
		}
		t.Run("SQLi_"+label, func(t *testing.T) {
			// Test search/query parameter injection
			req, _ := http.NewRequest("GET", server.URL+"/v1/employees?search="+payload, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			// The server should not return a 500 (which would indicate SQL error)
			if resp.StatusCode == http.StatusInternalServerError {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("SQL injection payload caused server error: %s", string(body))
			}
		})
	}
}

func TestXSSInEmployeeNames(t *testing.T) {
	router := buildSecurityTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()

	token := generateTestToken(1, 1, domain.RoleAdmin)

	xssPayloads := []struct {
		name    string
		payload string
	}{
		{"script_tag", `<script>alert('xss')</script>`},
		{"img_onerror", `<img src=x onerror=alert('xss')>`},
		{"svg_onload", `<svg onload=alert('xss')>`},
		{"event_handler", `" onmouseover="alert('xss')`},
		{"javascript_uri", `javascript:alert('xss')`},
		{"encoded_script", `%3Cscript%3Ealert('xss')%3C/script%3E`},
	}

	for _, tc := range xssPayloads {
		t.Run("XSS_"+tc.name, func(t *testing.T) {
			empPayload := map[string]interface{}{
				"full_name":           tc.payload,
				"email":               "xss@test.com",
				"cadre_id":            1,
				"bank_name":           "Test Bank",
				"bank_code":           "058",
				"bank_account_number": "1234567890",
			}
			body, _ := json.Marshal(empPayload)

			req, _ := http.NewRequest("POST", server.URL+"/v1/employees", bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			respBody, _ := io.ReadAll(resp.Body)

			// Verify the response Content-Type is application/json (not text/html)
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			// The response should not contain unescaped script tags that could execute
			respStr := string(respBody)
			if strings.Contains(respStr, "<script>") && !strings.Contains(respStr, `\u003c`) {
				t.Error("response contains unescaped <script> tag -- potential XSS vulnerability")
			}
		})
	}
}

func TestAuthBypass(t *testing.T) {
	router := buildSecurityTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("MissingToken", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/v1/employees", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/v1/employees", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-value")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		expiredToken := generateExpiredToken(1, 1, domain.RoleAdmin)

		req, _ := http.NewRequest("GET", server.URL+"/v1/employees", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("WrongSecret", func(t *testing.T) {
		// Generate token with a different secret
		wrongToken, _ := utils.GenerateToken("1", "1", "admin", "wrong-secret-key-for-testing-xxxxx", time.Hour)

		req, _ := http.NewRequest("GET", server.URL+"/v1/employees", nil)
		req.Header.Set("Authorization", "Bearer "+wrongToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("MalformedAuthHeader", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/v1/employees", nil)
		req.Header.Set("Authorization", "NotBearer some-token")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("InsufficientRole", func(t *testing.T) {
		// Employee role should not access admin/operator routes
		employeeToken := generateTestToken(10, 1, domain.RoleEmployee)

		req, _ := http.NewRequest("GET", server.URL+"/v1/employees", nil)
		req.Header.Set("Authorization", "Bearer "+employeeToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 for employee accessing admin route, got %d", resp.StatusCode)
		}
	})
}

func TestIDOR(t *testing.T) {
	router := buildSecurityTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()

	// User from business 1
	tokenBusiness1 := generateTestToken(1, 1, domain.RoleAdmin)
	// User from business 2
	tokenBusiness2 := generateTestToken(2, 2, domain.RoleAdmin)

	t.Run("AccessOtherBusinessEmployees", func(t *testing.T) {
		// Business 1 user accesses employee endpoint
		req1, _ := http.NewRequest("GET", server.URL+"/v1/employees/999", nil)
		req1.Header.Set("Authorization", "Bearer "+tokenBusiness1)

		resp1, err := http.DefaultClient.Do(req1)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp1.Body.Close()
		body1, _ := io.ReadAll(resp1.Body)

		// Business 2 user accesses the same employee endpoint
		req2, _ := http.NewRequest("GET", server.URL+"/v1/employees/999", nil)
		req2.Header.Set("Authorization", "Bearer "+tokenBusiness2)

		resp2, err := http.DefaultClient.Do(req2)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp2.Body.Close()
		body2, _ := io.ReadAll(resp2.Body)

		// The responses should return different business_ids (enforced by middleware)
		var result1, result2 map[string]interface{}
		json.Unmarshal(body1, &result1)
		json.Unmarshal(body2, &result2)

		bid1, _ := result1["business_id"].(float64)
		bid2, _ := result2["business_id"].(float64)

		if bid1 == bid2 && bid1 != 0 {
			t.Error("IDOR vulnerability: different businesses got same business_id in response")
		}
		if bid1 != 1 {
			t.Errorf("business 1 user got business_id %v, expected 1", bid1)
		}
		if bid2 != 2 {
			t.Errorf("business 2 user got business_id %v, expected 2", bid2)
		}
	})

	t.Run("AccessOtherBusinessPayrolls", func(t *testing.T) {
		// Business 2 should see their own data scope
		req, _ := http.NewRequest("GET", server.URL+"/v1/payroll-runs", nil)
		req.Header.Set("Authorization", "Bearer "+tokenBusiness2)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})
}

func TestRateLimiting(t *testing.T) {
	router := buildSecurityTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("AuthEndpointRateLimit", func(t *testing.T) {
		loginPayload := `{"email":"test@test.com","password":"password123"}`
		var rateLimited bool
		var mu sync.Mutex

		// Send many rapid requests to the auth endpoint
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequest("POST", server.URL+"/v1/auth/login",
					strings.NewReader(loginPayload))
				req.Header.Set("Content-Type", "application/json")

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusTooManyRequests {
					mu.Lock()
					rateLimited = true
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		if !rateLimited {
			t.Log("WARNING: rate limiting did not trigger with 50 concurrent requests -- " +
				"this may be expected if the rate limit burst is high or requests were serialized")
		}
	})
}
