// tests/e2e/contract_test.go
package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"payflow/internal/api/middleware"
	"payflow/internal/domain"
	"payflow/pkg/utils"

	"github.com/go-chi/chi/v5"
)

// ============================================================================
// Test helpers and mock router
// ============================================================================

const e2eJWTSecret = "e2e-test-secret-key-for-contract-tests-minimum-32"

// buildE2ERouter creates a contract-test router that returns expected JSON shapes.
// This simulates the API contract without requiring real service dependencies.
func buildE2ERouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)

	// --- Auth Routes (public) ---
	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/register", handleMockRegister)
		r.Post("/login", handleMockLogin)
	})

	// --- Protected Routes ---
	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(e2eJWTSecret))

		// Employee CRUD
		r.Route("/employees", func(r chi.Router) {
			r.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleOperator))
			r.Post("/", handleMockCreateEmployee)
			r.Get("/", handleMockListEmployees)
			r.Get("/{employeeID}", handleMockGetEmployee)
			r.Put("/{employeeID}", handleMockUpdateEmployee)
			r.Patch("/{employeeID}/deactivate", handleMockDeactivateEmployee)
		})

		// Payroll lifecycle
		r.Route("/payroll-runs", func(r chi.Router) {
			r.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleOperator, domain.RoleApprover))
			r.Post("/", handleMockCreatePayroll)
			r.Get("/", handleMockListPayrolls)
			r.Get("/{runID}", handleMockGetPayroll)
			r.Post("/{runID}/submit", handleMockSubmitPayroll)
			r.Post("/{runID}/approve", handleMockApprovePayroll)
			r.Get("/{runID}/status", handleMockGetPayrollStatus)
		})

		// Leave management
		r.Route("/leave", func(r chi.Router) {
			r.Post("/types", handleMockCreateLeaveType)
			r.Get("/types", handleMockListLeaveTypes)
			r.Post("/requests", handleMockSubmitLeaveRequest)
			r.Post("/requests/{id}/approve", handleMockApproveLeave)
		})

		// Wallet
		r.Route("/wallets", func(r chi.Router) {
			r.Get("/", handleMockGetWallet)
			r.Get("/balance", handleMockGetBalance)
		})
	})

	return r
}

// --- Mock handlers that return contract-compliant JSON ---

