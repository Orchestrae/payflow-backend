# PayFlow Architecture

## System Architecture

```mermaid
graph TB
    subgraph "Client Layer"
        ADMIN[Admin Dashboard<br/>React SPA]
        SELF[Employee Portal<br/>React SPA]
        MOBILE[Future: Mobile App]
    end

    subgraph "API Gateway"
        LB[Railway Load Balancer<br/>TLS Termination]
    end

    subgraph "Application Layer"
        ROUTER[Chi Router<br/>CORS + Rate Limiting]
        AUTH_MW[JWT Auth Middleware<br/>Role Guard]

        subgraph "Handler Layer"
            AUTH_H[Auth Handler]
            EMP_H[Employee Handler]
            CADRE_H[Cadre Handler]
            PAY_H[Payroll Handler]
            TXF_H[Transfer Handler]
            WAL_H[Wallet Handler]
            BILL_H[Billing Handler]
            PLAT_H[Platform Handler]
            WHOOK_H[Webhook Handlers]
        end

        subgraph "Service Layer"
            AUTH_S[Auth Service]
            EMP_S[Employee Service]
            CADRE_S[Cadre Service]
            PAY_S[Payroll Service]
            TXF_S[Transfer Service]
            WAL_S[Wallet Service]
            LEDGER_S[Ledger Service]
            BILL_S[Billing Service]
            RECON_S[Reconciliation Service]
            NOTIF_S[Notification Service]
        end

        subgraph "Repository Layer"
            USER_R[User Repo]
            BIZ_R[Business Repo]
            EMP_R[Employee Repo]
            PAY_R[Payroll Repo]
            WAL_R[Wallet Repo]
            TXF_R[Transfer Repo]
            LEDGER_R[Ledger Repo]
        end

        subgraph "Background Jobs"
            SCHED[Asynq Scheduler]
            EMAIL_Q[Email Queue]
            PAYOUT_Q[Payout Processor]
            RECON_BG[Daily Reconciliation]
        end
    end

    subgraph "Data Layer"
        PG[(PostgreSQL 16<br/>Primary)]
        PG_READ[(PostgreSQL<br/>Read Replica)]
        REDIS[(Redis 7<br/>Cache + Queue)]
    end

    subgraph "External Services"
        KORA[KoraPay<br/>Virtual Accounts<br/>Transfers]
        PSK[Paystack<br/>Transfers<br/>Billing<br/>Verification]
        VFD[VFD Bank<br/>Corporate Accounts<br/>Transfers]
        SMTP_SVC[SMTP Server<br/>Brevo / SendGrid]
    end

    ADMIN & SELF --> LB --> ROUTER
    ROUTER --> AUTH_MW --> AUTH_H & EMP_H & CADRE_H & PAY_H & TXF_H & WAL_H & BILL_H & PLAT_H
    WHOOK_H --> WAL_S & TXF_S

    AUTH_H --> AUTH_S
    EMP_H --> EMP_S
    PAY_H --> PAY_S
    TXF_H --> TXF_S
    WAL_H --> WAL_S
    BILL_H --> BILL_S

    AUTH_S & EMP_S --> USER_R & BIZ_R & EMP_R
    PAY_S --> PAY_R & EMP_R
    WAL_S --> WAL_R
    TXF_S --> TXF_R
    LEDGER_S --> LEDGER_R

    USER_R & BIZ_R & EMP_R & PAY_R & WAL_R & TXF_R --> PG
    LEDGER_R --> PG_READ

    SCHED --> REDIS
    EMAIL_Q --> REDIS
    CADRE_S --> REDIS

    TXF_S --> KORA & PSK & VFD
    WAL_S --> KORA
    BILL_S --> PSK
    AUTH_S --> VFD
    NOTIF_S --> SMTP_SVC

    KORA & PSK & VFD -->|Webhooks| WHOOK_H
```

---

## Clean Architecture Layers

PayFlow follows a clean (hexagonal) architecture with four distinct layers:

```
Handler Layer (HTTP)  -->  Service Layer (Business Logic)  -->  Repository Layer (Data Access)
       |                           |                                     |
  Request DTOs              Domain Models                         PostgreSQL
  Response DTOs             Domain Errors                         (via GORM)
  Validation                Interfaces
```

### Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|---------------|
| **Handler** | `internal/api/handler/` | HTTP request/response handling, input validation, auth extraction |
| **Service** | `internal/service/` | Business logic, orchestration, domain rules |
| **Repository** | `internal/repository/postgres/` | Database queries, GORM operations |
| **Domain** | `internal/domain/` | Models, interfaces, errors, constants |
| **Platform** | `internal/platform/` | External service clients (Korapay, Paystack, VFD, SMTP) |

