# PayFlow

Automated payroll and payment platform for Nigerian SMEs. Built with Go, PostgreSQL, and Redis.

PayFlow handles the complete payroll lifecycle: employee onboarding, salary structure configuration, payroll calculation, multi-role approval workflow, and bulk salary disbursement via Korapay and VFD Bank.

## Documentation

| Document | Description |
|----------|-------------|
| **[Architecture](docs/ARCHITECTURE.md)** | System design, clean architecture layers, scaling decisions, roadmap |
| **[API Reference](docs/API_REFERENCE.md)** | Complete endpoint reference with request/response examples |
| **[Payroll Guide](docs/PAYROLL_GUIDE.md)** | Payroll workflow, state machine, configuration options |
| **[Wallet Guide](docs/WALLET_GUIDE.md)** | Virtual accounts, KYC, deposits, withdrawals, webhooks |
| **[Deployment Guide](docs/DEPLOYMENT.md)** | Docker setup, Railway deployment, environment variables, migrations |
| **[Access Guide](docs/ACCESS_GUIDE.md)** | How to login as super admin, business admin, operator, approver, employee |

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.24+ |
| Router | Chi v5 |
| Database | PostgreSQL 15 |
| ORM | GORM |
| Cache / Queue | Redis 7 (go-redis + Asynq) |
| Migrations | golang-migrate |
| Auth | JWT (HS256) |
| Payments | Korapay (primary), VFD Bank (fallback) |
| Config | Viper |
| Logging | zerolog |
| Containers | Docker, Docker Compose |
| Deployment | Railway |

## Quick Start

### Prerequisites