func handleMockRegister(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	token, _ := utils.GenerateToken("1", "1", "admin", e2eJWTSecret, time.Hour)

	resp := map[string]interface{}{
		"message": "Business registered successfully",
		"token":   token,
		"user": map[string]interface{}{
			"id":          1,
			"email":       req["email"],
			"role":        "admin",
			"business_id": 1,
			"is_verified": false,
			"created_at":  time.Now().Format(time.RFC3339),
		},
		"business": map[string]interface{}{
			"id":       1,
			"name":     req["business_name"],
			"currency": "NGN",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func handleMockLogin(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	token, _ := utils.GenerateToken("1", "1", "admin", e2eJWTSecret, time.Hour)

	resp := map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":          1,
			"email":       req["email"],
			"role":        "admin",
			"business_id": 1,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func handleMockCreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":                  1,
			"business_id":        1,
			"cadre_id":           req["cadre_id"],
			"full_name":          req["full_name"],
			"email":              req["email"],
			"bank_name":          req["bank_name"],
			"bank_code":          req["bank_code"],
			"bank_account_number": req["bank_account_number"],
			"is_active":          true,
			"created_at":         time.Now().Format(time.RFC3339),
			"updated_at":         time.Now().Format(time.RFC3339),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func handleMockListEmployees(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"id":         1,
				"full_name":  "John Doe",
				"email":      "john@test.com",
				"is_active":  true,
				"cadre_id":   1,
				"created_at": time.Now().Format(time.RFC3339),
			},
		},
		"total": 1,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockGetEmployee(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":                  1,
			"business_id":        1,
			"full_name":          "John Doe",
			"email":              "john@test.com",
			"is_active":          true,
			"cadre_id":           1,
			"bank_name":          "Test Bank",
			"bank_code":          "058",
			"bank_account_number": "1234567890",
			"created_at":         time.Now().Format(time.RFC3339),
			"updated_at":         time.Now().Format(time.RFC3339),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockUpdateEmployee(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":         1,
			"full_name":  req["full_name"],
			"email":      req["email"],
			"is_active":  true,
			"updated_at": time.Now().Format(time.RFC3339),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockDeactivateEmployee(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"message": "Employee deactivated successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockCreatePayroll(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":               1,
			"business_id":     1,
			"period":          "2026-06-01T00:00:00Z",
			"status":          "draft",
			"total_gross_pay": 50000000,
			"total_deductions": 10000000,
			"total_net_pay":   40000000,
			"entries":         []interface{}{},
			"created_at":      time.Now().Format(time.RFC3339),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func handleMockListPayrolls(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"id":              1,
				"business_id":    1,
				"period":         "2026-06-01T00:00:00Z",
				"status":         "draft",
				"total_net_pay":  40000000,
				"created_at":     time.Now().Format(time.RFC3339),
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockGetPayroll(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":               1,
			"business_id":     1,
			"period":          "2026-06-01T00:00:00Z",
			"status":          "draft",
			"total_gross_pay": 50000000,
			"total_deductions": 10000000,
			"total_net_pay":   40000000,
			"entries":         []interface{}{},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockSubmitPayroll(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":     1,
			"status": "pending_approval",
		},
		"message": "Payroll submitted for approval",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockApprovePayroll(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":     1,
			"status": "approved",
		},
		"message": "Payroll approved",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockGetPayrollStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":     1,
			"status": "approved",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockCreateLeaveType(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":                1,
			"business_id":      1,
			"name":             req["name"],
			"default_days":     req["default_days"],
			"requires_approval": true,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func handleMockListLeaveTypes(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"id":            1,
				"name":          "Annual Leave",
				"default_days":  20,
				"requires_approval": true,
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockSubmitLeaveRequest(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":            1,
			"employee_id":   req["employee_id"],
			"leave_type_id": req["leave_type_id"],
			"start_date":    req["start_date"],
			"end_date":      req["end_date"],
			"days":          req["days"],
			"reason":        req["reason"],
			"status":        "pending",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func handleMockApproveLeave(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"message": "Leave request approved",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockGetWallet(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":                       1,
			"business_id":             1,
			"balance":                 5000000,
			"locked_balance":          0,
			"currency":                "NGN",
			"virtual_account_number":  "1234567890",
			"virtual_account_bank_name": "Test Bank",
			"virtual_account_status":  "active",
			"provider":                "korapay",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMockGetBalance(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"balance":           5000000,
			"available_balance": 5000000,
			"currency":          "NGN",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// Contract validation helpers
// ============================================================================

func requireField(t *testing.T, data map[string]interface{}, field string) {
	t.Helper()
	if _, ok := data[field]; !ok {
		t.Errorf("missing required field %q in response", field)
	}
}

func requireFieldType(t *testing.T, data map[string]interface{}, field string, expectedType string) {
	t.Helper()
	val, ok := data[field]
	if !ok {
		t.Errorf("missing required field %q", field)
		return
	}
	switch expectedType {
	case "string":
		if _, ok := val.(string); !ok {
			t.Errorf("field %q: expected string, got %T", field, val)
		}
	case "number":
		if _, ok := val.(float64); !ok {
			t.Errorf("field %q: expected number, got %T", field, val)
		}
	case "bool":
		if _, ok := val.(bool); !ok {
			t.Errorf("field %q: expected bool, got %T", field, val)
		}
	case "array":
		if _, ok := val.([]interface{}); !ok {
			t.Errorf("field %q: expected array, got %T", field, val)
		}
	case "object":
		if _, ok := val.(map[string]interface{}); !ok {
			t.Errorf("field %q: expected object, got %T", field, val)
		}
	}
}

func generateE2EToken(userID, businessID uint, role domain.UserRole) string {
	token, _ := utils.GenerateToken(
		fmt.Sprintf("%d", userID),
		fmt.Sprintf("%d", businessID),
		string(role),
		e2eJWTSecret,
		time.Hour,
	)
	return token
}

func doRequest(t *testing.T, server *httptest.Server, method, path, token string, body interface{}) (int, map[string]interface{}) {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}

	req, _ := http.NewRequest(method, server.URL+path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	return resp.StatusCode, result
}

// ============================================================================
// E2E Contract Tests
// ============================================================================

func TestAuthFlow(t *testing.T) {
	server := httptest.NewServer(buildE2ERouter())
	defer server.Close()

	t.Run("Register", func(t *testing.T) {
		registerBody := map[string]interface{}{
			"business_name":     "Contract Test Corp",
			"email":             "admin@contracttest.com",
			"password":          "SecureP@ss123",
			"rc_number":         "RC123456",
			"incorporation_date": "2020-01-01T00:00:00Z",
			"director_bvn":      "12345678901",
		}

		status, result := doRequest(t, server, "POST", "/v1/auth/register", "", registerBody)

		if status != http.StatusCreated {
			t.Fatalf("expected 201, got %d", status)
		}

		// Validate response contract
		requireField(t, result, "token")
		requireField(t, result, "user")
		requireField(t, result, "message")
		requireFieldType(t, result, "token", "string")
		requireFieldType(t, result, "user", "object")

		user := result["user"].(map[string]interface{})
		requireField(t, user, "id")
		requireField(t, user, "email")
		requireField(t, user, "role")
		requireField(t, user, "business_id")
	})

	t.Run("Login", func(t *testing.T) {
		loginBody := map[string]interface{}{
			"email":    "admin@contracttest.com",
			"password": "SecureP@ss123",
		}

		status, result := doRequest(t, server, "POST", "/v1/auth/login", "", loginBody)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		// Validate response contract
		requireField(t, result, "token")
		requireField(t, result, "user")
		requireFieldType(t, result, "token", "string")

		user := result["user"].(map[string]interface{})
		requireField(t, user, "id")
		requireField(t, user, "email")
		requireField(t, user, "role")
	})
}

func TestEmployeeCRUD(t *testing.T) {
	server := httptest.NewServer(buildE2ERouter())
	defer server.Close()

	token := generateE2EToken(1, 1, domain.RoleAdmin)

	t.Run("Create", func(t *testing.T) {
		empBody := map[string]interface{}{
			"full_name":           "Jane Smith",
			"email":               "jane@test.com",
			"cadre_id":            1,
			"bank_name":           "GTBank",
			"bank_code":           "058",
			"bank_account_number": "0123456789",
		}

		status, result := doRequest(t, server, "POST", "/v1/employees", token, empBody)

		if status != http.StatusCreated {
			t.Fatalf("expected 201, got %d", status)
		}

		data := result["data"].(map[string]interface{})
		requireField(t, data, "id")
		requireField(t, data, "full_name")
		requireField(t, data, "email")
		requireField(t, data, "cadre_id")
		requireField(t, data, "is_active")
		requireField(t, data, "created_at")
	})

	t.Run("List", func(t *testing.T) {
		status, result := doRequest(t, server, "GET", "/v1/employees", token, nil)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		requireField(t, result, "data")
		requireFieldType(t, result, "data", "array")
	})

	t.Run("GetByID", func(t *testing.T) {
		status, result := doRequest(t, server, "GET", "/v1/employees/1", token, nil)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		data := result["data"].(map[string]interface{})
		requireField(t, data, "id")
		requireField(t, data, "full_name")
		requireField(t, data, "email")
		requireField(t, data, "bank_name")
		requireField(t, data, "bank_account_number")
	})

	t.Run("Update", func(t *testing.T) {
		updateBody := map[string]interface{}{
			"full_name": "Jane Updated Smith",
			"email":     "jane.updated@test.com",
		}

		status, result := doRequest(t, server, "PUT", "/v1/employees/1", token, updateBody)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		data := result["data"].(map[string]interface{})
		requireField(t, data, "full_name")
		requireField(t, data, "updated_at")
	})

	t.Run("Deactivate", func(t *testing.T) {
		status, result := doRequest(t, server, "PATCH", "/v1/employees/1/deactivate", token, nil)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		requireField(t, result, "message")
	})
}

func TestPayrollLifecycle(t *testing.T) {
	server := httptest.NewServer(buildE2ERouter())
	defer server.Close()

	token := generateE2EToken(1, 1, domain.RoleAdmin)

	t.Run("Create", func(t *testing.T) {
		payrollBody := map[string]interface{}{
			"period":      "2026-06-01T00:00:00Z",
			"adjustments": map[string]interface{}{},
		}

		status, result := doRequest(t, server, "POST", "/v1/payroll-runs", token, payrollBody)

		if status != http.StatusCreated {
			t.Fatalf("expected 201, got %d", status)
		}

		data := result["data"].(map[string]interface{})
		requireField(t, data, "id")
		requireField(t, data, "status")
		requireField(t, data, "total_gross_pay")
		requireField(t, data, "total_net_pay")
		requireFieldType(t, data, "status", "string")

		if data["status"] != "draft" {
			t.Errorf("expected status 'draft', got %v", data["status"])
		}
	})

	t.Run("Submit", func(t *testing.T) {
		status, result := doRequest(t, server, "POST", "/v1/payroll-runs/1/submit", token, nil)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		data := result["data"].(map[string]interface{})
		if data["status"] != "pending_approval" {
			t.Errorf("expected status 'pending_approval', got %v", data["status"])
		}
	})

	t.Run("Approve", func(t *testing.T) {
		status, result := doRequest(t, server, "POST", "/v1/payroll-runs/1/approve", token, nil)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		data := result["data"].(map[string]interface{})
		if data["status"] != "approved" {
			t.Errorf("expected status 'approved', got %v", data["status"])
		}
	})

	t.Run("CheckStatus", func(t *testing.T) {
		status, result := doRequest(t, server, "GET", "/v1/payroll-runs/1/status", token, nil)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		data := result["data"].(map[string]interface{})
		requireField(t, data, "id")
		requireField(t, data, "status")
	})
}

func TestLeaveManagement(t *testing.T) {
	server := httptest.NewServer(buildE2ERouter())
	defer server.Close()

	token := generateE2EToken(1, 1, domain.RoleAdmin)

	t.Run("CreateLeaveType", func(t *testing.T) {
		body := map[string]interface{}{
			"name":              "Annual Leave",
			"default_days":      20,
			"requires_approval": true,
		}

		status, result := doRequest(t, server, "POST", "/v1/leave/types", token, body)

		if status != http.StatusCreated {
			t.Fatalf("expected 201, got %d", status)
		}

		data := result["data"].(map[string]interface{})
		requireField(t, data, "id")
		requireField(t, data, "name")
		requireField(t, data, "default_days")
		requireField(t, data, "requires_approval")
	})

	t.Run("ListLeaveTypes", func(t *testing.T) {
		status, result := doRequest(t, server, "GET", "/v1/leave/types", token, nil)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		requireField(t, result, "data")
		requireFieldType(t, result, "data", "array")
	})

	t.Run("SubmitLeaveRequest", func(t *testing.T) {
		body := map[string]interface{}{
			"employee_id":   1,
			"leave_type_id": 1,
			"start_date":    "2026-07-01T00:00:00Z",
			"end_date":      "2026-07-10T00:00:00Z",
			"days":          7,
			"reason":        "Family vacation",
		}

		status, result := doRequest(t, server, "POST", "/v1/leave/requests", token, body)

		if status != http.StatusCreated {
			t.Fatalf("expected 201, got %d", status)
		}

		data := result["data"].(map[string]interface{})
		requireField(t, data, "id")
		requireField(t, data, "employee_id")
		requireField(t, data, "leave_type_id")
		requireField(t, data, "status")

		if data["status"] != "pending" {
			t.Errorf("expected status 'pending', got %v", data["status"])
		}
	})

	t.Run("ApproveLeaveRequest", func(t *testing.T) {
		status, result := doRequest(t, server, "POST", "/v1/leave/requests/1/approve", token, nil)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		requireField(t, result, "message")
	})
}

func TestWalletOperations(t *testing.T) {
	server := httptest.NewServer(buildE2ERouter())
	defer server.Close()

	token := generateE2EToken(1, 1, domain.RoleAdmin)

	t.Run("GetWallet", func(t *testing.T) {
		status, result := doRequest(t, server, "GET", "/v1/wallets", token, nil)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		data := result["data"].(map[string]interface{})
		requireField(t, data, "id")
		requireField(t, data, "business_id")
		requireField(t, data, "balance")
		requireField(t, data, "currency")
		requireField(t, data, "virtual_account_number")
		requireField(t, data, "virtual_account_status")
		requireField(t, data, "provider")
		requireFieldType(t, data, "balance", "number")
		requireFieldType(t, data, "currency", "string")
	})

	t.Run("CheckBalance", func(t *testing.T) {
		status, result := doRequest(t, server, "GET", "/v1/wallets/balance", token, nil)

		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		data := result["data"].(map[string]interface{})
		requireField(t, data, "balance")
		requireField(t, data, "available_balance")
		requireField(t, data, "currency")
		requireFieldType(t, data, "balance", "number")
		requireFieldType(t, data, "available_balance", "number")
	})
}
