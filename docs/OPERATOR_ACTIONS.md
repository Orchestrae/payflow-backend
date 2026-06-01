# PayFlow Operator Actions

Complete reference for environment variables, webhook registration, post-deployment checklist, and ongoing operations.

---

## Environment Variables

All configuration is read from environment variables (or `.env` file for local development). The application uses Viper with `AutomaticEnv()`.

### Core / Required

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DB_URL` | Yes* | (none) | PostgreSQL connection string. Falls back to `DATABASE_URL`, `DATABASE_PRIVATE_URL`, `DATABASE_PUBLIC_URL`, then individual `PG*` vars. |
| `JWT_SECRET` | Yes | (none) | Secret key for signing JWT tokens. Minimum 32 characters. First 32 bytes also used as AES-256 encryption key for platform settings. **Server will not start without this.** |
| `JWT_EXPIRATION_HOURS` | No | `72` | Token expiration in hours. |
| `SERVER_PORT` | No | `8080` | HTTP server port. Overridden by `PORT` env var (Railway/Heroku inject this). |

*At least one database URL source must be available.

### Redis

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `REDIS_URL` | No | (none) | Redis connection string (e.g., `redis://default:password@host:6379`). If not set, caching is disabled, emails are sent synchronously, and the scheduler uses in-memory gocron (jobs lost on restart). |

### Payment Providers -- KoraPay

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `KORAPAY_API_KEY` | No | (none) | KoraPay secret/API key (e.g., `sk_live_...` or `sk_test_...`). Required for virtual accounts, KoraPay transfers, and deposit webhooks. |
| `KORAPAY_PUBLIC_KEY` | No | (none) | KoraPay public key (e.g., `pk_live_...`). Used for client-side integrations. |
| `KORAPAY_BASE_URL` | No | `https://api.korapay.com` | KoraPay API base URL. |

### Payment Providers -- Paystack

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PAYSTACK_SECRET_KEY` | No | (none) | Paystack secret key (e.g., `sk_live_...`). Required for Paystack transfers, bank verification, deposits, and billing/subscriptions. |
| `PAYSTACK_BASE_URL` | No | `https://api.paystack.co` | Paystack API base URL. |

### Payment Providers -- VFD Bank

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `VFD_CONSUMER_KEY` | No | (none) | VFD Bank OAuth consumer key. Required for VFD transfers and corporate account creation. |
| `VFD_CONSUMER_SECRET` | No | (none) | VFD Bank OAuth consumer secret. |
| `VFD_BASE_URL` | No | `https://api-devapps.vfdbank.systems` | VFD Bank API base URL. Change to production URL for live. |
| `VFD_WEBHOOK_SECRET` | No | (none) | Secret for verifying VFD webhook signatures. |

### Transfer Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ENABLED_PROVIDERS` | No | `korapay,vfd` | Comma-separated list of enabled transfer providers. Options: `korapay`, `paystack`, `vfd`. A provider is only active if both enabled AND its API key is set. |
| `TRANSFER_DEFAULT_PROVIDER` | No | `korapay` | Primary provider for transfers. Must match one of the enabled providers. |
| `TRANSFER_PROVIDER_FALLBACK_ORDER` | No | `vfd` | Comma-separated fallback order if the default provider fails. |
| `TRANSFER_MIN_AMOUNT` | No | `1000` | Minimum transfer amount in kobo (NGN 10.00). |
| `TRANSFER_MAX_AMOUNT` | No | `10000000` | Maximum transfer amount in kobo (NGN 100,000.00). |

### Email / SMTP

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SMTP_HOST` | No | `localhost` | SMTP server hostname. Use `smtp-relay.brevo.com` for Brevo, `smtp.sendgrid.net` for SendGrid. |
| `SMTP_PORT` | No | `1025` | SMTP port. Typically `587` for TLS, `1025` for local MailHog. |
| `SMTP_USER` | No | (none) | SMTP username/login. |
| `SMTP_PASSWORD` | No | (none) | SMTP password or API key. |
| `SMTP_FROM` | No | `no-reply@payflow.com` | Sender email address for outgoing emails. |
| `APP_URL` | No | `http://localhost:3000` | Frontend application URL. Used in email templates for links (password reset, invitations, etc.). |