- [Go](https://go.dev/doc/install) 1.24+
- [Docker](https://www.docker.com/get-started/) and Docker Compose
- [golang-migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) CLI

### Setup

```bash
# Clone and enter
git clone <repo-url>
cd payflow

# Create environment file
cp .env.example .env
# Edit .env — at minimum set JWT_SECRET

# Start infrastructure (PostgreSQL, Redis, Mailhog)
docker-compose up -d

# Run migrations
make migrate-up

# Start the server
make run
```

The server starts on `http://localhost:8080`. Health check: `GET /health`.

### Docker Compose Services

| Service | Port | Purpose |
|---------|------|---------|
| PostgreSQL | 5433 | Primary database |
| Redis | 6379 | Caching + job queue |
| Mailhog | 8025 (UI), 1025 (SMTP) | Email testing |
| App | 8082 | Development server (hot-reload) |

### Required Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `JWT_SECRET` | Yes | Server will not start without this |
| `DB_URL` | Yes | PostgreSQL connection string |
| `KORAPAY_API_KEY` | For payments | Korapay API key |
| `PAYSTACK_SECRET_KEY` | For payments | Paystack secret key |
| `REDIS_URL` | No | Redis connection (app works without it) |
| `SMTP_HOST` | No | SMTP server (default: localhost for MailHog) |
| `SMTP_FROM` | No | Sender email (default: no-reply@payflow.com) |

See [Deployment Guide](docs/DEPLOYMENT.md) for the complete environment variable reference.

## Project Structure

```
cmd/server/main.go              Entry point, dependency injection
internal/
  api/handler/                   HTTP handlers
  api/middleware/                 Auth, Logger, RateLimiter, RequestID
  api/router.go                  Route definitions
  config/                        Configuration (Viper)
  domain/                        Domain models and interfaces
  platform/cache/                Redis client + cache-aside service
  platform/email/                SMTP + Hermes email templates
  platform/database/             PostgreSQL setup
  platform/korapay/              Korapay payment provider
  platform/scheduler/            Asynq (Redis) + gocron (fallback)
  platform/vfd/                  VFD Bank provider
  repository/postgres/           PostgreSQL implementations
  repository/repository.go       Repository interfaces
  service/                       Business logic layer
  service/provider/              Transfer provider manager
pkg/utils/                       JWT, password hashing
migrations/                      SQL migration files (000001-000011)
docs/                            Project documentation
```

## API Overview

All endpoints are under `/v1`. Authentication uses JWT Bearer tokens.

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/auth/register` | Register business + admin user |
| POST | `/v1/auth/login` | Login, receive JWT |
| POST | `/v1/auth/forgot-password` | Request password reset email |
| POST | `/v1/auth/reset-password` | Reset password with token |
| POST | `/v1/auth/accept-invitation` | Accept invite, set password |
| GET | `/health` | Readiness probe (DB + Redis) |
| GET | `/health/live` | Liveness probe |

### Protected Endpoints (require JWT)

| Area | Endpoints | Roles |
|------|-----------|-------|
| Employees | CRUD at `/v1/employees` | Admin, Operator |
| Cadres | CRUD at `/v1/cadres` | Admin, Operator |
| Deduction Rules | CRUD at `/v1/deduction-rules` | Admin |
| Payroll | Create, submit, approve, reject, process at `/v1/payroll-runs` | Admin, Operator, Approver |
| Transfers | Single + batch at `/v1/transfers` (provider selection) | Authenticated |
| Wallets | Virtual accounts, balance, transactions at `/v1/wallets` | Authenticated |
| Dashboard | Summary metrics at `/v1/dashboard` | Authenticated |
| Settings | Business config at `/v1/business/settings` | Admin |
| Reports | CSV/PDF downloads at `/v1/payroll-runs/{id}/reports/*` | Admin, Operator |
| Payslips | PDF downloads at `/v1/payroll-runs/{id}/payslips/*` | Admin, Operator |
| Invitations | Invite users at `/v1/auth/invite` | Admin |

See [API Reference](docs/API_REFERENCE.md) for complete request/response examples.

### Webhook Endpoints (public, signature-verified)

| Method | Path | Provider |
|--------|------|----------|
| POST | `/korapay/webhooks/deposit` | Korapay (HMAC-SHA256 mandatory) |
| POST | `/vfd/webhooks/inward-credit` | VFD Bank |
| POST | `/vfd/webhooks/initial-inward-credit` | VFD Bank |

## Key Features

### Payroll Engine
- Automatic gross/deduction/net calculation from cadre earning components
- Multi-role approval workflow (operator creates, approver approves)
- Configurable: auto-approval, auto-process, scheduled processing
- Bulk disbursement via Korapay (2-50 per batch) with VFD fallback

### Wallet System
- Virtual bank account provisioning (Korapay)
- Real-time balance tracking with atomic SQL operations
- Deposit via webhooks, withdrawal via transfers
- KYC/account holder management

### Nigerian Tax Compliance
- PAYE 2026 (Nigeria Tax Act 2025 brackets, NGN 800K tax-free threshold)
- Pension RSA (8% employee + 10% employer)
- NHF (2.5% of basic salary)
- NSITF (1% employer-only)
- Configurable per-business (enable/disable each deduction)

### Reports & Payslips
- PDF payslips with Hermes-styled layout (earnings, deductions, employer costs)
- PAYE return CSV for TaxProMax upload
- Pension schedule CSV for PFA remittance
- NHF schedule CSV for FMBN
- Bank schedule CSV for bulk transfers
- Bulk payslip ZIP download

### User Management
- Admin invites operators/approvers via email
- Password reset flow with secure token (1-hour expiry)
- Beautiful HTML emails via Hermes (payslip notifications, approvals, resets)
- Configurable SMTP (Brevo free tier, SendGrid, any provider)

### Security
- JWT authentication with mandatory secret
- HMAC-SHA256/SHA512 webhook verification (Korapay, Paystack, VFD)
- Rate limiting (100/sec global, 5/sec on auth)
- Request logging with trace IDs
- Role-based access control (admin, operator, approver)

### Scaling
- Redis caching (cache-aside pattern, 5 min TTL)
- Asynq persistent job queue (3 retries, 10 min timeout)
- Batch database operations (FindByIDs, CreateInBatches)
- Atomic wallet operations (no race conditions)
- Performance indexes on hot query paths

See [Architecture](docs/ARCHITECTURE.md) for scaling roadmap and design decisions.

## Migrations

```bash
# Apply all pending migrations
make migrate-up

# Revert last migration
make migrate-down

# Create new migration
make migrate-create name=add_new_feature
```

Current version: 000013. See [Deployment Guide](docs/DEPLOYMENT.md) for migration strategy.

## Testing

```bash
# Import the Postman collection
docs/PayFlow_API.postman_collection.json

# Registration flow order:
# 1. Register business
# 2. Login
# 3. Create deduction rules
# 4. Create cadres
# 5. Add employees
# 6. Create payroll run
# 7. Submit for approval
# 8. Approve
# 9. Process
```

See [API Reference](docs/API_REFERENCE.md) for curl examples and testing instructions.
