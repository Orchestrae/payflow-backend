# PayFlow — Findings & Research

## Financial Audit (2026-05-30)

### Critical Bugs Found
1. **Batch transfer withdrawals NOT recorded** — balance locked but never decremented for batch payouts
2. **Failed batch payout locks never released** — locked_balance stuck forever
3. **Deposit recording not in DB transaction** — if transaction record fails, rollback is best-effort

### Current Accounting Model
- Single-balance (NOT double-entry)
- No reconciliation between balance and SUM(transactions)
- wallet_transactions logs BalanceBefore/BalanceAfter (audit trail)
- CHECK constraints prevent negative balance

### Paystack Capabilities
- DVA limit: 1,000 per business (can request increase)
- Payment methods: Card, Bank Transfer, USSD, Apple Pay, Mobile Money
- Pay-with-Transfer: temporary accounts (30 min expiry)
- Transfer limits: ~NGN 1M/day for verified businesses
- BVN verification: NGN 10/call (10 free/month)
- Account resolve: FREE

### BVN/RC Verification APIs
- **BVN verify**: Paystack (cheapest, already integrated), Korapay, Mono, Dojah
- **RC verify**: Korapay CAC lookup, Dojah, VerifyMe, CAC public search
- **Bank resolve**: Paystack (free), already implemented

### Regulatory Considerations
- CBN fined Paystack's Zap NGN 250M for operating wallet without license
- PayFlow holds money in Paystack DVA — Paystack handles compliance
- Individual wallets (B2C) would need PSP/MMO license from CBN
- BVN consent mandatory via NIBSS iGree platform

### Architecture Decision: Double-Entry Ledger
- RECOMMENDED for any product handling other people's money
- Every transaction creates 2 entries (debit + credit)
- Self-balancing: SUM(debits) == SUM(credits) always
- Enables reconciliation, audit, and financial reporting
