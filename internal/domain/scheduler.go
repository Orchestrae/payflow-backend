package domain

// Scheduler defines the interface for scheduling payroll runs
type Scheduler interface {
	Start()
	Stop()
	SchedulePayout(run PayrollRun) error
	SetPayrollService(payrollSvc PayrollService)
}
