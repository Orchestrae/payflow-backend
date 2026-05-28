// cmd/server/main.go
package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"payflow/internal/api"
	"payflow/internal/config"
	"payflow/internal/domain"
	"payflow/internal/platform/cache"
	"payflow/internal/platform/database"
	"payflow/internal/platform/email"
	"payflow/internal/platform/korapay"
	"payflow/internal/platform/paystack"
	"payflow/internal/platform/scheduler"
	"payflow/internal/platform/vfd"
	"payflow/internal/repository/postgres"
	"payflow/internal/service"
	"payflow/internal/service/provider"
	vfd2 "payflow/internal/service/vfd"

	"gorm.io/gorm"
)

func main() {
	// --- Phase 1: Configuration & Logger ---
	cfg, err := config.Load()
	if err != nil {
		// Use standard logger here as zerolog is not yet configured
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	if cfg.JWTSecret == "" {
		log.Fatal().Msg("JWT_SECRET is required — set it in environment or .env file")
	}

	if cfg.LogPretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if lvl, err := zerolog.ParseLevel(cfg.LogLevel); err == nil {
		zerolog.SetGlobalLevel(lvl)
	}
	log.Info().Msg("PayFlow server starting up...")

	// Log database config status (not the URL) for deployment debugging
	if cfg.DatabaseURL == "" {
		log.Warn().Msg("DATABASE_URL/DB_URL not set - ensure Postgres is linked in Railway")
	} else {
		log.Info().Msg("Database URL configured")
	}

	// --- Redis Initialization (optional — graceful degradation) ---
	var redisClient *cache.RedisClient
	if cfg.RedisURL != "" {
		redisClient, err = cache.NewRedisClient(cfg.RedisURL)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to connect to Redis — running without cache/queue")
		} else {
			log.Info().Msg("Redis connected")
			defer redisClient.Close()
		}
	} else {
		log.Warn().Msg("REDIS_URL not set — running without cache and persistent job queue")
	}

	// --- Phase 2: Platform & Repository Initialization ---
	// Use auto-migration when enabled (local dev) or auto-detected for Railway/PaaS
	var db *gorm.DB
	if cfg.EnableAutoMigration {
		log.Info().Msg("Auto-migration enabled (Railway/PaaS or ENABLE_AUTO_MIGRATION=true)")
		db, err = database.InitializeDatabaseWithAutoMigration(cfg.DatabaseURL)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize database with auto-migration")
		}
		log.Info().Msg("Database initialized with auto-migration (development mode)")
	} else {
		db, err = database.InitializeDatabase(cfg.DatabaseURL)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to initialize database: %v", err)
		}
		log.Info().Msg("Database initialized (traditional migrations only - production mode)")
		log.Info().Msg("ℹ️  Run migrations manually: make migrate-up (or use golang-migrate CLI)")
	}

	// Repositories
	txer := postgres.NewTransactioner(db)
	userRepo := postgres.NewUserRepository(db)
	businessRepo := postgres.NewBusinessRepository(db)
	employeeRepo := postgres.NewEmployeeRepository(db)
	cadreRepo := postgres.NewCadreRepository(db)
	payrollRepo := postgres.NewPayrollRepository(db)
	deductionRuleRepo := postgres.NewDeductionRuleRepository(db)
	webhookRepo := postgres.NewVFDWebhookNotificationRepository(db)
	transferRepo := postgres.NewVFDTransferRepository(db)
	walletRepo := postgres.NewWalletRepository(db)
	walletTxRepo := postgres.NewWalletTransactionRepository(db)

	// --- Phase 3: External Service & Core Service Initialization ---
	// External Platform Clients
	koraClient := korapay.NewClient(cfg.KoraPayAPIKey, cfg.KoraPayBaseURL)
	payoutSvc := korapay.NewPayoutService(koraClient)
	// Email service (configurable SMTP — MailHog for dev, Brevo/SendGrid for production)
	notificationSvc := email.NewEmailService(cfg)
	vfdClient := vfd.NewClient(cfg.VFDBaseURL, cfg.VFDConsumerKey, cfg.VFDConsumerSecret)
	vfdSvc := vfd.NewVFDService(vfdClient)
	log.Info().Msg("External platform services initialized (Kora, VFD, Notifications)")

	// Cache service (optional — nil-safe, degrades gracefully)
	cacheSvc := cache.NewCacheService(redisClient)

	// Core Services
	authSvc := service.NewAuthService(userRepo, businessRepo, txer, cfg.JWTSecret, cfg.JWTExpirationDuration, vfdSvc,
		service.WithNotificationService(notificationSvc, cfg.AppURL))
	employeeSvc := service.NewEmployeeService(employeeRepo, cadreRepo)
	cadreSvc := service.NewCadreService(cadreRepo, cacheSvc)
	deductionRuleSvc := service.NewDeductionRuleService(deductionRuleRepo)
	webhookSvc := vfd2.NewVFDWebhookService(webhookRepo, businessRepo, vfdSvc, txer)
	transferSvc := vfd2.NewVFDTransferService(transferRepo, vfdSvc, txer)

	// Initialize transfer providers (only enabled ones with valid credentials)
	enabledProviders := parseEnabledProviders(cfg.EnabledProviders)
	providers := make(map[domain.ProviderName]provider.TransferProvider)

	if enabledProviders["korapay"] && cfg.KoraPayAPIKey != "" {
		providers[domain.ProviderKorapay] = korapay.NewTransferProvider(koraClient)
		log.Info().Msg("Korapay transfer provider enabled")
	}
	if enabledProviders["vfd"] && cfg.VFDConsumerKey != "" {
		providers[domain.ProviderVFD] = provider.NewVFDProvider(vfdSvc)
		log.Info().Msg("VFD transfer provider enabled")
	}
	if enabledProviders["paystack"] && cfg.PaystackSecretKey != "" {
		paystackClient := paystack.NewClient(cfg.PaystackSecretKey, cfg.PaystackBaseURL)
		providers[domain.ProviderPaystack] = paystack.NewTransferProvider(paystackClient)
		log.Info().Msg("Paystack transfer provider enabled")
	}

	if len(providers) == 0 {
		log.Warn().Msg("No transfer providers enabled — transfers will fail")
	}

	// Create provider manager
	providerManager, err := provider.NewTransferProviderManager(
		cfg.TransferDefaultProvider,
		cfg.TransferProviderFallbackOrder,
		providers,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create transfer provider manager")
	}
	log.Info().Msgf("Transfer provider manager initialized with default provider: %s", cfg.TransferDefaultProvider)

	// Initialize virtual account provider (KoraPay)
	korapayVirtualAccountProvider := korapay.NewVirtualAccountProvider(koraClient)
	log.Info().Msg("KoraPay virtual account provider initialized")

	// Initialize account holder provider (KoraPay)
	korapayAccountHolderProvider := korapay.NewAccountHolderProvider(koraClient)
	log.Info().Msg("KoraPay account holder provider initialized")

	// New transfer repository and service (provider-agnostic)
	newTransferRepo := postgres.NewTransferRepository(db)
	transferConfig := service.TransferConfig{
		MinAmount: cfg.TransferMinAmount,
		MaxAmount: cfg.TransferMaxAmount,
	}
	transferSvcNew := service.NewTransferService(providerManager, newTransferRepo, userRepo, txer, transferConfig)
	log.Info().Msgf("Transfer service initialized with limits: min=%d, max=%d", cfg.TransferMinAmount, cfg.TransferMaxAmount)

	// Business, Dashboard, Audit services
	businessSvc := service.NewBusinessService(businessRepo)
	dashboardSvc := service.NewDashboardService(employeeRepo, payrollRepo, walletRepo)
	auditRepo := postgres.NewAuditRepository(db)
	auditSvc := service.NewAuditService(auditRepo)
	log.Info().Msg("Business service initialized")

	// Initialize wallet service with virtual account provider
	walletSvc := service.NewWalletService(walletRepo, walletTxRepo, korapayVirtualAccountProvider)
	log.Info().Msg("Wallet service initialized")

	// Initialize account holder service with account holder provider
	accountHolderSvc := service.NewAccountHolderService(korapayAccountHolderProvider)
	log.Info().Msg("Account holder service initialized")

	// Wire wallet service into transfer service for balance checking
	if transferSvcWithWallet, ok := transferSvcNew.(interface{ SetWalletService(service.WalletService) }); ok {
		transferSvcWithWallet.SetWalletService(walletSvc)
		log.Info().Msg("Wallet service wired into transfer service for balance checking")
	} else {
		log.Warn().Msg("Transfer service does not support SetWalletService - balance checking disabled")
	}


	// --- Phase 4: Resolving the Scheduler <-> Service Circular Dependency ---
	// Choose scheduler: Asynq (Redis-backed, persistent) or gocron (in-memory fallback)
	var payoutScheduler domain.Scheduler
	if redisClient != nil {
		payoutScheduler, err = scheduler.NewAsynqScheduler(cfg.RedisURL, nil, payoutSvc)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create Asynq scheduler")
		}
		log.Info().Msg("Using Asynq scheduler (Redis-backed, persistent)")
	} else {
		payoutScheduler, err = scheduler.NewCronScheduler(nil, payoutSvc)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create scheduler")
		}
		log.Warn().Msg("Using in-memory scheduler (gocron) — jobs lost on restart")
	}

	// 2. Create the PayrollService, injecting the scheduler and other dependencies.
	payrollSvc := service.NewPayrollService(
		payrollRepo,
		employeeRepo,
		cadreRepo,
		businessRepo,
		newTransferRepo,
		txer,
		payoutSvc,
		notificationSvc,
		userRepo,
		payoutScheduler,
		transferSvcNew,
	)

	// 3. Update scheduler with the PayrollService to resolve circular dependency
	// Cast to domain.PayrollService (the service implements both interfaces)
	if domainPayrollSvc, ok := payrollSvc.(domain.PayrollService); ok {
		payoutScheduler.SetPayrollService(domainPayrollSvc)
	} else {
		log.Fatal().Msg("PayrollService does not implement domain.PayrollService")
	}

	// --- Phase 5: API Router & Server Startup ---
	router := api.NewRouter(cfg, db, redisClient, authSvc, employeeSvc, cadreSvc, deductionRuleSvc, payrollSvc, webhookSvc, transferSvc, transferSvcNew, walletSvc, businessSvc, dashboardSvc, auditSvc, accountHolderSvc, koraClient, newTransferRepo, businessRepo)
	log.Info().Msg("API router initialized")

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	// Start the background job scheduler
	payoutScheduler.Start()
	log.Info().Msg("Background scheduler started")

	// Start server in a goroutine so it doesn't block.
	go func() {
		log.Info().Msgf("Server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server startup failed")
		}
	}()

	// --- Phase 6: Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until a signal is received.

	log.Warn().Msg("Shutdown signal received. Starting graceful shutdown...")

	// Create a context with a timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop the scheduler first, allowing running jobs to finish.
	payoutScheduler.Stop()
	log.Info().Msg("Scheduler stopped")

	// Now, shut down the HTTP server.
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exiting gracefully")
}

// parseEnabledProviders parses the ENABLED_PROVIDERS config into a set.
func parseEnabledProviders(enabledProviders string) map[string]bool {
	result := make(map[string]bool)
	for _, name := range strings.Split(enabledProviders, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			result[name] = true
		}
	}
	return result
}
