// internal/api/router.go
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"payflow/internal/api/handler"
	"payflow/internal/api/middleware"
	"payflow/internal/config"
	"payflow/internal/domain"
	"payflow/internal/platform/cache"
	"payflow/internal/platform/korapay"
	"payflow/internal/repository"
	"payflow/internal/service"
	"payflow/internal/service/platform"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"gorm.io/gorm"
)

// healthResponse represents the /health endpoint response.
type healthResponse struct {
	Status   string          `json:"status"`
	Message  string          `json:"message"`
	Server   string          `json:"server"`
	Database healthCompStatus `json:"database"`
	Redis    *healthCompStatus `json:"redis,omitempty"`
}

type healthCompStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func healthHandler(db *gorm.DB, redisClient *cache.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		dbStatus := healthCompStatus{Status: "ok", Message: "connected"}
		sqlDB, err := db.DB()
		if err != nil {
			dbStatus = healthCompStatus{Status: "unavailable", Message: err.Error()}
		} else if err := sqlDB.Ping(); err != nil {
			dbStatus = healthCompStatus{Status: "unavailable", Message: err.Error()}
		}

		resp := healthResponse{
			Server:   "ok",
			Database: dbStatus,
		}

		healthy := dbStatus.Status == "ok"

		// Check Redis if configured
		if redisClient != nil {
			redisStatus := healthCompStatus{Status: "ok", Message: "connected"}
			if err := redisClient.Ping(r.Context()); err != nil {
				redisStatus = healthCompStatus{Status: "unavailable", Message: err.Error()}
			}
			resp.Redis = &redisStatus
		}

		if healthy {
			resp.Status = "healthy"
			resp.Message = "Server is running. All systems operational."
		} else {
			resp.Status = "unhealthy"
			resp.Message = "Server is running but some services are unreachable."
		}

		statusCode := http.StatusOK
		if !healthy {
			statusCode = http.StatusServiceUnavailable
		}
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// liveHandler is a lightweight liveness probe (just returns 200).
func liveHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"alive"}`))
}

// NewRouter initializes and returns the main application router.
// It takes the application config and all required services as dependencies.
func NewRouter(
	cfg *config.Config,
	db *gorm.DB,
	redisClient *cache.RedisClient,
	authSvc service.AuthService,
	employeeSvc service.EmployeeService,
	cadreSvc service.CadreService,
	deductionSvc service.DeductionRuleService,
	payrollSvc service.PayrollService,
	webhookSvc service.VFDWebhookService,
	transferSvc service.VFDTransferService,
	newTransferSvc service.TransferService,
	walletSvc service.WalletService,
	businessSvc service.BusinessService,
	dashboardSvc service.DashboardService,
	auditSvc service.AuditService,
	notifCenterSvc service.NotificationCenterService,
	verificationSvc service.AccountVerificationService,
	loanSvc service.LoanService,
	billingSvc platform.BillingService,
	platformSvc platform.PlatformService,
	accountHolderSvc service.AccountHolderService,
	koraClient *korapay.Client,
	transferRepo repository.TransferRepository,
	businessRepo repository.BusinessRepository,
) http.Handler {
	r := chi.NewRouter()

	// --- Global Middleware ---
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	// CORS: use config-driven origins
	origins := strings.Split(cfg.CORSAllowedOrigins, ",")
	if cfg.CORSAllowedOrigins == "" {
		origins = []string{"https://payflowio.netlify.app"}
	}
	// Rate limiting: 100 req/sec per IP globally
	r.Use(middleware.RateLimiter(100, 200))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
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
	webhookHandler := handler.NewVFDWebhookHandler(webhookSvc, cfg.VFDWebhookSecret)
	transferHandler := handler.NewVFDTransferHandler(transferSvc)
	newTransferHandler := handler.NewTransferHandler(newTransferSvc)
	businessHandler := handler.NewBusinessHandler(businessSvc)
	dashboardHandler := handler.NewDashboardHandler(dashboardSvc)
	auditHandler := handler.NewAuditHandler(auditSvc)
	notifHandler := handler.NewNotificationHandler(notifCenterSvc)
	verificationHandler := handler.NewVerificationHandler(verificationSvc)
	loanHandler := handler.NewLoanHandler(loanSvc)
	billingHandler := handler.NewBillingHandler(billingSvc)
	platformHandler := handler.NewPlatformHandler(platformSvc)
	selfServiceHandler := handler.NewSelfServiceHandler(employeeSvc, payrollSvc)
	reportHandler := handler.NewReportHandler(payrollSvc, businessRepo)
	walletHandler := handler.NewWalletHandler(walletSvc, accountHolderSvc, cfg, koraClient)

	// --- Health Check ---
	// /health — readiness probe (checks DB + Redis)
	// /health/live — liveness probe (just returns 200)
	r.Get("/health", healthHandler(db, redisClient))
	r.Get("/health/live", liveHandler)

	// --- Public Routes ---
	// No authentication required. Stricter rate limit to prevent brute force.
	r.Route("/v1/auth", func(r chi.Router) {
		r.Use(middleware.RateLimiter(5, 10)) // 5 req/sec per IP for auth
		r.Post("/register", authHandler.RegisterBusiness)
		r.Post("/login", authHandler.Login)
		r.Post("/accept-invitation", authHandler.AcceptInvitation)
		r.Post("/forgot-password", authHandler.ForgotPassword)
		r.Post("/reset-password", authHandler.ResetPassword)
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

	// --- Paystack Webhook Routes ---
	// Public endpoint for Paystack transfer status updates
	paystackWebhookHandler := handler.NewPaystackWebhookHandler(cfg.PaystackSecretKey, transferRepo, walletSvc)
	r.Route("/paystack/webhooks", func(r chi.Router) {
		r.Post("/", paystackWebhookHandler.HandleWebhook)
	})

	// --- Protected API v1 Group ---
	// All routes within this group require a valid JWT.
	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret))

		// --- Dashboard (all authenticated roles) ---
		r.Get("/dashboard", dashboardHandler.GetSummary)

		// --- Notifications (all authenticated roles) ---
		r.Route("/notifications", func(r chi.Router) {
			r.Get("/", notifHandler.ListNotifications)
			r.Get("/unread-count", notifHandler.GetUnreadCount)
			r.Patch("/{id}/read", notifHandler.MarkAsRead)
			r.Patch("/read-all", notifHandler.MarkAllAsRead)
		})

		// --- Bank Account Verification ---
		r.Get("/verify/bank-account", verificationHandler.HandleVerifyBankAccount)

		// --- Employee Self-Service ---
		r.Route("/me", func(r chi.Router) {
			r.Get("/profile", selfServiceHandler.GetProfile)
			r.Patch("/bank-details", selfServiceHandler.UpdateBankDetails)
			r.Get("/payslips", selfServiceHandler.GetPayslips)
		})

		// --- Loans (Admin/Operator) ---
		r.Route("/loans", func(r chi.Router) {
			r.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleOperator))
			r.Post("/", loanHandler.CreateLoan)
			r.Get("/", loanHandler.ListLoans)
			r.Patch("/{id}/cancel", loanHandler.CancelLoan)
		})

		// --- Admin & Operator Routes ---
		// These routes are for day-to-day operations.
		r.Group(func(r chi.Router) {
			r.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleOperator))

			// Employee Management
			r.Route("/employees", func(r chi.Router) {
				r.Post("/", employeeHandler.CreateEmployee)
				r.Post("/import", employeeHandler.ImportEmployees)
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

			// Reports and Payslips (Admin + Operator)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RoleMiddleware(domain.RoleAdmin, domain.RoleOperator))
				r.Get("/{runID}/reports/paye", reportHandler.HandlePAYEReport)
				r.Get("/{runID}/reports/pension", reportHandler.HandlePensionSchedule)
				r.Get("/{runID}/reports/nhf", reportHandler.HandleNHFSchedule)
				r.Get("/{runID}/reports/bank-schedule", reportHandler.HandleBankSchedule)
				r.Get("/{runID}/reports/summary", reportHandler.HandlePayrollSummary)
				r.Get("/{runID}/payslips", reportHandler.HandleAllPayslips)
				r.Get("/{runID}/payslips/{employeeID}", reportHandler.HandlePayslip)
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

			// User Invitation (Admin only)
			r.Post("/auth/invite", authHandler.InviteUser)

			// Audit Logs
			r.Get("/audit-logs", auditHandler.ListAuditLogs)

			// Business Settings Management
			r.Route("/business", func(r chi.Router) {
				r.Get("/settings", businessHandler.GetSettings)
				r.Patch("/settings", businessHandler.UpdateSettings)
			})
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

		// --- Transfer Routes (Provider-Agnostic) ---
		r.Route("/transfers", func(r chi.Router) {
			r.Post("/", newTransferHandler.HandleSingleTransfer)
			r.Post("/batch", newTransferHandler.HandleBatchTransfer)
			r.Get("/", newTransferHandler.HandleListTransfers)
			r.Get("/{id}", newTransferHandler.HandleGetTransfer)
			r.Post("/{id}/retry", newTransferHandler.HandleRetryTransfer)
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

	// --- Billing Routes (inside v1, all authenticated) ---
	// Mounted at the end to keep tenant routes clean
	r.Route("/v1/billing", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		r.Get("/plans", billingHandler.GetPlans)
		r.Get("/subscription", billingHandler.GetSubscription)
		r.Post("/subscribe", billingHandler.Subscribe)
		r.Post("/cancel", billingHandler.CancelSubscription)
		r.Get("/invoices", billingHandler.ListInvoices)
	})

	// --- Platform Admin Routes (super admin only) ---
	r.Route("/platform", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		r.Use(middleware.SuperAdminMiddleware)
		r.Get("/stats", platformHandler.GetStats)
		r.Get("/organizations", platformHandler.ListOrganizations)
		r.Post("/organizations/{id}/suspend", platformHandler.SuspendOrganization)
		r.Post("/organizations/{id}/activate", platformHandler.ActivateOrganization)
	})

	return r
}