### Dependency Direction

Dependencies flow inward: handlers depend on services, services depend on repositories and domain interfaces, repositories depend on domain models. External services are abstracted behind interfaces defined in `domain/`.

---

## Data Flow Diagrams

### Business Registration

```mermaid
sequenceDiagram
    participant U as User
    participant API as Auth Handler
    participant AS as Auth Service
    participant VFD as VFD Bank
    participant DB as PostgreSQL
    participant SMTP as Email Service

    U->>API: POST /v1/auth/register
    API->>AS: RegisterBusiness()
    AS->>DB: Check email uniqueness
    AS->>AS: Hash password (bcrypt)
    AS->>DB: Create Business (transaction)
    AS->>DB: Create User (admin role)
    AS->>DB: Create default Cadre
    AS->>DB: Create free Subscription
    AS->>VFD: CreateCorporateAccount()
    VFD-->>AS: Account number
    AS->>DB: Store corporate account info
    AS->>SMTP: Send verification email
    AS-->>API: User + Corporate Account
    API-->>U: 201 Created
```

### Payroll Lifecycle

```mermaid
sequenceDiagram
    participant OP as Operator
    participant API as Payroll Handler
    participant PS as Payroll Service
    participant DB as PostgreSQL
    participant SCHED as Scheduler
    participant TXF as Transfer Service
    participant PROV as Payment Provider
    participant SMTP as Email Service

    Note over OP,SMTP: Phase 1: Create Payroll Run
    OP->>API: POST /v1/payroll-runs
    API->>PS: CreateAndStorePayrollRun()
    PS->>DB: Load all active employees + cadres
    PS->>PS: Calculate gross pay per employee
    PS->>PS: Calculate PAYE (Nigeria tax bands)
    PS->>PS: Calculate pension (8% employee + 10% employer)
    PS->>PS: Calculate NHF (2.5%)
    PS->>PS: Apply custom deduction rules
    PS->>PS: Deduct active loan repayments
    PS->>PS: Apply per-employee adjustments
    PS->>PS: Calculate net pay
    PS->>DB: Store payroll run + entries (status=draft)
    PS-->>API: PayrollRun with entries
    API-->>OP: 201 Created

    Note over OP,SMTP: Phase 2: Submit for Approval
    OP->>API: POST /v1/payroll-runs/{id}/submit
    API->>PS: SubmitForApproval()
    PS->>DB: Update status to pending_approval
    PS->>SMTP: Notify approvers
    PS-->>API: 202 Accepted

    Note over OP,SMTP: Phase 3: Approval
    OP->>API: POST /v1/payroll-runs/{id}/approve
    API->>PS: ApprovePayrollRun()
    PS->>DB: Update status to approved
    PS->>SCHED: Schedule payout job
    PS-->>API: PayrollRun

    Note over OP,SMTP: Phase 4: Processing
    SCHED->>PS: ProcessPayrollRun()
    loop For each employee entry
        PS->>TXF: ExecuteTransfer()
        TXF->>PROV: Send transfer
        PROV-->>TXF: Transfer result
        TXF->>DB: Record transfer
    end
    PS->>DB: Update status to completed
    PS->>SMTP: Notify admin of completion
```

### Wallet Deposit Flow

```mermaid
sequenceDiagram
    participant U as User
    participant API as Wallet Handler
    participant PSK as Paystack
    participant WH as Webhook Handler
    participant WS as Wallet Service
    participant LS as Ledger Service
    participant DB as PostgreSQL

    Note over U,DB: Option A: Paystack Checkout Deposit
    U->>API: POST /v1/wallets/deposit
    API->>PSK: Initialize transaction
    PSK-->>API: Payment URL
    API-->>U: Redirect URL

    U->>PSK: Complete payment
    PSK->>WH: POST /paystack/webhooks/ (charge.success)
    WH->>WH: Verify HMAC-SHA512 signature
    WH->>WS: RecordDeposit()
    WS->>DB: Credit wallet balance
    WS->>LS: Record ledger entry (double-entry)
    LS->>DB: Debit: Bank, Credit: Wallet

    Note over U,DB: Option B: Virtual Account Transfer
    U->>U: Bank transfer to virtual account
    PSK->>WH: POST /korapay/webhooks/deposit
    WH->>WS: RecordDeposit()
    WS->>DB: Credit wallet balance
    WS->>LS: Record ledger entry
```

---

## Background Jobs

