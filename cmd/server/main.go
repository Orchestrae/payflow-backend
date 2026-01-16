// cmd/server/main.go
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"payflow/internal/api"
	"payflow/internal/config"
	"payflow/internal/domain"
	"payflow/internal/platform/database"
	"payflow/internal/platform/korapay"
	"payflow/internal/platform/scheduler"
	"payflow/internal/platform/sendgrid" // Using Mailhog/Sendgrid as example
	"payflow/internal/platform/vfd"
	"payflow/internal/repository/postgres"
	"payflow/internal/service"
	"payflow/internal/service/provider"
	vfd2 "payflow/internal/service/vfd"
)

func main() {
	// --- Phase 1: Configuration & Logger ---
	cfg, err := config.Load()
	if err != nil {
		// Use standard logger here as zerolog is not yet configured
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().Msgf("JWT Secret: %s", cfg.JWTSecret)

	if cfg.LogPretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if lvl, err := zerolog.ParseLevel(cfg.LogLevel); err == nil {
		zerolog.SetGlobalLevel(lvl)
	}
	log.Info().Msg("PayFlow server starting up...")

	// --- Phase 2: Platform & Repository Initialization ---
	db, err := database.InitializeDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	log.Info().Msg("Database initialized and automigration completed")

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

	// --- Phase 3: External Service & Core Service Initialization ---
	// External Platform Clients
	koraClient := korapay.NewClient(cfg.KoraPayAPIKey, cfg.KoraPayBaseURL)
	payoutSvc := korapay.NewPayoutService(koraClient)
	// For local dev, use a mock/Mailhog service. For prod, use a real one.
	notificationSvc := sendgrid.NewMailhogService()
	vfdClient := vfd.NewClient(cfg.VFDBaseURL, cfg.VFDConsumerKey, cfg.VFDConsumerSecret)
	vfdSvc := vfd.NewVFDService(vfdClient)
	log.Info().Msg("External platform services initialized (Kora, VFD, Notifications)")

	// Core Services
	authSvc := service.NewAuthService(userRepo, businessRepo, txer, cfg.JWTSecret, cfg.JWTExpirationDuration, vfdSvc)
	employeeSvc := service.NewEmployeeService(employeeRepo, cadreRepo)
	cadreSvc := service.NewCadreService(cadreRepo)
	deductionRuleSvc := service.NewDeductionRuleService(deductionRuleRepo)
	webhookSvc := vfd2.NewVFDWebhookService(webhookRepo, businessRepo, vfdSvc, txer)
	transferSvc := vfd2.NewVFDTransferService(transferRepo, vfdSvc, txer)

	// Initialize transfer providers with new interface
	providers := make(map[domain.ProviderName]provider.TransferProvider)
	vfdProvider := provider.NewVFDProvider(vfdSvc)
	korapayProvider := korapay.NewTransferProvider(koraClient)
	providers[domain.ProviderVFD] = vfdProvider
	providers[domain.ProviderKorapay] = korapayProvider

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

	// New transfer repository and service (provider-agnostic)
	newTransferRepo := postgres.NewTransferRepository(db)
	transferConfig := service.TransferConfig{
		MinAmount: cfg.TransferMinAmount,
		MaxAmount: cfg.TransferMaxAmount,
	}
	transferSvcNew := service.NewTransferService(providerManager, newTransferRepo, userRepo, txer, transferConfig)
	log.Info().Msgf("Transfer service initialized with limits: min=%d, max=%d", cfg.TransferMinAmount, cfg.TransferMaxAmount)

	// Keep legacy bulk transfer service for backward compatibility (deprecated)
	bulkTransferSvc := service.NewBulkTransferService(providerManager, transferRepo, txer)

	// --- Phase 4: Resolving the Scheduler <-> Service Circular Dependency ---
	// 1. Create a temporary scheduler (will be updated with payroll service)
	payoutScheduler, err := scheduler.NewCronScheduler(nil, payoutSvc)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create scheduler")
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
	router := api.NewRouter(cfg, authSvc, employeeSvc, cadreSvc, deductionRuleSvc, payrollSvc, webhookSvc, transferSvc, bulkTransferSvc, transferSvcNew)
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
