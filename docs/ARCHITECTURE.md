# PayFlow Architecture

PayFlow is a multi-tenant payroll SaaS platform built in Go. It handles employee management, payroll processing, wallet-based fund disbursement, and integrations with Nigerian payment providers (Korapay, VFD Bank).

---

## Table of Contents

- [Tech Stack](#tech-stack)
- [Directory Structure](#directory-structure)
- [Clean Architecture Layers](#clean-architecture-layers)
- [Request Lifecycle](#request-lifecycle)
- [Domain Models](#domain-models)
- [Payroll State Machine](#payroll-state-machine)
- [Key Design Patterns](#key-design-patterns)
- [Security](#security)
- [Caching and Job Queue](#caching-and-job-queue)
- [Payment Providers](#payment-providers)
- [Scaling History and Roadmap](#scaling-history-and-roadmap)
- [Cross-References](#cross-references)

---

## Tech Stack

| Component         | Technology                                  |
|-------------------|---------------------------------------------|
| Language          | Go 1.24+                                    |
| HTTP Router       | Chi v5                                      |
| ORM               | GORM                                        |
| Database          | PostgreSQL 15                               |
| Cache / Queue     | Redis 7 (caching + Asynq job queue)         |
| Authentication    | JWT (HS256)                                 |
| Payment (primary) | Korapay (virtual accounts, disbursements)   |
| Payment (fallback)| VFD Bank (corporate accounts, transfers)    |
| Logging           | zerolog (structured JSON)                   |
| Configuration     | Viper                                       |
| Email             | SendGrid (production), Mailhog (dev)        |
| Migrations        | golang-migrate (SQL files, version 000011)  |
| Deployment        | Docker + Railway                            |

---

## Directory Structure

```
cmd/server/main.go              -- Entry point, dependency injection wiring

internal/
  api/
    handler/                    -- HTTP handlers (11 files)
    middleware/                  -- Auth, Logger, RateLimiter, RequestID
    request/                    -- Request DTOs + validation
    response/                   -- Response DTOs + error mapping
    router.go                   -- Chi router setup

  config/                       -- Viper-based configuration

  domain/                       -- Domain models + interfaces (14 files)

  platform/
    cache/                      -- Redis client + cache-aside service
    database/                   -- GORM PostgreSQL setup
    korapay/                    -- Korapay payment provider client
    scheduler/                  -- Asynq (Redis) + gocron (fallback)
    sendgrid/                   -- Email service
    vfd/                        -- VFD Bank provider client

  repository/
    postgres/                   -- PostgreSQL implementations
    repository.go               -- Repository interfaces

  service/
    provider/                   -- Transfer provider manager (strategy pattern)
    vfd/                        -- VFD-specific services
    *.go                        -- Core services (auth, payroll, transfer, wallet, etc.)

pkg/utils/                      -- JWT, password hashing utilities

migrations/                     -- golang-migrate SQL files (000001 through 000011)
```

---

## Clean Architecture Layers

The codebase follows clean architecture with strict dependency direction. Outer layers depend on inner layers; inner layers never import from outer layers.

```
  Handler  -->  Service  -->  Repository  -->  Database
     |              |
     v              v
  Request/       Domain
  Response       Models &
  DTOs           Interfaces
```

**Domain** -- Entities, repository interfaces, service interfaces, and typed errors. Zero external dependencies.

**Service** -- Business logic and orchestration. Services depend only on domain interfaces, never on concrete implementations.

**Repository** -- Data access behind interfaces. The `postgres/` package is the sole implementation. Some repositories use domain models directly with GORM tags (deduction_rule, cadre, payroll). Others use internal postgres models with `ToDomain`/`FromDomain` converters (user, business, employee).

**Handler** -- HTTP request/response translation. Parses requests into DTOs, calls services, maps results to response DTOs or error responses.

**Platform** -- External integrations (database connections, Redis, payment providers, email). These are injected into services at startup.

---

## Request Lifecycle

1. HTTP request arrives at Chi router.
2. Global middleware executes in order: RequestID, Logger, RateLimiter.
3. Route-specific middleware (Auth) validates JWT and injects claims into context.
4. Handler parses and validates the request body into a request DTO.
5. Handler calls the appropriate service method.
6. Service executes business logic, calling repositories and platform integrations.
7. Handler maps the result to a response DTO and writes JSON.

Claims extraction in handlers always uses `middleware.GetClaimsFromContext(ctx)`. Direct context value access is not used.

---

## Domain Models

Domain models carry both GORM struct tags (for direct-model repositories) and JSON tags (snake_case, for API serialization).

Monetary values are stored as `int64` in the smallest currency unit (kobo for NGN). For example, `500000` represents NGN 5,000.00.

Key entities:

- **Business** -- Tenant. All data is scoped to a business.
- **User** -- Business owner / admin.
- **Employee** -- Belongs to a business and a cadre.
- **Cadre** -- Employee group with associated deduction rules.
- **DeductionRule** -- Tax, pension, or custom deduction tied to a cadre.
- **Payroll** -- A pay run containing payroll items for employees.
- **PayrollItem** -- One employee's line in a payroll (gross, deductions, net).
- **Wallet** -- Business wallet for funding payroll disbursements.
- **Transfer** -- Individual disbursement to an employee's bank account.

---

## Payroll State Machine

A payroll moves through these states:

```
draft --> pending_approval --> approved --> processing --> completed
                |                             |
                v                             v
            rejected                        failed
```

- **draft**: Payroll created, items can be edited.
- **pending_approval**: Submitted for review.
- **approved**: Authorized, ready for disbursement.
- **processing**: Transfers in flight.
- **completed**: All transfers settled.
- **rejected**: Returned during approval.
- **failed**: Transfer errors during processing.

See [Payroll Guide](./PAYROLL_GUIDE.md) for the full payroll lifecycle.

---

## Key Design Patterns

### 1. Strategy Pattern -- Transfer Provider Manager

`TransferProviderManager` in `internal/service/provider/` implements a strategy pattern for payment disbursements. Korapay is the primary provider; VFD Bank is the fallback. The manager selects the active provider at runtime and can fall back automatically on failure.

### 2. Cache-Aside with Nil-Safe Degradation

The generic `GetOrLoad[T]` function in `internal/platform/cache/` implements cache-aside:
- Check Redis for cached value.
- On miss, call the loader function, cache the result (5 min TTL), and return.
- If Redis is unavailable, the loader is called directly. The application works without Redis.

### 3. Interface-Based Repositories

All repositories are defined as interfaces in the domain layer. The `postgres/` package provides the only concrete implementation. This enables testing with mocks and keeps the service layer decoupled from the database.

### 4. Atomic Wallet Operations

Wallet balance updates use atomic SQL (`UPDATE wallets SET balance = balance - ? WHERE balance >= ? AND id = ?`). There is no read-modify-write cycle, eliminating race conditions on concurrent debits.

### 5. Worker Pool for Bulk Transfers

Payroll disbursement fans out transfers to a pool of 5 concurrent workers communicating via Go channels. This bounds concurrency against the payment provider API while processing transfers in parallel.

---

## Security

- **JWT**: HS256 tokens. The signing secret is required at startup (no insecure defaults).
- **Webhook Verification**: Mandatory HMAC-SHA256 verification for both Korapay and VFD Bank webhooks. The raw request body is hashed with the provider's secret key.
- **Rate Limiting**: 100 requests/second globally, 5 requests/second on authentication endpoints.
- **Request Tracing**: Every request gets a unique request ID via middleware; zerolog includes it in all log entries.
- **CORS**: Restricted to `payflowio.netlify.app` and `localhost` (development).
- **Password Hashing**: bcrypt via `pkg/utils/`.

---

## Caching and Job Queue

Both caching and the job queue run on Redis 7.

**Caching**: Cache-aside pattern on read-heavy entities (e.g., cadres, 5 min TTL). The cache service degrades gracefully if Redis is down.

**Job Queue**: Asynq processes background jobs (e.g., scheduled payroll runs) with persistent storage in Redis, automatic retries (3 attempts), and dead-letter handling. Asynq replaced an earlier gocron-based scheduler; gocron remains as a fallback.

**Health Checks**: The `/health` endpoint performs deep checks, pinging both PostgreSQL and Redis to confirm connectivity.

---

## Payment Providers

### Korapay (Primary)

- Virtual account creation for businesses.
- Bank account disbursements for payroll transfers.
- Account holder KYC.
- Webhook notifications with HMAC-SHA256 verification (hash the `data` field with the secret key).
- Sandbox keys prefixed with `sk_test_`.

### VFD Bank (Fallback)

- Corporate account management.
- Bank transfers.
- Webhook verification (HMAC-SHA256).
- Integration is optional; the auth service handles a nil VFD service gracefully.

The `TransferProviderManager` abstracts both behind a common interface so the payroll service does not couple to either provider directly.

See [Wallet Guide](./WALLET_GUIDE.md) for funding and disbursement flows.

---

## Scaling History and Roadmap

### Phase 1 -- Quick Wins (Complete)

Target: ~500 organizations.

- Fixed N+1 queries with batch cadre and employee loading.
- Batch transfer inserts using GORM `CreateInBatches`.
- Atomic wallet operations at the SQL level.
- Performance indexes and CHECK constraints (migration 000011).

### Phase 2 -- Security and Redis (Complete)

Target: ~2,000 organizations.

- JWT secret enforcement (no insecure defaults).
- Mandatory webhook HMAC verification.
- Rate limiting middleware.
- Request logging and request ID middleware.
- Redis caching (cache-aside, 5 min TTL).
- Asynq job queue (persistent, 3 retries).
- Deep health checks on `/health`.

### Phase 3 -- Planned

Target: ~5,000 organizations.

- Read replica for reporting queries.
- Async payroll processing (fully event-driven).
- Circuit breaker on payment provider calls.

### Phase 4 -- Planned

Target: 50,000+ organizations.

- Table partitioning (payroll items, transfers by date).
- Event-driven architecture (publish/subscribe).
- Service extraction (payroll engine, transfer service as independent services).

---

## Cross-References

- [API Reference](./API_REFERENCE.md) -- Endpoint documentation and request/response schemas.
- [Payroll Guide](./PAYROLL_GUIDE.md) -- Payroll lifecycle, state transitions, and disbursement flow.
- [Wallet Guide](./WALLET_GUIDE.md) -- Wallet funding, balance management, and transfer settlement.
- [Deployment Guide](./DEPLOYMENT.md) -- Docker, Railway, and environment configuration.
- [Postman Collection](./PayFlow_API.postman_collection.json) -- Importable API collection for testing.
