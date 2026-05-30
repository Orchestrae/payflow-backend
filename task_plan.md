# PayFlow — Master Task Plan
> All remaining work to production-grade financial SaaS

**Goal:** Fix all 49 remaining issues, build financial infrastructure properly, update frontend for every backend change.

**Status:** 11 sprints completed. 7 test packages passing. 166 Go files. 38 frontend pages.

---

## Phase 1: CRITICAL Financial Fixes (Sprint F2)
> Fix money-losing bugs before anything else

| # | Task | File(s) | Status |
|---|------|---------|--------|
| 1.1 | Fix batch transfer: record withdrawal for each successful payout | `transfer_service.go` | `complete` |
| 1.2 | Fix batch transfer: unlock balance for failed payouts | `transfer_service.go` | `complete` |
| 1.3 | Wrap deposit in DB transaction (balance + record atomic) | `wallet_service.go` | `complete` |
| 1.4 | Fix deposit rollback to use same DB transaction | `wallet_service.go` | `complete` |

## Phase 2: Double-Entry Ledger (Sprint L)
> Proper financial accounting — every kobo tracked

| # | Task | File(s) | Status |
|---|------|---------|--------|
| 2.1 | Create `ledger_entries` domain model (debit/credit, account type) | `domain/ledger.go` | `complete` |
| 2.2 | Create ledger repository + migration 000021 | `postgres/ledger_repo.go` | `complete` |
| 2.3 | Create ledger service (record entry, get balance, reconcile) | `service/ledger_service.go` | `complete` |
| 2.4 | Wire ledger into deposit flow (credit wallet, debit external) | `wallet_service.go` | `pending` |
| 2.5 | Wire ledger into withdrawal flow (debit wallet, credit external) | `wallet_service.go` | `pending` |
| 2.6 | Reconciliation endpoint: SUM(credits) - SUM(debits) vs balance | `handler/ledger_handler.go` | `pending` |
| 2.7 | Frontend: transaction ledger page | `frontend/pages/wallet/` | `pending` |

## Phase 3: Card/USSD Deposit Support (Sprint L cont.)
> Accept more payment methods

| # | Task | File(s) | Status |
|---|------|---------|--------|
| 3.1 | Paystack Checkout: initialize payment (card/transfer/USSD) | `platform/billing/paystack_billing.go` | `complete` |
| 3.2 | Deposit via Paystack Checkout endpoint: POST /v1/wallets/deposit | `handler/wallet_handler.go` | `complete` |
| 3.3 | Handle payment callback + webhook for all methods | `handler/paystack_webhook_handler.go` | `complete` |
| 3.4 | Frontend: wallet API deposit method | `frontend/api/wallet.ts` | `complete` |

## Phase 4: Onboarding (Sprint H — remaining)
> New users can actually use the product

| # | Task | File(s) | Status |
|---|------|---------|--------|
| 4.1 | Default cadre on registration | `auth_service.go` | `complete` |
| 4.2 | Welcome email on registration | `auth_service.go` | `complete` |
| 4.3 | Auto-assign Free plan on registration | `auth_service.go`, `billing_service.go` | `pending` |
| 4.4 | CSV template download endpoint | `handler/employee_handler.go` | `pending` |
| 4.5 | Frontend: onboarding checklist on dashboard | `frontend/pages/dashboard/` | `pending` |
| 4.6 | Frontend: better empty states with CTAs | `frontend/components/` | `pending` |
| 4.7 | Email verification flow (send token, verify endpoint) | `auth_service.go`, `auth_handler.go` | `pending` |

## Phase 5: Validation (Sprint I)
> Prevent bad data, verify identities

| # | Task | File(s) | Status |
|---|------|---------|--------|
| 5.1 | BVN verification via Paystack API | `service/verification_service.go` | `pending` |
| 5.2 | Bank account resolve on employee creation | `handler/employee_handler.go` | `pending` |
| 5.3 | Verification gate before payroll (bypass for Free plan) | `payroll_service.go` | `pending` |
| 5.4 | Validate monetary inputs > 0 | `request/*.go` | `pending` |
| 5.5 | Validate bank codes against known list | `utils/banks.go` | `pending` |
| 5.6 | Payroll dry-run: check all employees have valid bank details | `payroll_service.go` | `pending` |
| 5.7 | Better error messages (field-specific, not generic) | `response/response.go` | `pending` |
| 5.8 | Frontend: validation on all forms | `frontend/pages/` | `pending` |

## Phase 6: Credential Management (Sprint G)
> Manage API keys from dashboard, not Railway

