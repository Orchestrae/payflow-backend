// internal/platform/scheduler/cron.go
package scheduler

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"
)

type cronScheduler struct {
	scheduler  *gocron.Scheduler
	payrollSvc domain.PayrollService
	payoutSvc  domain.PayoutService
}

func NewCronScheduler(payrollSvc domain.PayrollService, payoutSvc domain.PayoutService) (domain.Scheduler, error) {
	s := gocron.NewScheduler(time.Local)
	return &cronScheduler{scheduler: s, payrollSvc: payrollSvc, payoutSvc: payoutSvc}, nil
}

func (c *cronScheduler) Start() {
	log.Info().Msg("Starting background job scheduler")
	c.scheduler.StartAsync()
}

func (c *cronScheduler) Stop() {
	log.Info().Msg("Stopping background job scheduler")
	c.scheduler.Stop()
}

func (c *cronScheduler) SchedulePayout(run domain.PayrollRun) error {
	jobID := fmt.Sprintf("payout-run-%d", run.ID)

	_, err := c.scheduler.Every(1).Day().At(run.ScheduledFor.Format("15:04")).Do(
		func(runID uint, businessID uint) {
			log.Info().Uint("runID", runID).Msg("Executing scheduled payout job")

			// 1. Fetch the latest run details to prevent stale data
			freshRun, err := c.payrollSvc.GetPayrollRunForDisbursement(context.Background(), runID)
			if err != nil {
				log.Error().Err(err).Uint("runID", runID).Msg("Failed to fetch run details")
				return
			}

			// 2. Mark as processing
			if err := c.payrollSvc.UpdateRunStatus(context.Background(), runID, domain.StatusProcessing); err != nil {
				log.Error().Err(err).Uint("runID", runID).Msg("Failed to update run status")
				return
			}

			// 3. Call the payment gateway
			ref, err := c.payoutSvc.DisburseBulkPayment(context.Background(), *freshRun)
			if err != nil {
				log.Error().Err(err).Uint("runID", runID).Msg("Disbursement failed")
				if err := c.payrollSvc.MarkRunAsFailed(context.Background(), runID, err.Error()); err != nil {
					log.Error().Err(err).Uint("runID", runID).Msg("Failed to mark run as failed")
				}
				return
			}

			log.Info().Str("reference", ref).Uint("runID", runID).Msg("Disbursement successful")
			if err := c.payrollSvc.MarkRunAsCompleted(context.Background(), runID, ref); err != nil {
				log.Error().Err(err).Uint("runID", runID).Msg("Failed to mark run as completed")
			}
		},
		run.ID,
		run.BusinessID,
	)
	if err != nil {
		return fmt.Errorf("failed to schedule payout job for run %d: %w", run.ID, err)
	}
	log.Info().Time("scheduledFor", run.ScheduledFor).Msgf("Successfully scheduled job '%s'", jobID)
	return nil
}
