# PayFlow API Flow Guide

> Complete endpoint reference derived from `internal/api/router.go` and handler source code.
> All monetary amounts are in **kobo** (minor currency units) unless stated otherwise.

**Base URL (local):** `http://localhost:8080`
**Base URL (production):** `https://api.payflow.example.com`

**Authentication:** JWT Bearer token in the `Authorization` header.
```
Authorization: Bearer <token>
```

**Common Error Response:**
```json
{
  "error": "Human-readable error message",
  "details": { "field": "reason" }
}
```

---

## Table of Contents

1. [Health](#1-health)
2. [Auth](#2-auth)
3. [Employees](#3-employees)
4. [Cadres (Salary Structures)](#4-cadres)
5. [Deduction Rules](#5-deduction-rules)
6. [Payroll Runs](#6-payroll-runs)
7. [Payroll Reports](#7-payroll-reports)
8. [Transfers](#8-transfers)
9. [VFD Transfers (Legacy)](#9-vfd-transfers-legacy)
10. [Wallet](#10-wallet)
11. [Account Holders / KYC](#11-account-holders--kyc)
12. [Billing / Subscriptions](#12-billing--subscriptions)
13. [Loans](#13-loans)
14. [Leave Management](#14-leave-management)
15. [Notifications](#15-notifications)
16. [Verification](#16-verification)
17. [Self-Service (Employee Portal)](#17-self-service)
18. [Business Settings](#18-business-settings)
19. [Audit Logs](#19-audit-logs)
20. [Platform Admin (Super Admin)](#20-platform-admin)
21. [Webhooks (Inbound)](#21-webhooks-inbound)

---

## 1. Health

### GET /health
Readiness probe. Checks database and Redis connectivity.

**Auth required:** No

**Example response (200):**
```json
{
  "status": "healthy",
  "message": "Server is running. All systems operational.",
  "server": "ok",
  "database": { "status": "ok", "message": "connected" },
  "redis": { "status": "ok", "message": "connected" }
}
```

### GET /health/live
Liveness probe. Always returns 200 if the process is running.

**Auth required:** No

**Example response (200):**
```json
{ "status": "alive" }
```

---

## 2. Auth

All auth endpoints are rate-limited to 5 req/sec per IP.

### POST /v1/auth/register
Register a new business and admin user. Creates a VFD corporate account.

**Auth required:** No

**Request body:**
```json
{
  "business_name": "Acme Ltd",
  "email": "admin@acme.com",
  "password": "securePass123",
  "rc_number": "RC123456",
  "incorporation_date": "2020-01-15T00:00:00Z",
  "director_bvn": "12345678901"
}
```

**Response (201):**
```json
{
  "user": {
    "id": 1,
    "email": "admin@acme.com",
    "role": "admin",
    "business_id": 1
  },
  "corporate_account": {
    "account_number": "8012345678",
    "account_name": "Acme Ltd"
  }
}
```

### POST /v1/auth/login
Authenticate an admin/operator/approver user.

**Auth required:** No

**Request body:**
```json
{
  "email": "admin@acme.com",
  "password": "securePass123"
}
```

**Response (200):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "email": "admin@acme.com",
    "role": "admin",
    "business_id": 1
  }
}
```

### POST /v1/auth/employee/login
Authenticate an employee for the self-service portal.

**Auth required:** No

**Request body:**
```json
{
  "email": "john@acme.com",
  "password": "tempPass123"
}
```

**Response (200):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": { "id": 5, "email": "john@acme.com", "role": "employee", "business_id": 1 }
}
```

### POST /v1/auth/accept-invitation
Accept a role invitation and set a password.

**Auth required:** No

**Request body:**
```json
{
  "token": "invite-token-uuid",
  "password": "newSecurePass"
}
```

**Response (200):** Same shape as login response.

### POST /v1/auth/forgot-password
Request a password reset email. Always returns success (does not reveal whether email exists).

**Auth required:** No

**Request body:**
```json
{ "email": "admin@acme.com" }
```

**Response (200):**
```json
{ "message": "If this email is registered, a password reset link has been sent" }
```

### POST /v1/auth/reset-password
Reset password using a token from the email.

**Auth required:** No

**Request body:**
```json
{
  "token": "reset-token-uuid",
  "new_password": "newSecurePass"
}
```

**Response (200):**
```json
{ "message": "Password reset successfully" }
```

### POST /v1/auth/verify-email
Verify email address using token from verification email.

**Auth required:** No

**Request body:**
```json
{ "token": "verify-token-uuid" }
```

**Response (200):**
```json
{ "message": "Email verified successfully" }
```

### POST /v1/auth/resend-verification
Resend email verification link to the authenticated user.

**Auth required:** Yes (any role)

**Request body:** None

**Response (200):**
```json
{ "message": "Verification email sent" }
```

### POST /v1/auth/invite
Invite a new user (operator or approver) to the business.

**Auth required:** Yes (admin only)

**Request body:**
```json
{
  "email": "approver@acme.com",
  "role": "approver"
}
```

**Response (201):**
```json
{ "message": "Invitation sent successfully" }
```

### POST /v1/employees/{employeeID}/create-login
Create a login account for an existing employee (enables self-service portal).

**Auth required:** Yes (admin, operator)

**Request body:**
```json
{ "temp_password": "tempPass123" }
```

**Response (201):** User object with id, email, role, business_id.

---

## 3. Employees

All employee endpoints require auth with role **admin** or **operator**.

### POST /v1/employees
Create a new employee.

**Request body:**
```json
{
  "cadre_id": 1,
  "full_name": "John Doe",
  "email": "john@acme.com",
  "bank_name": "GTBank",
  "bank_code": "058",
  "bank_account_number": "0123456789",
  "tin": "1234567890",
  "pension_rsa_pin": "PEN001234",
  "nhf_number": "NHF001234",
  "annual_rent_paid": 240000000
}
```

**Response (201):** Full employee object.

### GET /v1/employees
List all employees for the business.

**Response (200):** Array of employee objects.

### GET /v1/employees/{employeeID}
Get a single employee by ID.

**Response (200):** Employee object.

### PUT /v1/employees/{employeeID}
Update an employee. All fields are optional (partial update).

**Request body:**
```json
{
  "full_name": "John Updated",
  "cadre_id": 2,
  "bank_name": "Access Bank",
  "bank_code": "044",
  "bank_account_number": "9876543210",
  "is_active": true,
  "tin": "0987654321",
  "pension_rsa_pin": "PEN009999",
  "nhf_number": "NHF009999",
  "annual_rent_paid": 360000000
}
```

**Response (200):** Updated employee object.

### PATCH /v1/employees/{employeeID}/deactivate
Deactivate an employee (soft delete from payroll).

**Response (204):** No content.

### POST /v1/employees/import
Bulk import employees from a CSV file. Multipart form upload (max 5 MB).

**Form field:** `file` (CSV)

**Response (200):**
```json
{
  "created": 8,
  "failed": 2,
  "total": 10,
  "errors": ["row 3 (bad@email): cadre not found"]
}
```

### GET /v1/employees/import/template
Download a CSV template for employee import.

**Response (200):** CSV file download.

---

## 4. Cadres

Cadres define salary structures (earning components). Requires **admin** or **operator** role.

### POST /v1/cadres

**Request body:**
```json
{
  "name": "Senior Engineer",
  "earning_components": [
    { "name": "Basic Salary", "amount": 50000000 },
    { "name": "Housing Allowance", "amount": 20000000 },
    { "name": "Transport Allowance", "amount": 10000000 }
  ]
}
```

**Response (201):** Full cadre object with ID.

### GET /v1/cadres
List all cadres for the business.

**Response (200):** Array of cadre objects.

### GET /v1/cadres/{cadreID}
Get a single cadre.

**Response (200):** Cadre object.

### PUT /v1/cadres/{cadreID}
Update a cadre. Same request body as create.

**Response (200):** Updated cadre object.

### DELETE /v1/cadres/{cadreID}
Delete a cadre.

**Response (204):** No content.

---

## 5. Deduction Rules

Custom deduction rules for the business. Requires **admin** role.

### POST /v1/deduction-rules

**Request body:**
```json
{
  "name": "Health Insurance",
  "type": "percentage",
  "value": 2.5,
  "calculation_basis": "gross_pay"
}
```

`type`: `"percentage"` or `"flat"`
`calculation_basis`: `"gross_pay"` or `"basic_pay"`

**Response (201):** Deduction rule object.

### GET /v1/deduction-rules
List all custom deduction rules.

**Response (200):** Array of deduction rule objects.

### PUT /v1/deduction-rules/{ruleID}
Update a deduction rule. Same body as create.

**Response (200):** Updated rule object.

### DELETE /v1/deduction-rules/{ruleID}
Delete a deduction rule.

**Response (204):** No content.

---

## 6. Payroll Runs

### POST /v1/payroll-runs
Create a new payroll run. Calculates gross, statutory deductions (PAYE, pension, NHF, NSITF), custom deductions, loan deductions, and net pay for all active employees.

**Auth required:** Yes (admin, operator)

**Request body:**
```json
{
  "period": "2026-06",
  "adjustments": {
    "1": [
      { "item_name": "Performance Bonus", "amount": 5000000, "description": "Q2 bonus" },
      { "item_name": "Late Penalty", "amount": -200000, "component_type": "deduction" }
    ],
    "jane@acme.com": [
      { "item_name": "Overtime", "amount": 1500000 }
    ]
  }
}
```

- `period`: optional, defaults to current month. Format: `YYYY-MM` or `YYYY-MM-DD`.
- `adjustments`: optional. Keys can be employee ID (string) or email. `amount` positive = earnings, negative = deduction.

**Response (201):** Full payroll run object with entries.

### GET /v1/payroll-runs
List all payroll runs for the business.

**Auth required:** Yes (admin, operator, approver)

**Response (200):** Array of payroll run objects.

### GET /v1/payroll-runs/{runID}
Get detailed payroll run by ID.

**Auth required:** Yes (admin, operator, approver)

**Response (200):** Payroll run object with all entries.

### GET /v1/payroll-runs/{runID}/status
Poll payroll processing status.

**Auth required:** Yes (admin, operator, approver)

**Response (200):**
```json
{
  "id": 1,
  "status": "processing",
  "processing_job_id": "asynq-job-id",
  "processing_error": null,
  "processed_at": null
}
```

### POST /v1/payroll-runs/{runID}/submit
Submit payroll run for processing/approval.

**Auth required:** Yes (admin, operator)

**Response (202):**
```json
{
  "payroll_run": { ... },
  "job_id": "asynq-job-id",
  "status": "pending_approval",
  "message": "Payroll submitted for processing"
}
```

### POST /v1/payroll-runs/{runID}/approve
Approve a payroll run.

**Auth required:** Yes (admin, approver)

**Response (200):** Updated payroll run object.

### POST /v1/payroll-runs/{runID}/reject
Reject a payroll run with a reason.

**Auth required:** Yes (admin, approver)

**Request body:**
```json
{ "reason": "Incorrect bonus amounts for engineering team" }
```

**Response (200):** Updated payroll run object.

### POST /v1/payroll-runs/{runID}/process-now
Process a payroll run immediately (bypass scheduler).

**Auth required:** Yes (admin, operator)

**Response (200):** Updated payroll run object.

### PUT /v1/payroll-runs/{runID}/amend
Amend a draft payroll run with new adjustments.

**Auth required:** Yes (admin, operator)

**Request body:**
```json
{
  "adjustments": {
    "1": [{ "item_name": "Bonus", "amount": 3000000 }]
  }
}
```

**Response (200):** Updated payroll run object.

### POST /v1/payroll-runs/{runID}/reverse
Reverse a completed payroll run.

**Auth required:** Yes (admin, operator)

**Request body:**
```json
{ "reason": "Duplicate payroll run created in error" }
```

**Response (200):** Reversed payroll run object.

---

## 7. Payroll Reports

All report endpoints require **admin** or **operator** role.

### GET /v1/payroll-runs/{runID}/reports/paye
Download PAYE tax return as CSV.

**Response (200):** CSV file download.

### GET /v1/payroll-runs/{runID}/reports/pension
Download pension schedule as CSV.

**Response (200):** CSV file download.

### GET /v1/payroll-runs/{runID}/reports/nhf
Download NHF schedule as CSV.

**Response (200):** CSV file download.

### GET /v1/payroll-runs/{runID}/reports/bank-schedule
Download bank schedule as CSV.

**Response (200):** CSV file download.

### GET /v1/payroll-runs/{runID}/reports/summary
Download payroll summary as CSV.

**Response (200):** CSV file download.

### GET /v1/payroll-runs/{runID}/payslips
Download all payslips as a ZIP of PDFs.

**Response (200):** ZIP file download.

### GET /v1/payroll-runs/{runID}/payslips/{employeeID}
Download individual payslip as PDF.

**Response (200):** PDF file download.

---

## 8. Transfers

Provider-agnostic transfers (Korapay, Paystack, VFD with automatic fallback).

### POST /v1/transfers
Execute a single transfer. Velocity-limited: 10/hour, 50/day per business.

**Auth required:** Yes (any authenticated role)

**Request body:**
```json
{
  "amount": "500000",
  "bank_code": "058",
  "account_number": "0123456789",
  "account_name": "John Doe",
  "narration": "Salary payment",
  "reference": "custom-ref-001",
  "provider": "korapay"
}
```

- `amount`: string, in kobo
- `provider`: optional. `"korapay"`, `"paystack"`, or `"vfd"`. Falls back to default chain if omitted.
- `reference`: optional, auto-generated if not provided.

**Response (200):**
```json
{
  "success": true,
  "transfer_id": 42,
  "reference": "TRF-1-1717200000",
  "transaction_id": "kpy_txn_abc123",
  "status": "success",
  "message": "Transfer completed",
  "provider": "korapay",
  "currency": "NGN",
  "fee": 5000,
  "processing_time": "1.234s",
  "error": ""
}
```

### POST /v1/transfers/batch
Execute a batch of transfers (max 100). Velocity-limited: 5/hour, 20/day per business.

**Request body:**
```json
{
  "transfers": [
    { "amount": "500000", "bank_code": "058", "account_number": "0123456789", "account_name": "John Doe" },
    { "amount": "600000", "bank_code": "044", "account_number": "9876543210", "account_name": "Jane Smith" }
  ]
}
```

**Response (200):**
```json
{
  "total_transfers": 2,
  "successful_transfers": 2,
  "failed_transfers": 0,
  "transfers": [ ... ],
  "processing_time": "3.456s"
}
```

### GET /v1/transfers
List transfers for the business. Supports pagination.

**Query params:** `page` (default 1), `limit` (default 20, max 100)

**Response (200):**
```json
{
  "transfers": [ ... ],
  "total": 42,
  "page": 1,
  "limit": 20
}
```

### GET /v1/transfers/{id}
Get a specific transfer by ID.

**Response (200):** Transfer object.

### POST /v1/transfers/{id}/retry
Retry a failed transfer.

**Response (200):** Transfer result object (same shape as single transfer response).

---

## 9. VFD Transfers (Legacy)

Direct VFD Bank transfer operations. Authenticated routes.

### GET /v1/vfd/transfers/banks
Get list of supported banks.

### GET /v1/vfd/transfers/account-enquiry
Verify a bank account. Query params: `bank_code`, `account_number`.

### GET /v1/vfd/transfers/beneficiary-enquiry
Look up a beneficiary. Query params: `account_number`.

### POST /v1/vfd/transfers/initiate
Initiate a VFD transfer.

### GET /v1/vfd/transfers
List VFD transfers.

### GET /v1/vfd/transfers/{id}
Get a specific VFD transfer.

### GET /v1/vfd/transfers/from-account
List transfers by source account.

### GET /v1/vfd/transfers/to-account
List transfers by destination account.

---

## 10. Wallet

Virtual account and wallet management.

### POST /v1/wallets/virtual-account
Create a KoraPay virtual bank account for the business.

**Auth required:** Yes

**Request body:**
```json
{
  "account_name": "Acme Payroll",
  "customer_name": "Acme Ltd",
  "customer_email": "admin@acme.com",
  "bvn": "12345678901",
  "nin": "12345678901",
  "permanent": true
}
```

**Response (201):** Virtual account details with account number and bank name.

### GET /v1/wallets
Get wallet details for the business.

**Response (200):** Wallet object with virtual account info.

### GET /v1/wallets/balance
Get current wallet balance.

**Response (200):**
```json
{ "balance": 50000000, "currency": "NGN" }
```

### GET /v1/wallets/transactions
List wallet transactions with pagination.

**Query params:** `page` (default 1), `limit` (default 10)

**Response (200):**
```json
{
  "transactions": [ ... ],
  "total": 25,
  "page": 1,
  "limit": 10
}
```

### POST /v1/wallets/deposit
Initiate a Paystack deposit (card/bank transfer/USSD). Returns a payment URL.

**Auth required:** Yes

**Request body:**
```json
{
  "amount": 10000000,
  "email": "admin@acme.com"
}
```

**Response (200):**
```json
{
  "payment_url": "https://checkout.paystack.com/abc123",
  "reference": "DEP-1-1717200000",
  "amount": 10000000,
  "message": "Redirect to complete payment. Wallet will be credited on success."
}
```

### GET /v1/wallets/ledger
Get double-entry ledger entries.

**Query params:** `page`, `limit`

**Response (200):**
```json
{
  "entries": [ ... ],
  "total": 100,
  "page": 1,
  "limit": 50
}
```

### GET /v1/wallets/reconcile
Run wallet balance reconciliation against ledger.

**Response (200):** Reconciliation result object.

### POST /v1/wallets/sandbox/credit
Credit a virtual account in sandbox mode (testing only). **Admin only.**

**Request body:**
```json
{
  "account_number": "8012345678",
  "amount": 100000,
  "currency": "NGN"
}
```

**Response (200):**
```json
{ "status": true, "message": "Virtual bank account credited successfully" }
```

---

## 11. Account Holders / KYC

### POST /v1/wallets/account-holders
Create an account holder for KYC onboarding.

**Request body:**
```json
{
  "first_name": "John",
  "last_name": "Doe",
  "use_case": "Business",
  "type": "individual",
  "date_of_birth": "1990-05-15",
  "nationality": "NG",
  "email": "john@acme.com",
  "phone": "+2348133443000",
  "source_of_inflow": "business_income",
  "identification": {
    "type": "national_id",
    "number": "AB12345678"
  },
  "address": {
    "country": "NG",
    "state": "Lagos",
    "city": "Ikeja",
    "address": "123 Main Street"
  }
}
```

**Response (201):** Account holder details with reference.

### GET /v1/wallets/account-holders/{reference}/details
Get account holder details by reference.

**Response (200):** Account holder object.

### PATCH /v1/wallets/account-holders/{reference}/update-kyc
Update account holder KYC information.

**Request body:**
```json
{
  "first_name": "John",
  "last_name": "Doe",
  "source_of_inflow": "salary"
}
```

**Response (200):** Updated account holder object.

### POST /v1/wallets/files/generate-upload-url
Generate a file upload URL for KYC documents.

**Request body:**
```json
{
  "reference": "file-ref-001",
  "purpose": "kyc_document",
  "content_type": "image/jpeg"
}
```

**Response (200):** Upload URL and file reference.

---

## 12. Billing / Subscriptions

### GET /v1/billing/plans
List available subscription plans.

**Auth required:** Yes

**Response (200):** Array of plan objects.

### GET /v1/billing/subscription
Get current subscription for the business.

**Auth required:** Yes

**Response (200):** Subscription object.

### POST /v1/billing/subscribe
Subscribe to a plan. For paid plans, returns a Paystack payment URL.

**Auth required:** Yes

**Request body:**
```json
{
  "tier": "growth",
  "callback_url": "https://app.payflow.io/billing"
}
```

**Response (200):**
```json
{ "payment_url": "https://checkout.paystack.com/abc", "message": "Redirect to complete payment" }
```

### POST /v1/billing/cancel
Cancel current subscription. Downgrades to Free plan.

**Auth required:** Yes

**Response (200):**
```json
{ "message": "Subscription cancelled. Downgraded to Free plan." }
```

### GET /v1/billing/invoices
List billing invoices.

**Auth required:** Yes

**Query params:** `page`, `limit`

**Response (200):**
```json
{ "data": [ ... ], "total": 5 }
```

---

## 13. Loans

Employee loan management. Requires **admin** or **operator** role.

### POST /v1/loans
Create a new employee loan. Deductions are automatically applied to payroll.

**Request body:**
```json
{
  "employee_id": 3,
  "loan_amount": 50000000,
  "monthly_deduction": 5000000,
  "start_date": "2026-07-01",
  "description": "Staff emergency loan"
}
```

**Response (201):** Loan object.

### GET /v1/loans
List all loans for the business.

**Query params:** `page`, `limit`

**Response (200):**
```json
{ "data": [ ... ], "total": 10 }
```

### PATCH /v1/loans/{id}/cancel
Cancel an active loan.

**Response (200):**
```json
{ "message": "loan cancelled" }
```

---

## 14. Leave Management

### GET /v1/leave/types
List leave types for the business.

**Auth required:** Yes

**Response (200):** Array of leave type objects.

### POST /v1/leave/types
Create a new leave type.

**Auth required:** Yes

**Request body:**
```json
{
  "name": "Annual Leave",
  "default_days": 20,
  "requires_approval": true
}
```

**Response (201):** Leave type object.

### POST /v1/leave/requests
Submit a leave request.

**Auth required:** Yes

**Request body:**
```json
{
  "employee_id": 3,
  "leave_type_id": 1,
  "start_date": "2026-07-01T00:00:00Z",
  "end_date": "2026-07-10T00:00:00Z",
  "reason": "Family vacation"
}
```

**Response (201):** Leave request object.

### GET /v1/leave/requests
List leave requests for the business.

**Query params:** `page`, `limit`

**Response (200):**
```json
{ "requests": [ ... ], "total": 15 }
```

### POST /v1/leave/requests/{id}/approve
Approve a leave request.

**Response (200):**
```json
{ "message": "Leave approved" }
```

### POST /v1/leave/requests/{id}/reject
Reject a leave request.

**Request body:**
```json
{ "reason": "Insufficient staff coverage" }
```

**Response (200):**
```json
{ "message": "Leave rejected" }
```

### GET /v1/leave/balances/{employeeID}
Get leave balances for an employee.

**Query params:** `year` (optional, defaults to current year)

**Response (200):** Array of leave balance objects.

---

## 15. Notifications

### GET /v1/notifications
List notifications for the authenticated user.

**Auth required:** Yes

**Query params:** `page`, `limit`

**Response (200):**
```json
{ "data": [ ... ], "total": 20 }
```

### GET /v1/notifications/unread-count
Get count of unread notifications.

**Response (200):**
```json
{ "unread": 5 }
```

### PATCH /v1/notifications/{id}/read
Mark a notification as read.

**Response (200):**
```json
{ "message": "marked as read" }
```

### PATCH /v1/notifications/read-all
Mark all notifications as read.

**Response (200):**
```json
{ "message": "all marked as read" }
```

---

## 16. Verification

### GET /v1/verify/bank-account
Verify a bank account via Paystack.

**Auth required:** Yes

**Query params:** `bank_code=058`, `account_number=0123456789`

**Response (200):** Bank account verification result.

### GET /v1/verify/bvn
Verify a BVN.

**Auth required:** Yes

**Query params:** `bvn=12345678901`

**Response (200):** BVN verification result.

---

## 17. Self-Service

Employee self-service portal endpoints.

### GET /v1/me/profile
Get the authenticated employee's profile.

**Auth required:** Yes

**Response (200):** Employee object.

### PATCH /v1/me/bank-details
Update bank details (employee self-service).

**Auth required:** Yes

**Request body:**
```json
{
  "bank_name": "GTBank",
  "bank_code": "058",
  "bank_account_number": "0123456789"
}
```

**Response (200):** Updated employee object.

### GET /v1/me/payslips
Get payslip history for the authenticated employee.

**Auth required:** Yes

**Response (200):**
```json
[
  { "run_id": 1, "period": "2026-05", "status": "completed", "net_pay": 45000000 },
  { "run_id": 2, "period": "2026-06", "status": "completed", "net_pay": 47500000 }
]
```

---

## 18. Business Settings

**Admin only.**

### GET /v1/business/settings
Get business configuration.

**Response (200):** Business settings object.

### PATCH /v1/business/settings
Update business settings. All fields optional.

**Request body:**
```json
{
  "pension_enabled": true,
  "nhf_enabled": false,
  "nsitf_enabled": true,
  "paye_enabled": true,
  "payroll_requires_approval": true,
  "payroll_auto_process": false
}
```

**Response (200):** Updated business settings object.

### GET /v1/business/provider-keys
List org-level provider key overrides.

**Response (200):** Array of provider key summaries (keys are masked).

### PUT /v1/business/provider-keys/{provider}/{key}
Set an org-level provider key override.

**Path params:** `provider` = `paystack` or `korapay`. `key` = provider-specific key name.

**Request body:**
```json
{ "value": "sk_live_abc123..." }
```

**Response (200):**
```json
{ "message": "Provider key saved successfully" }
```

### DELETE /v1/business/provider-keys/{provider}/{key}
Delete an org-level provider key override.

**Response (204):** No content.

---

## 19. Audit Logs

**Admin only.**

### GET /v1/audit-logs
List audit logs for the business.

**Query params:** `page`, `limit`

**Response (200):**
```json
{
  "data": [ ... ],
  "total": 100,
  "page": 1,
  "limit": 20
}
```

---

## 20. Platform Admin

**Super admin only.** All routes under `/platform` require the `super_admin` role.

### GET /platform/stats
Get platform-wide statistics.

**Response (200):** Platform stats object (total organizations, users, employees, revenue).

### GET /platform/organizations
List all organizations on the platform.

**Query params:** `page`, `limit`

**Response (200):**
```json
{ "data": [ ... ], "total": 50 }
```

### POST /platform/organizations/{id}/suspend
Suspend an organization.

**Request body:**
```json
{ "reason": "Terms of service violation" }
```

**Response (200):**
```json
{ "message": "Organization suspended" }
```

### POST /platform/organizations/{id}/activate
Reactivate a suspended organization.

**Response (200):**
```json
{ "message": "Organization activated" }
```

### GET /platform/settings
List platform settings (encrypted API keys, SMTP config, etc.).

**Query params:** `category` (optional filter)

**Response (200):** Array of setting summaries.

### PUT /platform/settings/{key}
Set a platform setting (value is encrypted at rest with AES-256-GCM).

**Request body:**
```json
{
  "value": "sk_live_...",
  "description": "Production Paystack key",
  "category": "payment_providers"
}
```

**Response (200):**
```json
{ "message": "Setting updated successfully" }
```

### DELETE /platform/settings/{key}
Delete a platform setting.

**Response (204):** No content.

### GET /platform/reconciliation/provider
Manually trigger a provider reconciliation.

**Response (200):** Reconciliation result object.

---

## 21. Webhooks (Inbound)

These endpoints are called by external payment providers. **No auth required** (verified by signature/secret).

### POST /vfd/webhooks/inward-credit
VFD Bank inward credit notification.

### POST /vfd/webhooks/initial-inward-credit
VFD Bank initial inward credit notification.

### POST /vfd/webhooks/retrigger
VFD webhook retrigger.

### POST /korapay/webhooks/deposit
KoraPay deposit webhook. Called when a virtual account receives a credit.

### POST /paystack/webhooks/
Paystack transfer status webhook. Verified via `x-paystack-signature` header.

### POST /paystack/webhooks/billing
Paystack billing/subscription webhook. Verified via `x-paystack-signature` header.

---

## VFD Webhook Viewer (Authenticated)

### GET /v1/vfd/webhooks
List received VFD webhook notifications.

### GET /v1/vfd/webhooks/{id}
Get a specific webhook notification by ID.

### GET /v1/vfd/webhooks/account/{accountNumber}
Get webhook notifications for a specific account number.