| # | Task | File(s) | Status |
|---|------|---------|--------|
| 6.1 | Platform settings DB table (encrypted key storage) | `domain/platform_setting.go` | `pending` |
| 6.2 | AES-256 encryption utility for sensitive fields | `pkg/utils/encryption.go` | `pending` |
| 6.3 | Super admin: manage Paystack/SMTP/Termii keys via UI | `handler/platform_handler.go` | `pending` |
| 6.4 | Org-specific provider key override | `service/transfer_service.go` | `pending` |
| 6.5 | Test/live mode indicator in frontend | `frontend/components/layout/` | `pending` |
| 6.6 | Frontend: platform settings page | `frontend/pages/platform/` | `pending` |

## Phase 7: Product Gaps
> Features that exist in domain but aren't wired

| # | Task | File(s) | Status |
|---|------|---------|--------|
| 7.1 | Loan deductions in payroll calculation | `payroll_service.go` | `pending` |
| 7.2 | Employee self-service auth (create login for employee) | `auth_service.go` | `pending` |
| 7.3 | Leave balance enforcement (block over-limit) | `leave_service.go` | `pending` |
| 7.4 | Payroll amendment/reversal | `payroll_service.go` | `pending` |
| 7.5 | Billing webhook for subscription payment events | `handler/billing_webhook_handler.go` | `pending` |
| 7.6 | Frontend: employee self-service portal (separate layout) | `frontend/pages/self-service/` | `pending` |

## Phase 8: Reconciliation & Velocity Controls
> Financial safety mechanisms

| # | Task | File(s) | Status |
|---|------|---------|--------|
| 8.1 | Daily reconciliation job (internal balance vs transaction sum) | `service/reconciliation_service.go` | `pending` |
| 8.2 | Weekly provider reconciliation (internal vs Paystack balance) | `service/reconciliation_service.go` | `pending` |
| 8.3 | Transfer velocity limits (max/day, max/hour) | `middleware/velocity.go` | `pending` |
| 8.4 | Anomaly detection alerts (unusual transfer patterns) | `service/alert_service.go` | `pending` |
| 8.5 | Frontend: reconciliation dashboard (super admin) | `frontend/pages/platform/` | `pending` |

## Phase 9: Testing (Sprint J)
> Catch bugs before users do

| # | Task | File(s) | Status |
|---|------|---------|--------|
| 9.1 | Integration tests with testcontainers (registration, payroll) | `*_test.go` | `pending` |
| 9.2 | Frontend tests (Vitest + React Testing Library) | `frontend/**/*.test.tsx` | `pending` |
| 9.3 | PAYE accuracy tests (verify against GRA calculator) | `tax/engine_test.go` | `pending` |
| 9.4 | Ghana SSNIT accuracy tests | `tax/ghana/engine_test.go` | `pending` |
| 9.5 | Load test (k6 — 100 concurrent payroll runs) | `tests/load/` | `pending` |
| 9.6 | Security scan (SQL injection, XSS, exposed secrets) | `tests/security/` | `pending` |
| 9.7 | End-to-end API contract tests (Postman CLI) | `tests/e2e/` | `pending` |

## Phase 10: Code Quality (Sprint K)
> Clean up for maintainability

| # | Task | File(s) | Status |
|---|------|---------|--------|
| 10.1 | Split payroll_service.go (867 lines → 3 files) | `payroll_*.go` | `pending` |
| 10.2 | Split transfer_service.go → single + batch + retry | `transfer_*.go` | `pending` |
| 10.3 | Split wallet_handler.go → wallet + webhook | `handler/wallet_*.go` | `pending` |
| 10.4 | Route emails through Asynq (retry, not goroutine) | `service/*.go` | `pending` |
| 10.5 | Extend cache to deduction rules + wallet balance | `service/deduction_service.go` | `pending` |
| 10.6 | Read replica for report queries | `repository/postgres/` | `pending` |
| 10.7 | Async payroll processing (return 202, poll status) | `payroll_service.go` | `pending` |
| 10.8 | Circuit breaker for payment providers | `provider/manager.go` | `pending` |

---

## Execution Order
```
Phase 1 (CRITICAL) → Fix money bugs NOW
Phase 2 (HIGH) → Double-entry ledger
Phase 3 (HIGH) → Card/USSD deposits
Phase 4 (HIGH) → Onboarding completion
Phase 5 (HIGH) → Validation
Phase 6 (MEDIUM) → Credential management
Phase 7 (MEDIUM) → Product gap wiring
Phase 8 (MEDIUM) → Reconciliation
Phase 9 (MEDIUM) → Testing
Phase 10 (LOW) → Code quality
```

## Errors Encountered
| Error | Attempt | Resolution |
|-------|---------|------------|
| (none yet) | | |