### CORS

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CORS_ALLOWED_ORIGINS` | No | `https://payflowio.vercel.app,https://payflowio.netlify.app` | Comma-separated list of allowed CORS origins. Must include the frontend URL with protocol. |

### SMS (Termii)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TERMII_API_KEY` | No | (none) | Termii API key for SMS notifications. |
| `TERMII_SENDER_ID` | No | (none) | Termii sender ID. |

### Database Advanced

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_READ_URL` | No | (none) | Optional read replica connection string. If set, ledger and report queries are routed here. Falls back to primary DB on connection failure. |
| `ENABLE_AUTO_MIGRATION` | No | `false` | Enable GORM auto-migration on startup. Auto-enabled when DB URL contains `railway` or `rlwy`. Set to `false` for production (use golang-migrate instead). |

### Logging

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LOG_LEVEL` | No | `info` | Log level: `debug`, `info`, `warn`, `error`, `fatal`. |
| `LOG_PRETTY` | No | `false` | Enable human-readable console log output (local dev only). |

---

## What Breaks If Missing

| Missing Variable | Impact |
|-----------------|--------|
| `DB_URL` / `DATABASE_URL` + no `PG*` vars | **Server crashes on startup.** Cannot connect to database. |
| `JWT_SECRET` | **Server crashes on startup.** Explicit fatal log message. |
| `REDIS_URL` | Degraded mode: no caching, synchronous emails (no retry), in-memory job scheduler (jobs lost on restart). |
| `KORAPAY_API_KEY` | No virtual accounts, no KoraPay transfers, deposit webhooks still accepted but may fail processing. |
| `PAYSTACK_SECRET_KEY` | No Paystack transfers, no bank/BVN verification, no deposits via Paystack, no billing/subscription payments. |
| `VFD_CONSUMER_KEY` / `VFD_CONSUMER_SECRET` | No VFD transfers, no corporate account creation on registration. |
| All provider keys missing | **Transfers completely disabled.** Warning logged on startup. |
| `SMTP_HOST` / `SMTP_PASSWORD` | Invitation emails, password reset emails, and payroll notifications silently fail. |
| `APP_URL` | Email links point to `http://localhost:3000` (wrong in production). |
| `CORS_ALLOWED_ORIGINS` | Frontend requests blocked by CORS. Falls back to Vercel/Netlify URLs. |

---

## Webhook URLs to Register

After deploying the backend, register these webhook URLs with each payment provider.

### Paystack

