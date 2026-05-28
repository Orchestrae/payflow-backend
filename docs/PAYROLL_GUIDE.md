# Payroll Guide

Comprehensive documentation for PayFlow's payroll system, covering the state machine, business configuration, calculation logic, transfer integration, and background scheduling.

---

## Table of Contents

- [Payroll State Machine](#payroll-state-machine)
- [Business Configuration](#business-configuration)
- [Workflow Variations](#workflow-variations)
- [API Endpoints](#api-endpoints)
- [Roles and Permissions](#roles-and-permissions)
- [Calculation Logic](#calculation-logic)
- [Transfer Integration](#transfer-integration)
- [Background Scheduling](#background-scheduling)
- [Cross-References](#cross-references)

---

## Payroll State Machine

A payroll run progresses through the following states:

```
draft --> pending_approval --> approved --> processing --> completed
              |  (reject)                    |  (fail)
              v                              v
          rejected                        failed
```

| State              | Description                                      |
|--------------------|--------------------------------------------------|
| `draft`            | Initial state. Payroll has been created but not yet submitted. |
| `pending_approval` | Submitted and awaiting approval from an authorized user. |
| `approved`         | Approved and ready to be scheduled or processed.  |
| `processing`       | Transfers are actively being executed.             |
| `completed`        | All transfers have been successfully completed.    |
| `rejected`         | An approver has rejected the payroll run.          |
| `failed`           | One or more transfers failed during processing.    |

---

## Business Configuration

Two boolean flags on the business entity control how payroll flows through the state machine.

### payroll_requires_approval

- **Type:** bool
- **Default:** true
- When `true`, submitting a payroll run moves it to `pending_approval`, and an explicit approve action is required.
- When `false`, submitting a payroll run automatically approves it, skipping `pending_approval`.

### payroll_auto_process

- **Type:** bool
- **Default:** false
- When `true`, the system begins processing immediately after approval, skipping the manual trigger step.
- When `false`, an approved payroll run waits for a scheduled job or a manual `process-now` call.

---

## Workflow Variations

### Standard Flow (default)

Both flags at their defaults (`requires_approval=true`, `auto_process=false`).

```
draft --> submit --> pending_approval --> approve --> approved --> scheduled --> processing --> completed
```

### Auto-Approval Flow

`payroll_requires_approval=false`, `payroll_auto_process=false`.

```
draft --> submit --> approved --> scheduled --> processing --> completed
```

### Auto-Process Flow

`payroll_requires_approval=true`, `payroll_auto_process=true`.

```
draft --> submit --> pending_approval --> approve --> processing --> completed
```

### Instant Flow

Both flags enabled (`requires_approval=false`, `auto_process=true`).

```
draft --> submit --> processing --> completed
```

---

## API Endpoints

All endpoints are scoped under `/v1/payroll-runs` and require JWT authentication.

| Method | Path                                  | Description                                |
|--------|---------------------------------------|--------------------------------------------|
| POST   | `/v1/payroll-runs`                    | Create a payroll run (calculates all employees) |
| GET    | `/v1/payroll-runs`                    | List payroll runs                          |
| GET    | `/v1/payroll-runs/{runID}`            | Get payroll run details                    |
| POST   | `/v1/payroll-runs/{runID}/submit`     | Submit for approval                        |
| POST   | `/v1/payroll-runs/{runID}/approve`    | Approve payroll                            |
| POST   | `/v1/payroll-runs/{runID}/reject`     | Reject payroll                             |
| POST   | `/v1/payroll-runs/{runID}/process-now`| Instant processing (bypasses scheduler)    |

### Create Payroll Run

`POST /v1/payroll-runs`

Creates a new payroll run in `draft` state. The system iterates over all active employees in the business, computes gross pay, applies deduction rules, and produces a net pay figure for each employee.

### Submit Payroll Run

`POST /v1/payroll-runs/{runID}/submit`

Transitions the run from `draft` to either `pending_approval` or `approved`, depending on the `payroll_requires_approval` setting. If `payroll_auto_process` is also enabled, processing begins immediately.

### Approve / Reject Payroll Run

`POST /v1/payroll-runs/{runID}/approve`
`POST /v1/payroll-runs/{runID}/reject`

Approve moves the run from `pending_approval` to `approved` (or directly to `processing` if `payroll_auto_process` is true). Reject moves it to `rejected`.

### Process Now

`POST /v1/payroll-runs/{runID}/process-now`

Bypasses the scheduler and immediately begins transfer execution. The run must be in `approved` state.

---

## Roles and Permissions

| Action                  | Admin | Operator | Approver | Viewer |
|-------------------------|:-----:|:--------:|:--------:|:------:|
| Create payroll run      | Yes   | Yes      | No       | No     |
| Submit payroll run      | Yes   | Yes      | No       | No     |
| Approve payroll run     | Yes   | No       | Yes      | No     |
| Reject payroll run      | Yes   | No       | Yes      | No     |
| Process payroll run     | Yes   | Yes      | No       | No     |
| View payroll runs       | Yes   | Yes      | Yes      | Yes    |

All authenticated users with a valid business context can view payroll runs.

---

## Calculation Logic

### Gross Pay

Gross pay is the sum of all earning components defined on the employee's cadre.

```
gross_pay = sum(cadre.earning_components)
```

### Deductions

Deduction rules are applied to compute total deductions. Each rule specifies:

- **Type:** `percentage` or `flat`
- **Calculation basis:** `gross` (applied to the full gross pay) or a specific earning component

For percentage-based rules:

```
deduction_amount = (rule.percentage / 100) * calculation_basis_amount
```

For flat rules:

```
deduction_amount = rule.flat_amount
```

### Net Pay

```
net_pay = gross_pay - total_deductions
```

### Currency

All monetary values are stored as `int64` in the smallest currency unit (kobo for NGN).

| Kobo Value | Naira Equivalent |
|------------|------------------|
| 500000     | NGN 5,000        |
| 100        | NGN 1            |
| 50         | NGN 0.50         |

---

## Transfer Integration

### Provider Manager

Payroll processing creates bulk transfer requests routed through the provider manager:

- **Korapay** (primary): Native bulk transfer API
- **VFD Bank** (fallback): Used when Korapay is unavailable

### Worker Pool

Transfer execution uses a pool of **5 concurrent workers**. Each worker picks up transfer tasks and processes them in parallel.

### Korapay Bulk Transfers

Korapay's native bulk transfer endpoint accepts **2 to 50 transfers per batch**. The system automatically batches employee transfers to stay within these limits.

### Failure Handling

If a transfer fails, the individual employee's transfer is marked as failed. The overall payroll run transitions to `failed` if any transfers do not complete successfully.

---

## Background Scheduling

PayFlow supports two scheduling backends, selected automatically based on infrastructure availability.

### Asynq (Redis-backed) -- Primary

- **Requirement:** Redis connection available
- **Persistence:** Jobs survive server restarts
- **Retries:** 3 attempts per task
- **Timeout:** 10 minutes per task
- **Recommended for:** Production deployments

### gocron (In-Memory) -- Fallback

- **Requirement:** None (runs in-process)
- **Persistence:** None. All scheduled jobs are lost on server restart.
- **Recommended for:** Development and environments without Redis

The system checks for Redis availability at startup and selects the appropriate backend automatically.

---

## Cross-References

- [Architecture](./ARCHITECTURE.md)
- [API Reference](./API_REFERENCE.md)
- [Wallet Guide](./WALLET_GUIDE.md)
