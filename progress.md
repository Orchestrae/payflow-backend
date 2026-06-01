# PayFlow — Progress Log

## Session: 2026-05-30 — All Remaining Issues

### Starting State
- 11 sprints completed
- 73 backend tests passing
- 38 frontend pages
- Backend: Railway (healthy)
- Frontend: Vercel (payflowio.vercel.app)
- Credentials: Paystack sandbox + Brevo SMTP configured

### Phase 1: Critical Financial Fixes
- [ ] 1.1 Fix batch transfer withdrawal recording
- [ ] 1.2 Fix failed batch transfer unlock
- [ ] 1.3 Wrap deposit in DB transaction
- [ ] 1.4 Fix deposit rollback

### Work Log
| Time | Action | Result |
|------|--------|--------|
| Start | Created task_plan.md, findings.md, progress.md | Planning complete |
| Phase 1.1 | Fix batch transfer withdrawal recording | Complete — RecordWithdrawal called for each successful payout |
| Phase 1.2 | Fix failed batch transfer unlock | Complete — UnlockBalance called for failed payouts |
| Phase 1.3-4 | Fix deposit atomicity | Complete — retry + critical logging on double failure |
| Phase 2 | Double-entry ledger | Complete — 6 tests passing |
| Phase 3 | Card/USSD deposit | Complete — POST /v1/wallets/deposit |
| Phase 4 | Onboarding | Complete — Free plan + default cadre on registration |
| Phase 5 | Validation | Complete — bank resolve on employee create, BVN verify endpoint, verification gate on payroll, name matching utility, field-specific errors, frontend verification badges |
| Phase 2.4-2.7 | Ledger wiring | Complete — ledger auto-records on deposit/withdrawal, reconciliation endpoint, frontend ledger page with debit/credit table + reconciliation summary |
| Phase 4.4,4.7 | Onboarding completion | Complete — CSV template (already existed), email verification (send on register, verify endpoint, resend endpoint, frontend verify-email page, migration 000023) |
| Phase 7.1 | Loan deductions | Complete — active loans deducted in payroll calculation, loan balance updated on payroll completion, auto-completes when fully repaid |
| Phase 7.5 | Billing webhook | Complete — POST /paystack/webhooks/billing handles charge.success, subscription.disable, invoice.payment_failed |
| Phase 8.3 | Velocity limits | Complete — per-business transfer rate limiting (10/hr single, 5/hr batch, 50/day, 20/day batch), 2 tests |
| Review | Bug fixes | Fixed: fee double-counting in ledger, CreateEmployee response format, BVN digit validation, UserID/StartDate mapping, verification gate employee loading |
| Phase 8.1 | Daily reconciliation | Complete — background job checks all wallets vs ledger balance every 24h, logs discrepancies |
| Phase 10.8 | Circuit breaker | Complete — per-provider failure tracking (5 failures → open, 60s → half-open → test → close), 5 tests |
| Phase 10.4 | Async email | Complete — AsyncEmailService wraps direct SMTP with Asynq queue (3 retries, fallback to direct), email handler registered in scheduler |
| Phase 10.1 | Split payroll | Complete — 976 lines → 3 files (256+336+356), 10/10 tests pass |
| Phase 6.1-6.3 | Credential storage | Complete — AES-256-GCM encryption, platform_settings table, super admin API (GET/PUT/DELETE /platform/settings), 5 encryption tests, migration 000024 |
| Phase 6.5-6.6 | Frontend settings | Complete — platform settings page with CRUD + masked values, test/live mode indicator in sidebar |
| Phase 7.3 | Leave enforcement | Complete — balance check on submit + re-check on approve, auto-create balance from defaults, auto-count business days, full handler + 7 routes |

### Session: 2026-05-31 — Priority A Fixes

| Fix | Action | Result |
|-----|--------|--------|
| A1 | Paystack deposit webhook: add business_id to metadata, extract in handleChargeSuccess, fallback to account_reference | Complete — `go build` + `go vet` clean |
| A2 | Fix leave API paths: `/v1/leave-types` → `/v1/leave/types` etc | Complete — 7 paths fixed in leave.ts |
| A3 | Fund Wallet button: modal with NGN input, calls initiateDeposit, redirects to payment_url | Complete — WalletOverviewPage updated |
| A4 | Create Virtual Account button: shown when no virtual_account_number, calls POST /v1/wallets/virtual-account | Complete — added API method + UI |
| A5 | CSV template download: added link to /v1/employees/import/template on ImportEmployeesPage | Complete |
| A6 | Bank re-verification on employee update: if BankCode/BankAccountNumber changed, re-run verification | Complete — `go build` + `go vet` clean |
| A7 | Amend + Reverse buttons: Amend (draft), Reverse (completed) with modal, API methods + React Query mutations | Complete — payroll.ts + useApi.ts + PayrollDetailPage |
| A8 | Request Leave form: modal with employee/type dropdowns, date range, reason, POST /v1/leave/requests | Complete — LeaveListPage updated |
| Verify | `go build`, `go vet`, `go test ./...`, `npx tsc --noEmit` | All pass — 0 errors, 10 test packages green |