**Dashboard:** [dashboard.paystack.com](https://dashboard.paystack.com) > Settings > API Keys & Webhooks

| Purpose | URL |
|---------|-----|
| Transfer status updates | `POST {BASE_URL}/paystack/webhooks/` |
| Billing/subscription events | `POST {BASE_URL}/paystack/webhooks/billing` |

Events to enable: `transfer.success`, `transfer.failed`, `transfer.reversed`, `charge.success`, `subscription.create`, `subscription.disable`, `invoice.payment_failed`.

**Webhook verification:** Paystack sends an `x-paystack-signature` header with HMAC-SHA512 of the body using your secret key. PayFlow verifies this automatically.

### KoraPay

**Dashboard:** [merchant.korapay.com](https://merchant.korapay.com) > Settings > Webhooks

| Purpose | URL |
|---------|-----|
| Virtual account deposits | `POST {BASE_URL}/korapay/webhooks/deposit` |

Events to enable: `charge.completed`, `virtual_bank_account.transfer.completed`.

### VFD Bank

Contact VFD Bank support to register webhook URLs.

| Purpose | URL |
|---------|-----|
| Inward credit notifications | `POST {BASE_URL}/vfd/webhooks/inward-credit` |

---

## Post-Deployment Checklist

### First-Time Deployment

- [ ] **Database provisioned** -- PostgreSQL 16+ instance running and accessible
- [ ] **Redis provisioned** -- Redis 7+ instance running (recommended but optional)
- [ ] **Environment variables set** -- At minimum: `JWT_SECRET`, database connection
- [ ] **Migrations applied** -- Run `migrate -database "$DATABASE_URL" -path ./migrations up` or set `ENABLE_AUTO_MIGRATION=true`
- [ ] **Health check passing** -- `GET /health` returns `{"status":"healthy"}`
- [ ] **CORS configured** -- `CORS_ALLOWED_ORIGINS` includes the frontend URL
- [ ] **Frontend deployed** -- Vercel deployment with `VITE_API_URL` pointing to backend
- [ ] **Frontend routing works** -- SPA rewrite configured in `vercel.json`
- [ ] **SMTP configured** -- Test by registering a new business (should receive verification email)
- [ ] **At least one payment provider configured** -- Set API keys for KoraPay, Paystack, or VFD
- [ ] **Webhook URLs registered** -- Register with each active payment provider (see above)
- [ ] **Test webhook delivery** -- Make a test deposit/transfer and verify webhook is received
- [ ] **SSL/TLS active** -- Railway and Vercel handle this automatically
- [ ] **Custom domain configured** -- Optional but recommended for production
- [ ] **Subscription plans seeded** -- Create billing plans in the database if using billing features
- [ ] **Super admin created** -- Manually set a user's role to `super_admin` in the database for platform admin access

### After Each Backend Deploy

- [ ] **Health check passes** -- `GET /health` returns healthy
- [ ] **Migrations applied** -- If new migrations were added, run them before or during deploy
- [ ] **Test critical flows** -- Register, login, create employee, run payroll
- [ ] **Check background jobs** -- Verify daily reconciliation runs (check logs after 1 minute)
- [ ] **Monitor error logs** -- Watch for unexpected 500s in the first hour

---

## Ongoing Operations

### Daily

- **Monitor logs** for errors (5xx responses, provider failures, reconciliation issues).
- **Check health endpoint** -- automated uptime monitoring recommended (UptimeRobot, Better Uptime).

### Weekly

- **Review reconciliation results.** The system runs automatic daily balance reconciliation and weekly provider reconciliation. Check logs for discrepancies.
- **Check transfer failure rate.** High failure rates may indicate provider issues or insufficient funds.

### Monthly

- **Review audit logs.** `GET /v1/audit-logs` for each business, or query the `audit_logs` table directly.
- **Check subscription renewals.** Paystack handles automatic billing, but monitor for payment failures.
- **Update provider API keys** if rotating keys.

### As Needed

- **Seed subscription plans:** Insert rows into the `subscription_plans` table for Free, Growth, and Enterprise tiers.
- **Create super admin:** Update a user row: `UPDATE users SET role = 'super_admin' WHERE email = 'admin@payflow.io';`
- **Force reconciliation:** `GET /platform/reconciliation/provider` (requires super_admin token).
- **Suspend a business:** `POST /platform/organizations/{id}/suspend` with reason.
- **Rotate JWT_SECRET:** Update the env var and restart. All existing tokens will be invalidated -- users must re-login.
- **Add new migrations:** `make migrate-create`, write SQL, test locally, then `make migrate-up` in production.
- **Scale horizontally:** PayFlow is stateless (session in JWT, jobs in Redis). Add more Railway instances behind a load balancer. Ensure all instances share the same `JWT_SECRET` and database.

### Database Maintenance

```bash
# Check current migration version
migrate -database "$DATABASE_URL" -path ./migrations version

# Apply pending migrations
migrate -database "$DATABASE_URL" -path ./migrations up

# Rollback last migration
migrate -database "$DATABASE_URL" -path ./migrations down 1

# Force version (use when migration is stuck in dirty state)
migrate -database "$DATABASE_URL" -path ./migrations force 27
```

### Backup Strategy

- **Database:** Railway provides daily automatic backups for Pro plans. For custom backups: `pg_dump -Fc $DATABASE_URL > backup.dump`
- **Redis:** Data is ephemeral (cache + job queue). No backup needed -- state rebuilds on restart.