| Job | Schedule | Description |
|-----|----------|-------------|
| **Payroll Payout Processing** | On-demand (after approval) | Processes approved payroll runs by executing transfers for each employee. Uses Asynq if Redis available, gocron otherwise. |
| **Email Delivery** | Async (queue) | Emails are queued via Asynq for reliable delivery with automatic retry. Falls back to synchronous sending if Redis unavailable. |
| **Daily Reconciliation** | Every 24 hours | Verifies wallet balances match ledger totals. Alerts admin on discrepancy. Starts 1 minute after server boot. |
| **Weekly Provider Reconciliation** | Every 7 days | Cross-checks internal transfer records with payment provider APIs. Identifies stuck/orphaned transactions. Starts 5 minutes after boot. |

---

## Security Model

### Authentication
- **JWT Bearer tokens** with configurable expiration (default 72 hours).
- Passwords hashed with **bcrypt**.
- Tokens contain: `user_id`, `business_id`, `role`, `exp`.

### Authorization (RBAC)
| Role | Scope |
|------|-------|
| `super_admin` | Platform-wide: all organizations, platform settings, reconciliation |
| `admin` | Business-wide: all features for their organization |
| `operator` | Employees, payroll creation/submission, transfers, loans |
| `approver` | View payroll, approve/reject payroll runs |
| `employee` | Self-service: profile, payslips, leave requests |

### Multi-Tenancy
- Every query is scoped by `business_id` extracted from the JWT.
- Repository methods enforce tenant isolation at the query level.
- No cross-tenant data access is possible through the API.

### Rate Limiting
- Global: 100 req/sec per IP.
- Auth endpoints: 5 req/sec per IP (brute force protection).
- Transfer creation: 10/hour, 50/day per business (velocity limiting).
- Batch transfers: 5/hour, 20/day per business.

### Encryption
- Platform settings (API keys, secrets) encrypted at rest with **AES-256-GCM**.
- Encryption key derived from first 32 bytes of `JWT_SECRET`.
- Org-level provider key overrides also encrypted.

### Webhook Verification
- Paystack: HMAC-SHA512 signature verification using `x-paystack-signature` header.
- VFD: Webhook secret verification.
- KoraPay: Signature verification via KoraPay client.

---

## Database Schema Overview

Current migration version: **000027**.

### Core Tables

| Table | Description |
|-------|-------------|
| `businesses` | Tenant organizations with statutory settings and payroll config |
| `users` | Admin, operator, approver, employee accounts with bcrypt passwords |
| `user_tokens` | Password reset tokens, invitation tokens, email verification tokens |
| `employees` | Employee records with bank details, cadre assignment, tax IDs |
| `cadres` | Salary structures with JSON earning components |
| `deduction_rules` | Custom deduction rules (percentage or flat, based on gross or basic) |

### Payroll Tables

| Table | Description |
|-------|-------------|
| `payroll_runs` | Payroll run header: period, status, totals, processing metadata |
| `payroll_run_entries` | Per-employee payroll breakdown: gross, PAYE, pension, NHF, net |
| `payroll_entry_details` | Line-item details for each entry (earning components, deductions) |

### Financial Tables

| Table | Description |
|-------|-------------|
| `wallets` | Business wallet with balance and virtual account details |
| `wallet_transactions` | Wallet transaction log (deposits, withdrawals) |
| `transfers` | Provider-agnostic transfer records |
| `vfd_transfers` | Legacy VFD-specific transfer records |
| `ledger_entries` | Double-entry accounting ledger |
| `employee_loans` | Loan records with monthly deduction tracking |

### Platform Tables

| Table | Description |
|-------|-------------|
| `subscription_plans` | Billing plan definitions (Free, Growth, Enterprise) |
| `subscriptions` | Business subscription state |
| `invoices` | Billing invoice records |
| `platform_settings` | Encrypted platform-wide settings |
| `org_provider_settings` | Encrypted org-level provider key overrides |

### Supporting Tables

| Table | Description |
|-------|-------------|
| `audit_logs` | Audit trail for admin actions |
| `notifications` | In-app notification records |
| `vfd_webhook_notifications` | VFD webhook payload archive |
| `leave_types` | Leave type definitions per business |
| `leave_requests` | Employee leave requests with approval status |
| `leave_balances` | Leave balance tracking per employee per year |

---

## Caching Strategy

PayFlow uses Redis as an optional cache layer. All caching is nil-safe -- if Redis is unavailable, operations fall through to the database.

### Cached Data

| Data | TTL | Invalidation |
|------|-----|-------------|
| Cadre list (per business) | Until mutation | Invalidated on create/update/delete |
| Deduction rules (per business) | Until mutation | Invalidated on create/update/delete |
| Wallet balance | Short-lived | Invalidated on deposit/withdrawal |

### Cache Pattern

```
Read: Cache -> DB (cache-aside / lazy loading)
Write: DB first -> Invalidate cache
```

The cache service is injected into services that benefit from caching. Services that do not benefit (payroll runs, transfers) always read from the database.
