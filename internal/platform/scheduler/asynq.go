package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"payflow/internal/domain"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

const (
	TypePayrollProcess = "payroll:process"
)

type payrollPayload struct {
	RunID uint `json:"run_id"`
}

// EmailTaskHandler can handle email tasks from the queue.
type EmailTaskHandler interface {
	HandleEmailTask(ctx context.Context, t *asynq.Task) error
}

// asynqScheduler implements domain.Scheduler using Asynq (Redis-backed).
type asynqScheduler struct {
	client       *asynq.Client
	server       *asynq.Server
	payrollSvc   domain.PayrollService
	payoutSvc    domain.PayoutService
	emailHandler EmailTaskHandler
}

// NewAsynqScheduler creates a new Asynq-based scheduler.
func NewAsynqScheduler(redisURL string, payrollSvc domain.PayrollService, payoutSvc domain.PayoutService) (domain.Scheduler, error) {
	redisOpt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL for asynq: %w", err)
	}

	client := asynq.NewClient(redisOpt)
	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"payroll": 5,
			"email":   3,
			"default": 2,
		},
		RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
			return time.Duration(n*n) * time.Minute // Exponential backoff
		},
	})

	return &asynqScheduler{
		client:     client,
		server:     server,
		payrollSvc: payrollSvc,
		payoutSvc:  payoutSvc,
	}, nil
}

// SetEmailHandler registers the email task handler.
func (s *asynqScheduler) SetEmailHandler(h EmailTaskHandler) {
	s.emailHandler = h
}

func (s *asynqScheduler) Start() {
	log.Info().Msg("Starting Asynq background job processor")
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypePayrollProcess, s.handlePayrollProcess)
	if s.emailHandler != nil {
		mux.HandleFunc("email:send", s.emailHandler.HandleEmailTask)
		log.Info().Msg("Email task handler registered with Asynq")
	}

	go func() {
		if err := s.server.Start(mux); err != nil {
			log.Error().Err(err).Msg("Asynq server failed")
		}
	}()
}

func (s *asynqScheduler) Stop() {
	log.Info().Msg("Stopping Asynq background job processor")
	s.server.Shutdown()
	s.client.Close()
}

func (s *asynqScheduler) SetPayrollService(payrollSvc domain.PayrollService) {
	s.payrollSvc = payrollSvc
	log.Info().Msg("Payroll service set on Asynq scheduler")
}

func (s *asynqScheduler) SchedulePayout(run domain.PayrollRun) error {
	payload, err := json.Marshal(payrollPayload{RunID: run.ID})
	if err != nil {
		return fmt.Errorf("failed to marshal payroll payload: %w", err)
	}

	opts := []asynq.Option{
		asynq.MaxRetry(3),
		asynq.Queue("payroll"),
		asynq.Timeout(10 * time.Minute),
	}

	// Schedule for the future or process immediately
	now := time.Now()
	if run.ScheduledFor.After(now) {
		opts = append(opts, asynq.ProcessAt(run.ScheduledFor))
	}

	task := asynq.NewTask(TypePayrollProcess, payload)
	info, err := s.client.Enqueue(task, opts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue payroll job for run %d: %w", run.ID, err)
	}

	log.Info().
		Uint("run_id", run.ID).
		Str("task_id", info.ID).
		Time("scheduled_for", run.ScheduledFor).
		Msg("Payroll job enqueued")

	return nil
}

func (s *asynqScheduler) handlePayrollProcess(ctx context.Context, task *asynq.Task) error {
	var p payrollPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal payroll payload: %w", err)
	}

	log.Info().Uint("run_id", p.RunID).Msg("Processing payroll job from queue")

	if err := s.payrollSvc.ProcessApprovedPayroll(ctx, p.RunID); err != nil {
		log.Error().Err(err).Uint("run_id", p.RunID).Msg("Failed to process payroll")
		return err // Asynq will retry based on MaxRetry
	}

	log.Info().Uint("run_id", p.RunID).Msg("Payroll job completed successfully")
	return nil
}
