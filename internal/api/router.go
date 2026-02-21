// internal/api/router.go
package api

import (
	"encoding/json"
	"net/http"
	"payflow/internal/api/handler"
	"payflow/internal/api/middleware"
	"payflow/internal/config"
	"payflow/internal/domain"
	"payflow/internal/platform/korapay"
	"payflow/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"gorm.io/gorm"
)

// healthResponse represents the /health endpoint response.
type healthResponse struct {
	Status   string            `json:"status"`
	Message  string            `json:"message"`
	Server   string            `json:"server"`
	Database healthDBStatus    `json:"database"`
}

type healthDBStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func healthHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		dbStatus := healthDBStatus{Status: "ok", Message: "connected"}
		sqlDB, err := db.DB()
		if err != nil {
			dbStatus = healthDBStatus{Status: "unavailable", Message: err.Error()}
		} else if err := sqlDB.Ping(); err != nil {
			dbStatus = healthDBStatus{Status: "unavailable", Message: err.Error()}
		}

		healthy := dbStatus.Status == "ok"
		resp := healthResponse{
			Server:   "ok",
			Database: dbStatus,
		}
		if healthy {
			resp.Status = "healthy"
			resp.Message = "Server is running. All systems operational."
		} else {
			resp.Status = "unhealthy"
			resp.Message = "Server is running but database is unreachable."
		}

		statusCode := http.StatusOK
		if !healthy {
			statusCode = http.StatusServiceUnavailable
		}
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// NewRouter initializes and returns the main application router.
// It takes the application config and all required services as dependencies.
func NewRouter(
	cfg *config.Config,
	db *gorm.DB,
	authSvc service.AuthService,
	employeeSvc service.EmployeeService,
	cadreSvc service.CadreService,
	deductionSvc service.DeductionRuleService,
	payrollSvc service.PayrollService,
	webhookSvc service.VFDWebhookService,
	transferSvc service.VFDTransferService,
	bulkTransferSvc service.BulkTransferService,
	newTransferSvc service.TransferService,
	walletSvc service.WalletService,
	accountHolderSvc service.AccountHolderService,
	koraClient *korapay.Client,
) http.Handler {
	r := chi.NewRouter()

	// --- Global Middleware ---
	// A more secure CORS policy for production would be:
	// AllowedOrigins: []string{"https://app.payflow.com", "https://www.payflow.com"}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://*", "https://*"}, // Permissive for local dev
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// --- Handler Initialization ---
	// Instantiate all handlers here to keep the routing section clean.
	authHandler := handler.NewAuthHandler(authSvc)
	employeeHandler := handler.NewEmployeeHandler(employeeSvc)
	cadreHandler := handler.NewCadreHandler(cadreSvc)
	deductionHandler := handler.NewDeductionRuleHandler(deductionSvc)
	payrollHandler := handler.NewPayrollHandler(payrollSvc, employeeSvc)
	webhookHandler := handler.NewVFDWebhookHandler(webhookSvc)
	transferHandler := handler.NewVFDTransferHandler(transferSvc)
	bulkTransferHandler := handler.NewBulkTransferHandler(bulkTransferSvc)
	newTransferHandler := handler.NewTransferHandler(newTransferSvc)
	walletHandler := handler.NewWalletHandler(walletSvc, accountHolderSvc, cfg, koraClient)

	// --- Health Check ---
	// Used by load balancers and deployment verification (no auth required).
	r.Get("/health", healthHandler(db))

	// --- Public Routes ---
	// No authentication required for these endpoints.
	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.RegisterBusiness)
		r.Post("/login", authHandler.Login)
		// r.Post("/forgot-password", authHandler.ForgotPassword)
		// r.Post("/reset-password", authHandler.ResetPassword)
	})

	// --- VFD Webhook Routes ---
	// These are public endpoints that VFD will call
	r.Route("/vfd/webhooks", func(r chi.Router) {
		r.Post("/inward-credit", webhookHandler.HandleInwardCreditWebhook)
		r.Post("/initial-inward-credit", webhookHandler.HandleInitialInwardCreditWebhook)
		r.Post("/retrigger", webhookHandler.RetriggerWebhook)
	})

	// --- KoraPay Webhook Routes ---
	// These are public endpoints that KoraPay will call
	r.Route("/korapay/webhooks", func(r chi.Router) {
		r.Post("/deposit", walletHandler.HandleDepositWebhook)
	})

	// --- Protected API v1 Group ---
	// All routes within this group require a valid JWT.
	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret))

		// --- Admin & Operator Routes ---
		// These routes are for day-to-day operations.
		r.Group(func(r chi.Router) {
			r.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleOperator))

			// Employee Management
			r.Route("/employees", func(r chi.Router) {
				r.Post("/", employeeHandler.CreateEmployee)
				r.Get("/", employeeHandler.ListEmployees)
				r.Get("/{employeeID}", employeeHandler.GetEmployeeByID)
				r.Put("/{employeeID}", employeeHandler.UpdateEmployee)
				r.Patch("/{employeeID}/deactivate", employeeHandler.DeactivateEmployee)
			})

			// Cadre (Salary Structure) Management
			r.Route("/cadres", func(r chi.Router) {
				r.Post("/", cadreHandler.CreateCadre)
				r.Get("/", cadreHandler.ListCadres)
				r.Get("/{cadreID}", cadreHandler.GetCadreByID)
				r.Put("/{cadreID}", cadreHandler.UpdateCadre)
				r.Delete("/{cadreID}", cadreHandler.DeleteCadre)
			})
		})

		// --- Payroll Workflow ---
		// Accessible by Admin, Operator, and Approver roles
		r.Route("/payroll-runs", func(r chi.Router) {
			// Admin and Operator can create, list, view, submit, and process payroll
			r.Group(func(r chi.Router) {
				r.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleOperator))
				r.Post("/", payrollHandler.CreatePayrollRun)
				r.Post("/{runID}/submit", payrollHandler.SubmitForApproval)
				r.Post("/{runID}/process-now", payrollHandler.ProcessPayrollRunInstantly)
			})

			// All authenticated roles can view payroll runs
			r.Group(func(r chi.Router) {
				r.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleOperator, domain.RoleApprover))
				r.Get("/", payrollHandler.ListPayrollRuns)
				r.Get("/{runID}", payrollHandler.GetPayrollRunByID)
			})

			// Admin and Approver can approve/reject payroll
			r.Group(func(r chi.Router) {
				r.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleApprover))
				r.Post("/{runID}/approve", payrollHandler.ApprovePayrollRun)
				r.Post("/{runID}/reject", payrollHandler.RejectPayrollRun)
			})
		})

		// --- Admin-Only Routes ---
		// These routes are for company-level configuration.
		r.Group(func(r chi.Router) {
			r.Use(middleware.RoleMiddleware(domain.RoleAdmin))

			// Deduction Rule Management
			r.Route("/deduction-rules", func(r chi.Router) {
				r.Post("/", deductionHandler.CreateDeductionRule)
				r.Get("/", deductionHandler.ListDeductionRules)
				r.Put("/{ruleID}", deductionHandler.UpdateDeductionRule)
				r.Delete("/{ruleID}", deductionHandler.DeleteDeductionRule)
			})

			// User/Team Management could go here
			// r.Route("/users", ...)
		})

		// --- Webhook Management Routes ---
		// These routes require authentication and are for viewing webhook notifications
		r.Route("/vfd/webhooks", func(r chi.Router) {
			r.Get("/", webhookHandler.ListWebhookNotifications)
			r.Get("/{id}", webhookHandler.GetWebhookNotificationByID)
			r.Get("/account/{accountNumber}", webhookHandler.GetWebhookNotificationsByAccountNumber)
		})

		// --- VFD Transfer Routes ---
		// These routes require authentication and are for transfer operations
		r.Route("/vfd/transfers", func(r chi.Router) {
			r.Get("/account-enquiry", transferHandler.HandleAccountEnquiry)
			r.Get("/beneficiary-enquiry", transferHandler.HandleBeneficiaryEnquiry)
			r.Get("/banks", transferHandler.HandleGetBankList)
			r.Post("/initiate", transferHandler.HandleInitiateTransfer)
			r.Get("/", transferHandler.HandleListTransfers)
			r.Get("/{id}", transferHandler.HandleGetTransferByID)
			r.Get("/from-account", transferHandler.HandleGetTransfersByFromAccount)
			r.Get("/to-account", transferHandler.HandleGetTransfersByToAccount)
		})

		// --- Bulk Transfer Routes (Legacy - Deprecated) ---
		// These routes require authentication and are for bulk transfer operations
		r.Route("/bulk-transfers", func(r chi.Router) {
			r.Post("/single", bulkTransferHandler.HandleSingleTransfer)
			r.Post("/batch", bulkTransferHandler.HandleBatchTransfer)
			r.Post("/flow-data", bulkTransferHandler.HandleGetTransferFlowData)
		})

		// --- Transfer Routes (New - Provider-Agnostic) ---
		// Clean, provider-agnostic transfer API with Korapay as primary provider
		r.Route("/transfers", func(r chi.Router) {
			r.Post("/", newTransferHandler.HandleSingleTransfer)
			r.Post("/batch", newTransferHandler.HandleBatchTransfer)
			r.Get("/", newTransferHandler.HandleListTransfers)
			r.Get("/{id}", newTransferHandler.HandleGetTransfer)
		})

		// --- Wallet Routes ---
		// Wallet and virtual account management
		r.Route("/wallets", func(r chi.Router) {
			// Virtual Account Management
			r.Post("/virtual-account", walletHandler.HandleCreateVirtualAccount)
			r.Get("/", walletHandler.HandleGetWallet)
			r.Get("/balance", walletHandler.HandleGetBalance)
			r.Get("/transactions", walletHandler.HandleGetTransactions)

			// Account Holder / KYC Management
			r.Route("/account-holders", func(r chi.Router) {
				r.Post("/", walletHandler.HandleCreateAccountHolder)
				r.Get("/{reference}/details", walletHandler.HandleGetAccountHolderDetails)
				r.Patch("/{reference}/update-kyc", walletHandler.HandleUpdateAccountHolderKYC)
			})

			// File Upload URL Generation
			r.Post("/files/generate-upload-url", walletHandler.HandleGenerateFileUploadURL)

			// Sandbox-only routes (admin only for safety)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RoleMiddleware(domain.RoleAdmin))
				r.Post("/sandbox/credit", walletHandler.HandleSandboxCredit)
			})
		})
	})

	return r
}
