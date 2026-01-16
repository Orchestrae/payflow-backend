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

// SetPayrollService sets the payroll service after scheduler creation
// This is needed to resolve circular dependency during initialization
func (c *cronScheduler) SetPayrollService(payrollSvc domain.PayrollService) {
	c.payrollSvc = payrollSvc
	log.Info().Msg("Payroll service set on scheduler")
}

func (c *cronScheduler) SchedulePayout(run domain.PayrollRun) error {
	jobID := fmt.Sprintf("payout-run-%d", run.ID)

	// Schedule the job to run at the specified time
	// If ScheduledFor is in the past, run immediately; otherwise schedule for that time
	scheduledTime := run.ScheduledFor
	now := time.Now()
	
	var job *gocron.Job
	var err error
	
	if scheduledTime.Before(now) || scheduledTime.Equal(now) {
		// Schedule to run immediately (next second)
		job, err = c.scheduler.Every(1).Second().LimitRunsTo(1).Do(
			func(runID uint) {
				log.Info().Uint("runID", runID).Msg("Executing scheduled payout job (immediate)")
				if err := c.payrollSvc.ProcessApprovedPayroll(context.Background(), runID); err != nil {
					log.Error().Err(err).Uint("runID", runID).Msg("Failed to process approved payroll")
				}
			},
			run.ID,
		)
	} else {
		// Schedule for specific time
		job, err = c.scheduler.Every(1).Day().At(scheduledTime.Format("15:04")).LimitRunsTo(1).Do(
			func(runID uint) {
				log.Info().Uint("runID", runID).Msg("Executing scheduled payout job")
				if err := c.payrollSvc.ProcessApprovedPayroll(context.Background(), runID); err != nil {
					log.Error().Err(err).Uint("runID", runID).Msg("Failed to process approved payroll")
				}
			},
			run.ID,
		)
	}
	
	if err != nil {
		return fmt.Errorf("failed to schedule payout job for run %d: %w", run.ID, err)
	}
	
	log.Info().
		Str("job_id", jobID).
		Time("scheduled_for", scheduledTime).
		Str("next_run", job.NextRun().Format(time.RFC3339)).
		Msg("Successfully scheduled payroll payout job")
	
	return nil
}
