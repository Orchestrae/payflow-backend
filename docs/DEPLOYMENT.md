# PayFlow Deployment Guide

This document covers Docker local development, Railway production deployment, environment variable reference, and the database migration strategy.

See also:
- [Architecture](./ARCHITECTURE.md)
- [API Reference](./API_REFERENCE.md)

---

## Table of Contents

1. [Local Development with Docker](#local-development-with-docker)
2. [Production Dockerfile](#production-dockerfile)
3. [Railway Deployment](#railway-deployment)
4. [Environment Variables](#environment-variables)
5. [Database Migrations](#database-migrations)
6. [Health Checks](#health-checks)
7. [Troubleshooting](#troubleshooting)

---

## Local Development with Docker

Docker Compose provides a full local stack. Start everything with:

```bash
make up
# or
docker-compose up -d
```

### Services

| Service   | Image                  | Host Port | Container Port | Purpose                        |
|-----------|------------------------|-----------|----------------|--------------------------------|
| db        | postgres:15-alpine     | 5433      | 5432           | PostgreSQL database            |
| redis     | redis:7-alpine         | 6379      | 6379           | Caching (optional)             |
| mailhog   | mailhog/mailhog        | 8025 / 1025 | 8025 / 1025 | Email capture (Web UI / SMTP)  |
| migrate   | migrate/migrate        | --        | --             | Runs golang-migrate on startup |
| app       | Dockerfile.dev (hot-reload) | 8082 | 8080           | Go dev server with air         |

The `migrate` service runs all pending migrations automatically before `app` starts. PostgreSQL uses port **5433** on the host to avoid conflicts with any local Postgres instance.

### Stopping

```bash
make down
# or
docker-compose down
```

To also remove data volumes:

```bash
docker-compose down -v
```

### Port Conflicts

Docker containers sometimes hold onto port 8080 after a crash. If you see a bind error, either kill the stale container or use the mapped host port 8082 (which is the default in docker-compose).

---

## Production Dockerfile

The production `Dockerfile` uses a multi-stage build:

1. **Builder stage** -- `golang:1.24-alpine`
   - Downloads dependencies (`go mod download`)
   - Compiles the binary: `CGO_ENABLED=0 go build -ldflags="-s -w" -o /payflow ./cmd/server`
2. **Runner stage** -- `alpine:3.19`
   - Copies the compiled binary
   - Installs only `ca-certificates`
   - Exposes port 8080
   - Entrypoint: `./payflow`

The resulting image is minimal (no Go toolchain, no source code).

---

## Railway Deployment

### Configuration (railway.toml)

```toml
[build]
builder = "DOCKERFILE"
dockerfilePath = "./Dockerfile"

[deploy]
healthcheckPath = "/health"
healthcheckTimeout = 30
restartPolicyType = "ON_FAILURE"
restartPolicyMaxRetries = 3
```

Railway builds using the production Dockerfile and monitors the `/health` endpoint. If the health check fails within 30 seconds of deploy, Railway rolls back. Failed containers restart up to 3 times.

### Setup Steps

1. Create a new Railway project and link the GitHub repository.
2. Add a **PostgreSQL** service from the Railway dashboard.
3. In the app service, add the following environment variables (at minimum):
   - `DATABASE_URL` -- reference the Postgres service variable (e.g., `${{Postgres.DATABASE_URL}}`)
   - `JWT_SECRET` -- a strong random string, at least 32 characters
   - `KORAPAY_API_KEY` and `KORAPAY_PUBLIC_KEY` -- from your Korapay dashboard
4. Railway injects `PORT` automatically; the app reads it at startup.
5. Deploy. Railway will build the Docker image, run the health check, and route traffic.

### Auto-Migration on Railway

When `DATABASE_URL` contains `railway` or `rlwy`, the app automatically enables GORM auto-migration at startup. This is a convenience for early-stage deployments. For mature schemas, disable this by setting `ENABLE_AUTO_MIGRATION=false` and running golang-migrate manually (see [Database Migrations](#database-migrations)).

---

## Environment Variables

All configuration is loaded via [Viper](https://github.com/spf13/viper) from environment variables or a `.env` file in the project root. Environment variables take precedence over the file.

### Server

| Variable            | Default  | Description                              |
|---------------------|----------|------------------------------------------|
| `SERVER_PORT`       | `8080`   | HTTP listen port. Railway overrides this via `PORT`. |
| `LOG_LEVEL`         | `info`   | Log verbosity: debug, info, warn, error  |
| `LOG_PRETTY`        | `false`  | Human-readable log output (set `true` for local dev) |
| `CORS_ALLOWED_ORIGINS` | `https://payflowio.netlify.app` | Comma-separated list of allowed origins |

### Database

The app resolves the database DSN using a fallback chain. The first non-empty value wins:

1. `DB_URL`
2. `DATABASE_URL`
3. `DATABASE_PRIVATE_URL`
4. `DATABASE_PUBLIC_URL`
5. Built from individual vars: `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, `PGSSLMODE`

| Variable       | Default    | Description                       |
|----------------|------------|-----------------------------------|
| `PGHOST`       | --         | PostgreSQL host                   |
| `PGPORT`       | `5432`     | PostgreSQL port                   |
| `PGUSER`       | --         | PostgreSQL user                   |
| `PGPASSWORD`   | --         | PostgreSQL password               |
| `PGDATABASE`   | --         | PostgreSQL database name          |
| `PGSSLMODE`    | `require`  | SSL mode (set `disable` for local dev) |

| Variable              | Default | Description                                  |
|-----------------------|---------|----------------------------------------------|
| `ENABLE_AUTO_MIGRATION` | `false` | GORM auto-migrate on startup. Auto-enabled when DATABASE_URL contains "railway". Use `false` in production. |

### Authentication

| Variable              | Default | Description                                    |
|-----------------------|---------|------------------------------------------------|
| `JWT_SECRET`          | --      | **Required.** Server refuses to start without it. Must be a strong random string. |
| `JWT_EXPIRATION_HOURS`| `72`    | Token lifetime in hours                        |

### Korapay (Primary Payment Provider)

| Variable             | Default                      | Description              |
|----------------------|------------------------------|--------------------------|
| `KORAPAY_API_KEY`    | --                           | Secret key from Korapay  |
| `KORAPAY_PUBLIC_KEY` | --                           | Public key from Korapay  |
| `KORAPAY_BASE_URL`   | `https://api.korapay.com`    | API base URL             |

### VFD Bank (Secondary Provider, Optional)

| Variable              | Default                                      | Description               |
|-----------------------|----------------------------------------------|---------------------------|
| `VFD_CONSUMER_KEY`    | --                                           | OAuth consumer key        |
| `VFD_CONSUMER_SECRET` | --                                           | OAuth consumer secret     |
| `VFD_BASE_URL`        | `https://api-devapps.vfdbank.systems`        | API base URL              |
| `VFD_WEBHOOK_SECRET`  | --                                           | HMAC verification secret  |

VFD configuration is entirely optional. The app gracefully handles a nil VFD service.

### Transfer Configuration

| Variable                          | Default     | Description                                    |
|-----------------------------------|-------------|------------------------------------------------|
| `TRANSFER_DEFAULT_PROVIDER`       | `korapay`   | Primary disbursement provider                  |
| `TRANSFER_PROVIDER_FALLBACK_ORDER`| `vfd`       | Fallback provider(s) if primary fails          |
| `TRANSFER_MIN_AMOUNT`             | `1000`      | Minimum transfer in kobo (NGN 10)              |
| `TRANSFER_MAX_AMOUNT`             | `10000000`  | Maximum transfer in kobo (NGN 100,000)         |

All monetary values are in the smallest currency unit (kobo for NGN). For example, 500000 = NGN 5,000.

### Redis (Optional)

| Variable    | Default | Description                                       |
|-------------|---------|---------------------------------------------------|
| `REDIS_URL` | --      | Redis connection string. App degrades gracefully without it. |

---

## Database Migrations

PayFlow uses [golang-migrate](https://github.com/golang-migrate/migrate) for schema migrations. GORM auto-migrate is **not** used in production.

### Migration Files

All migration files live in `/migrations/` and follow the naming convention:

```
000001_create_users_table.up.sql
000001_create_users_table.down.sql
```

The project is currently at version **11** (`000011_add_performance_indexes`).

### Running Migrations

**Via Docker Compose** (automatic on `docker-compose up`):

The `migrate` service applies all pending up-migrations before the app starts.

**Via Make targets** (manual, requires `migrate` CLI installed):

```bash
# Set DSN for local database
export DSN="postgres://payflow_user:payflow_secret@localhost:5433/payflow_db?sslmode=disable"

# Apply all pending migrations
make migrate-up

# Revert the last migration
make migrate-down
```

**Via migrate CLI directly:**

```bash
migrate -database "$DSN" -path ./migrations up
migrate -database "$DSN" -path ./migrations down 1
```

### Creating a New Migration

```bash
make migrate-create
# Prompts for a name, then creates:
#   migrations/000012_your_name.up.sql
#   migrations/000012_your_name.down.sql
```

Write the up (apply) and down (rollback) SQL, then commit both files.

### Migration Strategy

- **Local development**: Docker Compose runs migrations automatically. You can also run `make migrate-up` manually against the host-mapped port (5433).
- **Railway**: Auto-migration via GORM is enabled by default (detected from DATABASE_URL). For more control, set `ENABLE_AUTO_MIGRATION=false` and run golang-migrate against the Railway Postgres instance using the connection string from the dashboard.
- **General rule**: Every schema change must have a corresponding migration file. Never modify a migration that has already been applied to a shared environment.

---

## Health Checks

Two endpoints are available for orchestrators and load balancers:

| Endpoint        | Purpose          | Behavior                                           |
|-----------------|------------------|----------------------------------------------------|
| `GET /health`   | Readiness probe  | Checks DB and Redis connectivity. Returns JSON with component status. |
| `GET /health/live` | Liveness probe | Returns HTTP 200 immediately. No dependency checks. |

Railway uses `GET /health` with a 30-second timeout as its readiness gate during deployments.

Health check requests are excluded from request logging to reduce noise.

---

## Troubleshooting

### Server fails to start: missing JWT_SECRET

`JWT_SECRET` is required and has no default. Set it in your `.env` file or as an environment variable:

```bash
export JWT_SECRET="your-secret-key-at-least-32-characters"
```

### Port 8080 already in use

This commonly happens when a previous Docker container did not shut down cleanly. Solutions:

```bash
# Find and kill the process
lsof -i :8080
kill -9 <PID>

# Or use the host-mapped port 8082 (default in docker-compose)
```

### Database connection refused

Ensure the Postgres container is running and healthy:

```bash
docker-compose ps db
```

If connecting from the host, use port **5433** (not 5432):

```bash
psql "postgres://payflow_user:payflow_secret@localhost:5433/payflow_db?sslmode=disable"
```

### Migrations fail with "dirty" state

If a migration partially applied and left the schema in a dirty state:

```bash
# Force the version to the last known good migration
migrate -database "$DSN" -path ./migrations force <VERSION_NUMBER>

# Then re-run
migrate -database "$DSN" -path ./migrations up
```

### Railway deploy fails health check

Check the deploy logs in the Railway dashboard. Common causes:
- `DATABASE_URL` not set or not referencing the Postgres service correctly
- `JWT_SECRET` not set
- The app crashed before it could bind to the port (check for panic in logs)

---

## Production Deployment Checklist

### Railway Setup

```
[ ] 1. Create Railway project, link GitHub repo
[ ] 2. Add PostgreSQL service
[ ] 3. Add Redis service (optional but recommended)
[ ] 4. Set environment variables (see below)
[ ] 5. Deploy — verify /health returns "healthy"
[ ] 6. Run migrations if ENABLE_AUTO_MIGRATION=false
```

### Required Environment Variables for Production

```bash
# MUST SET — server won't start without these
JWT_SECRET=<random-64-char-string>

# Database — reference Railway's Postgres service
DATABASE_URL=${{Postgres.DATABASE_URL}}

# Payment provider — at least one
PAYSTACK_SECRET_KEY=sk_live_your_paystack_live_key
ENABLED_PROVIDERS=paystack
TRANSFER_DEFAULT_PROVIDER=paystack

# Email — Brevo SMTP (free 300/day)
SMTP_HOST=smtp-relay.brevo.com
SMTP_PORT=587
SMTP_USER=your_brevo_login
SMTP_PASSWORD=your_brevo_smtp_key
SMTP_FROM=payroll@yourdomain.com

# Application URL (for email links)
APP_URL=https://payflowio.netlify.app
CORS_ALLOWED_ORIGINS=https://payflowio.netlify.app

# Redis (if added)
REDIS_URL=${{Redis.REDIS_URL}}

# Production settings
LOG_LEVEL=info
LOG_PRETTY=false
ENABLE_AUTO_MIGRATION=false
```

### Brevo SMTP Setup (Free)

1. Sign up at [brevo.com](https://www.brevo.com) (free, no credit card)
2. Go to Settings > SMTP & API
3. Create an SMTP key
4. Use these values:
   - Host: `smtp-relay.brevo.com`
   - Port: `587`
   - User: your Brevo login email
   - Password: the SMTP key you created

### Paystack Live Keys

1. Log into [dashboard.paystack.com](https://dashboard.paystack.com)
2. Toggle from Test to Live mode
3. Copy your live secret key (`sk_live_...`)
4. Set `PAYSTACK_SECRET_KEY` in Railway

### Post-Deploy Verification

```bash
# Health check
curl https://your-railway-url.up.railway.app/health

# Expected response:
# {"status":"healthy","message":"Server is running. All systems operational.","server":"ok","database":{"status":"ok","message":"connected"},"redis":{"status":"ok","message":"connected"}}

# Test login
curl -X POST https://your-railway-url.up.railway.app/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@yourcompany.com","password":"yourpassword"}'
```

### Database Backup (Railway)

Railway provides automated daily backups for PostgreSQL. To manually backup:

```bash
# From Railway CLI
railway run pg_dump $DATABASE_URL > backup_$(date +%Y%m%d).sql

# Restore
railway run psql $DATABASE_URL < backup_20260528.sql
```
