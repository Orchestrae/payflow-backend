# PayFlow Deployment Guide

## Overview

PayFlow is deployed as two services:
- **Backend:** Go API server on Railway
- **Frontend:** React SPA on Vercel

Both services connect to a Railway-hosted PostgreSQL database and Redis instance.

---

## Railway Backend Setup

### 1. Create Railway Project

1. Go to [railway.app](https://railway.app) and create a new project.
2. Add a **PostgreSQL** service from the Railway marketplace.
3. Add a **Redis** service from the Railway marketplace.
4. Add a new service from your GitHub repo (the Go backend).

### 2. Build Configuration

| Setting | Value |
|---------|-------|
| Root Directory | `/` (project root) |
| Build Command | `go build -o bin/payflow ./cmd/server/main.go` |
| Start Command | `./bin/payflow` |
| Health Check Path | `/health` |
| Health Check Timeout | `30s` |

Railway auto-detects Go projects. The `Dockerfile` is not required -- Railway uses Nixpacks.

### 3. Environment Variables

Set these in the Railway service settings. See [OPERATOR_ACTIONS.md](OPERATOR_ACTIONS.md) for the complete reference.

**Required:**
```
JWT_SECRET=<generate a random 64+ char string>
```

**Auto-injected by Railway (if Postgres/Redis are linked):**
```
DATABASE_URL=<auto-injected by Railway Postgres plugin>
REDIS_URL=<auto-injected by Railway Redis plugin>
```

Railway also injects `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE` which PayFlow uses as fallback if `DB_URL`/`DATABASE_URL` are not set.

**Payment Providers (at least one required for transfers):**
```
KORAPAY_API_KEY=sk_live_...
KORAPAY_PUBLIC_KEY=pk_live_...
PAYSTACK_SECRET_KEY=sk_live_...
```

**Email (required for invitations, password resets, payslip delivery):**
```
SMTP_HOST=smtp-relay.brevo.com
SMTP_PORT=587
SMTP_USER=your-brevo-login
SMTP_PASSWORD=your-smtp-password
SMTP_FROM=no-reply@payflow.io
APP_URL=https://app.payflow.io
```

**CORS (required for frontend access):**
```
CORS_ALLOWED_ORIGINS=https://app.payflow.io,https://payflowio.vercel.app
```

### 4. Database Migrations

PayFlow supports two migration modes:

**Option A: Auto-migration (Railway default)**
When the database URL contains `railway` or `rlwy`, auto-migration is enabled automatically. GORM will create/update tables on startup. This is convenient but not recommended for production.

**Option B: Traditional migrations (recommended for production)**
```bash
# Set ENABLE_AUTO_MIGRATION=false in Railway env vars
# Then run migrations manually:

# Install golang-migrate
brew install golang-migrate  # macOS
# or: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Apply migrations
migrate -database "$DATABASE_URL" -path ./migrations up

# Check current version
migrate -database "$DATABASE_URL" -path ./migrations version

# Rollback one migration
migrate -database "$DATABASE_URL" -path ./migrations down 1
```

Current migration version: **000027** (payroll job tracking).

### 5. Health Check Verification

After deployment, verify:
```bash
curl https://your-app.railway.app/health
```

Expected response:
```json
{
  "status": "healthy",
  "server": "ok",
  "database": { "status": "ok", "message": "connected" },
  "redis": { "status": "ok", "message": "connected" }
}
```

---

## Vercel Frontend Setup

### 1. Import Project

1. Go to [vercel.com](https://vercel.com) and import the repository.
2. Set the **Root Directory** to `frontend`.

### 2. Build Configuration

| Setting | Value |
|---------|-------|
| Framework Preset | Vite |
| Root Directory | `frontend` |
| Build Command | `npm run build` |
| Output Directory | `dist` |
| Install Command | `npm install` |

### 3. Environment Variables

```
VITE_API_URL=https://your-app.railway.app
```

### 4. SPA Routing

Vercel needs a rewrite rule for client-side routing. Create `frontend/vercel.json`:
```json
{
  "rewrites": [{ "source": "/(.*)", "destination": "/" }]
}
```

---

## Redis / Asynq Setup

Redis powers three features:
1. **Caching:** Cadre and deduction rule caches for faster payroll computation.
2. **Job Queue:** Asynq-based async email delivery with retry logic.
3. **Scheduled Jobs:** Payroll processing scheduler (persistent across restarts).

If Redis is not available, PayFlow degrades gracefully:
- Cache misses fall through to the database.
- Emails are sent synchronously (no retry).
- Scheduler falls back to in-memory gocron (jobs lost on restart).

**Railway:** Add the Redis plugin and link it to your backend service. The `REDIS_URL` variable is injected automatically.

---

## Domain / DNS Setup

### Custom Domain for Backend (Railway)
1. In Railway, go to your backend service > Settings > Networking.
2. Add your custom domain (e.g., `api.payflow.io`).
3. Add a CNAME record in your DNS provider pointing to the Railway-provided hostname.

### Custom Domain for Frontend (Vercel)
1. In Vercel, go to your project > Settings > Domains.
2. Add your custom domain (e.g., `app.payflow.io`).
3. Add the DNS records Vercel provides (A record or CNAME).

### Webhook URLs
After setting up your custom domain, register these webhook URLs with payment providers:

| Provider | URL | Dashboard Location |
|----------|-----|--------------------|
| Paystack (transfers) | `https://api.payflow.io/paystack/webhooks/` | Settings > API Keys & Webhooks |
| Paystack (billing) | `https://api.payflow.io/paystack/webhooks/billing` | Settings > API Keys & Webhooks |
| KoraPay (deposits) | `https://api.payflow.io/korapay/webhooks/deposit` | Settings > Webhooks |
| VFD Bank (inward credit) | `https://api.payflow.io/vfd/webhooks/inward-credit` | Contact VFD support |

---

## Monitoring

### Health Checks
- **Readiness:** `GET /health` - returns 200 when DB + Redis are reachable, 503 otherwise.
- **Liveness:** `GET /health/live` - returns 200 if the process is running.

### Logs
Railway provides built-in log streaming. PayFlow uses structured JSON logging (zerolog).

Set `LOG_LEVEL=debug` for verbose output during troubleshooting, `LOG_LEVEL=info` for production.

Set `LOG_PRETTY=true` for human-readable console output (local development only).

### Background Jobs
Two background jobs run automatically:
1. **Daily Reconciliation:** Runs 1 minute after startup, then every 24 hours. Verifies wallet balances match ledger totals.
2. **Weekly Provider Reconciliation:** Runs 5 minutes after startup, then every 7 days. Cross-checks transfer records with payment provider APIs.

---

## Troubleshooting

### "JWT_SECRET is required" on startup
Ensure `JWT_SECRET` is set in Railway environment variables. It must be a non-empty string (minimum 32 characters recommended).

### Database connection fails
Check that PostgreSQL is linked to the backend service in Railway. Verify with:
```bash
railway run printenv | grep -E "DATABASE_URL|PGHOST"
```

### Migrations stuck (dirty state)
```bash
# Force the version to the last known good migration
migrate -database "$DATABASE_URL" -path ./migrations force <version_number>
# Then re-run
migrate -database "$DATABASE_URL" -path ./migrations up
```

### CORS errors from frontend
Ensure `CORS_ALLOWED_ORIGINS` includes the exact frontend URL (including protocol, no trailing slash).

### Transfers failing
1. Check that at least one provider is enabled: `ENABLED_PROVIDERS=korapay,paystack,vfd`
2. Verify provider API keys are set and valid.
3. Check that `TRANSFER_DEFAULT_PROVIDER` matches an enabled provider.
