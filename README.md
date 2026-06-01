# PayFlow

Automated payroll and payment infrastructure for Nigerian SMEs. PayFlow handles the complete payroll lifecycle -- employee onboarding, salary structure configuration, payroll calculation with Nigerian tax compliance (PAYE, pension, NHF, NSITF), multi-role approval workflows, and bulk salary disbursement via multiple payment providers.

## Value Proposition

- **Nigerian Tax Compliance:** Automatic PAYE, pension (8% employee + 10% employer), NHF (2.5%), and NSITF calculations following FIRS guidelines.
- **Multi-Provider Payments:** Transfers via Korapay, Paystack, and VFD Bank with automatic failover.
- **Virtual Wallet:** KoraPay-powered virtual bank accounts for instant deposits and disbursements.
- **Self-Service Portal:** Employees can view payslips, request leave, and update bank details.
- **Platform Multi-Tenancy:** Each business operates in complete isolation with its own data and configuration.

## Tech Stack

### Backend
| Component | Technology |
|-----------|-----------|
| Language | Go 1.26 |
| Router | Chi v5 |
| ORM | GORM (PostgreSQL) |
| Database | PostgreSQL 16 |
| Cache/Queue | Redis + Asynq |
| Auth | JWT (golang-jwt/v5) |
| Email | SMTP (Hermes v2 templates) |
| Logging | zerolog |
| Config | Viper |
| Validation | go-playground/validator |
| PDF Generation | Maroto v2 |
| Rate Limiting | golang.org/x/time/rate |
| Migrations | golang-migrate |

### Frontend
| Component | Technology |
|-----------|-----------|
| Framework | React 19 + TypeScript |
| Build Tool | Vite |
| Styling | Tailwind CSS v4 |
| State Management | Zustand |
| Data Fetching | TanStack Query v5 |
| Forms | React Hook Form + Zod |
| Routing | React Router v7 |
| Charts | Recharts |
| Animations | Framer Motion |
| Icons | Lucide React |

## Architecture

```mermaid
graph TB
    subgraph "Frontend (Vercel)"
        UI[React SPA]
    end

    subgraph "Backend (Railway)"
        API[Chi Router + Middleware]
        AUTH[JWT Auth Middleware]
        HANDLERS[Domain Handlers]
        SERVICES[Service Layer]
        REPOS[Repository Layer]
        SCHEDULER[Asynq / gocron Scheduler]
    end

    subgraph "Data Layer"
        PG[(PostgreSQL)]
        REDIS[(Redis)]
    end

    subgraph "External Providers"
        KORA[KoraPay API]
        VFD[VFD Bank API]
        PSK[Paystack API]
        SMTP[SMTP Server]
    end

    UI -->|HTTPS + JWT| API
    API --> AUTH --> HANDLERS --> SERVICES
    SERVICES --> REPOS --> PG
    SERVICES --> REDIS
    SCHEDULER --> REDIS
    SERVICES --> KORA
    SERVICES --> VFD
    SERVICES --> PSK
    SERVICES --> SMTP
```

## Project Structure

```
payflow/
  cmd/server/main.go          # Application entry point and dependency wiring
  internal/
    api/
      handler/                 # HTTP handlers (one per domain)
      middleware/               # Auth, CORS, rate limiting, role guards
      request/                 # Request DTOs with validation tags
      response/                # Response DTOs and helpers
      router.go                # Route definitions
    config/config.go           # Environment configuration (Viper)
    domain/                    # Domain models, errors, interfaces
    platform/                  # External provider clients (KoraPay, VFD, Paystack, SMTP)
      billing/                 # Paystack billing client
      cache/                   # Redis client + cache service
      database/                # GORM database initialization
      email/                   # SMTP email + async queue
      korapay/                 # KoraPay API client
      paystack/                # Paystack API client
      scheduler/               # Asynq + gocron schedulers
      vfd/                     # VFD Bank API client
    repository/postgres/       # PostgreSQL repository implementations
    service/                   # Business logic (clean architecture)
      platform/                # Platform admin + billing services
      provider/                # Transfer provider abstraction
      report/                  # CSV/PDF report generation
      vfd/                     # VFD-specific services
  migrations/                  # SQL migrations (000001 through 000027)
  frontend/                    # React SPA
    src/
      api/                     # Axios API client
      components/              # Reusable UI components
      hooks/                   # Custom React hooks
      pages/                   # Page components by domain
      routes.tsx               # Route definitions
      store/                   # Zustand stores
      types/                   # TypeScript type definitions
  docs/                        # Documentation
  tests/                       # Integration tests
```

## Local Development Setup

### Prerequisites

- Go 1.26+
- Node.js 20+ and npm
- PostgreSQL 16+
- Redis 7+ (optional but recommended)
- golang-migrate CLI

### 1. Clone and configure

```bash
git clone https://github.com/Orchestrae/payflow-backend.git payflow
cd payflow
cp .env.example .env
# Edit .env with your local database credentials and API keys
```

### 2. Start infrastructure

```bash
# Using Docker Compose (recommended)
make up

# Or manually start PostgreSQL and Redis
```

### 3. Run migrations

```bash
# Set your database connection string
export DSN="postgres://payflow_user:payflow_secret@localhost:5433/payflow_db?sslmode=disable"

# Apply all migrations
make migrate-up

# Check migration status
make migrate-status
```

### 4. Start the backend

```bash
make run
# Server starts on http://localhost:8080
# Health check: http://localhost:8080/health
```

### 5. Start the frontend

```bash
cd frontend
npm install
npm run dev
# Frontend starts on http://localhost:5173
```

### Minimum `.env` for local development

```env
DB_URL=postgres://payflow_user:payflow_secret@localhost:5433/payflow_db?sslmode=disable
JWT_SECRET=your-32-char-minimum-secret-key-here
REDIS_URL=redis://localhost:6379
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000
ENABLE_AUTO_MIGRATION=false
```

## Running Tests

```bash
# Run all tests with race detection
make test

# Run specific package tests
go test ./internal/service/... -v

# Lint
make lint
```

## Deployment

PayFlow is designed for deployment on Railway (backend) and Vercel (frontend).

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for the complete deployment guide, and [docs/OPERATOR_ACTIONS.md](docs/OPERATOR_ACTIONS.md) for the post-deployment checklist and environment variable reference.

## Key Make Targets

```
make run             # Run the application
make build           # Build binary to ./bin/payflow
make test            # Run tests with race detection
make vet             # Run go vet
make lint            # Run golangci-lint
make up              # Start Docker Compose services
make down            # Stop Docker Compose services
make migrate-up      # Apply all pending migrations
make migrate-down    # Revert last migration
make migrate-create  # Create a new migration file
make migrate-status  # Show current migration version
```

## Documentation

- [API Flow Guide](docs/API_FLOW_GUIDE.md) - Complete endpoint reference
- [Architecture](docs/ARCHITECTURE.md) - System design and data flows
- [Deployment](docs/DEPLOYMENT.md) - Railway + Vercel deployment guide
- [Operator Actions](docs/OPERATOR_ACTIONS.md) - Environment variables, webhooks, post-deploy checklist
