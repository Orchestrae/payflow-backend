# PayFlow Operator's Manual

This document is the definitive operations guide for deploying, configuring, and maintaining a PayFlow instance. It assumes zero prior context. Every environment variable, webhook URL, migration step, and operational procedure is listed here.

---

## 1. Environment Variables (Complete List)

All variables are read via Viper from either a `.env` file in the project root or from OS environment variables. Environment variables always override `.env` file values.

### Server / General

| Variable | Required | Format | Default | Description | What breaks if missing |
|---|---|---|---|---|---|
| `SERVER_PORT` | Optional | Integer as string | `8080` | HTTP listen port. PaaS platforms (Railway, Heroku, Render) inject `PORT` at runtime, which overrides this. | Falls back to `8080`. |
| `LOG_LEVEL` | Optional | `debug`, `info`, `warn`, `error`, `fatal` | `info` | Zerolog log level. | Falls back to `info`. |
| `LOG_PRETTY` | Optional | `true` / `false` | `false` | Human-readable console log output. Use only in development. | Falls back to JSON log format. |
| `APP_URL` | Optional | Full URL | `http://localhost:3000` | Frontend application URL. Used in email links (password reset, email verification, billing). | Email links will point to localhost. |

### Database

| Variable | Required | Format | Default | Description | What breaks if missing |
|---|---|---|---|---|---|
| `DB_URL` | **Required** | Postgres connection string | None | Primary database URL. Example: `postgres://user:pass@host:5432/payflow?sslmode=require` | **Server will not start.** Falls back to checking `DATABASE_URL`, `DATABASE_PRIVATE_URL`, `DATABASE_PUBLIC_URL`, then individual `PG*` vars (see below). |
| `DATABASE_URL` | Optional | Postgres connection string | None | Fallback if `DB_URL` is empty. Common on Heroku/Render. | Used only as fallback. |
| `DATABASE_PRIVATE_URL` | Optional | Postgres connection string | None | Fallback if both above are empty. Common on Railway. | Used only as fallback. |
| `DATABASE_PUBLIC_URL` | Optional | Postgres connection string | None | Last-resort fallback. | Used only as fallback. |
| `PGHOST` | Optional | Hostname | None | Individual Postgres host. Used to build DSN if no connection string is set. | Ignored if a full connection string is set. |
| `PGPORT` | Optional | Integer as string | `5432` | Individual Postgres port. | Falls back to `5432`. |
| `PGUSER` | Optional | String | None | Postgres user. Required if building DSN from individual vars. | DSN build fails. |
| `PGPASSWORD` | Optional | String | None | Postgres password. | DSN build fails. |
| `PGDATABASE` | Optional | String | None | Postgres database name. Required if building DSN from individual vars. | DSN build fails. |
| `PGSSLMODE` | Optional | `require`, `disable`, `verify-full` | `require` | SSL mode for individual-var DSN. | Falls back to `require`. |
| `DATABASE_READ_URL` | Optional | Postgres connection string | None | Read replica for ledger/report queries. Falls back to primary DB if not set. | All reads go to primary DB (higher load). |
| `ENABLE_AUTO_MIGRATION` | Optional | `true` / `false` | `false` | When `true`, runs GORM AutoMigrate on startup. **Never enable in production.** Auto-enabled if the DB URL contains `railway` or `rlwy`. | Falls back to `false` (traditional migrations only). |

### Authentication

| Variable | Required | Format | Default | Description | What breaks if missing |
|---|---|---|---|---|---|
| `JWT_SECRET` | **Required** | String, min 32 characters | None | Secret key for signing JWTs. The first 32 bytes are also used as the AES-256-GCM encryption key for platform settings. | **Server will not start.** Fatal error on boot: `JWT_SECRET is required`. |
| `JWT_EXPIRATION_HOURS` | Optional | Integer | `72` | JWT token lifetime in hours. | Falls back to 72 hours. |

### Payment Providers - Paystack

| Variable | Required | Format | Default | Description | What breaks if missing |
|---|---|---|---|---|---|
| `PAYSTACK_SECRET_KEY` | Optional* | `sk_test_...` or `sk_live_...` | None | Paystack secret key. Required for: transfers via Paystack, bank account verification, billing/subscriptions, provider reconciliation. | Paystack transfer provider disabled. Bank account verification disabled. Billing payment collection disabled. |
| `PAYSTACK_BASE_URL` | Optional | URL | `https://api.paystack.co` | Paystack API base URL. No reason to change this. | Falls back to default. |

### Payment Providers - Korapay

| Variable | Required | Format | Default | Description | What breaks if missing |
|---|---|---|---|---|---|
| `KORAPAY_API_KEY` | Optional* | String | None | Korapay secret API key. Required for: transfers via Korapay, virtual account creation, wallet deposits. | Korapay transfer provider disabled. Virtual account creation fails. Wallet deposits fail. |
| `KORAPAY_PUBLIC_KEY` | Optional | String | None | Korapay public key. Used for client-side deposit initialization. | Client-side deposit flows break. |
| `KORAPAY_BASE_URL` | Optional | URL | `https://api.korapay.com` | Korapay API base URL. | Falls back to default. |

### Payment Providers - VFD Bank

| Variable | Required | Format | Default | Description | What breaks if missing |
|---|---|---|---|---|---|
| `VFD_CONSUMER_KEY` | Optional* | String | None | VFD MicroFinance Bank consumer key. Required for VFD transfers and account management. | VFD transfer provider disabled. |
| `VFD_CONSUMER_SECRET` | Optional* | String | None | VFD consumer secret. | VFD transfers fail. |
| `VFD_BASE_URL` | Optional | URL | `https://api-devapps.vfdbank.systems` | VFD API base URL. Change to production URL (`https://api-apps.vfdbank.systems`) for live. | Falls back to dev/sandbox URL. |
| `VFD_WEBHOOK_SECRET` | Optional | String | None | Secret for verifying VFD inbound webhook signatures. | Webhook signature verification skipped (insecure). |

> *At least one payment provider (Paystack, Korapay, or VFD) must have valid credentials, otherwise all transfer operations will fail. The server will start but log a warning: "No transfer providers enabled -- transfers will fail".

### Transfer Configuration

| Variable | Required | Format | Default | Description | What breaks if missing |
|---|---|---|---|---|---|
| `TRANSFER_DEFAULT_PROVIDER` | Optional | `korapay`, `vfd`, `paystack` | `korapay` | Primary transfer provider. Must be in `ENABLED_PROVIDERS` and have valid credentials. | Falls back to `korapay`. |
| `TRANSFER_PROVIDER_FALLBACK_ORDER` | Optional | Comma-separated | `vfd` | Fallback provider(s) if the default fails. | No fallback on failure. |
| `TRANSFER_MIN_AMOUNT` | Optional | Integer (kobo) | `1000` | Minimum transfer amount in minor currency units. 1000 kobo = NGN 10.00. | Falls back to NGN 10.00. |
| `TRANSFER_MAX_AMOUNT` | Optional | Integer (kobo) | `10000000` | Maximum transfer amount. 10000000 kobo = NGN 100,000.00. | Falls back to NGN 100,000.00. |
| `ENABLED_PROVIDERS` | Optional | Comma-separated | `korapay,vfd` | Which transfer providers to activate. Only providers listed here AND with valid credentials are enabled. Values: `korapay`, `vfd`, `paystack`. | Falls back to `korapay,vfd`. |

### Email / SMTP

| Variable | Required | Format | Default | Description | What breaks if missing |
|---|---|---|---|---|---|
| `SMTP_HOST` | Optional | Hostname | `localhost` | SMTP server. Use `smtp-relay.brevo.com` for Brevo (production). | Falls back to localhost (MailHog for dev). |
| `SMTP_PORT` | Optional | Integer as string | `1025` | SMTP port. Brevo uses `587`. | Falls back to MailHog port. |
| `SMTP_USER` | Optional | String | None | SMTP username. For Brevo, this is your Brevo login email. | Email sending fails silently. |
| `SMTP_PASSWORD` | Optional | String | None | SMTP password / API key. For Brevo, use an SMTP key (not the main API key). | Email sending fails. |
| `SMTP_FROM` | Optional | Email address | `no-reply@payflow.com` | Sender "From" address. Must be verified with your SMTP provider. | Falls back to default (may be rejected by provider). |

### Redis

| Variable | Required | Format | Default | Description | What breaks if missing |
|---|---|---|---|---|---|
| `REDIS_URL` | Optional | Redis connection string | None | Example: `redis://default:password@host:6379`. Used for: Asynq job queue, email queue with retry, persistent scheduler, caching. | Server runs without cache. Scheduler uses in-memory gocron (jobs lost on restart). Emails sent synchronously (no retry). Warning logged. |

### CORS

| Variable | Required | Format | Default | Description | What breaks if missing |
|---|---|---|---|---|---|
| `CORS_ALLOWED_ORIGINS` | Optional | Comma-separated URLs | `https://payflowio.vercel.app,https://payflowio.netlify.app` | Allowed CORS origins. Add your frontend domain(s) here. | Falls back to production defaults. Custom frontend domains will be blocked by CORS. |

### SMS (Termii)

| Variable | Required | Format | Default | Description | What breaks if missing |
|---|---|---|---|---|---|
| `TERMII_API_KEY` | Optional | String | None | Termii API key for SMS notifications. | SMS notifications disabled. |
| `TERMII_SENDER_ID` | Optional | String | None | Termii sender ID (alphanumeric, max 11 chars). | SMS sending fails. |

---

## 2. Third-Party Setup (Step by Step)

### 2.1 Paystack

1. **Create account** at [dashboard.paystack.com](https://dashboard.paystack.com).
2. Navigate to Settings > API Keys & Webhooks.
3. Copy your **Secret Key** (`sk_test_...` for test, `sk_live_...` for production).
4. Set `PAYSTACK_SECRET_KEY` in your environment.
5. **Register webhook URLs** in Paystack dashboard:
   - **Transfer status updates:** `POST {BASE_URL}/paystack/webhooks/`  (trailing slash required)
   - **Billing/subscription events:** `POST {BASE_URL}/paystack/webhooks/billing`
6. Select events to listen for: `transfer.success`, `transfer.failed`, `transfer.reversed`, `charge.success`, `subscription.create`, `subscription.disable`, `invoice.payment_failed`.
7. Add `paystack` to your `ENABLED_PROVIDERS` list if using Paystack for transfers.
8. Paystack verifies webhooks using HMAC-SHA512 of the request body with your secret key via the `x-paystack-signature` header. PayFlow validates this automatically.

### 2.2 Korapay

1. **Create account** at [merchant.korapay.com](https://merchant.korapay.com).
2. Navigate to Settings > API Keys.
3. Copy your **Secret Key** and **Public Key**.
4. Set `KORAPAY_API_KEY` (secret) and `KORAPAY_PUBLIC_KEY` in your environment.
5. **Register webhook URL** in Korapay dashboard:
   - **Deposit/charge notifications:** `POST {BASE_URL}/korapay/webhooks/deposit`
6. Select events: `charge.completed`, `charge.failed`, `transfer.success`, `transfer.failed`, `virtual_bank_account.transfer.completed`.
7. Korapay is in `ENABLED_PROVIDERS` by default (`korapay,vfd`).

### 2.3 VFD MicroFinance Bank

1. **Apply for API access** with VFD Bank. You will receive a consumer key and secret.
2. Set environment variables:
   - `VFD_CONSUMER_KEY` -- your consumer key
   - `VFD_CONSUMER_SECRET` -- your consumer secret
   - `VFD_BASE_URL` -- use `https://api-devapps.vfdbank.systems` for sandbox, `https://api-apps.vfdbank.systems` for production
   - `VFD_WEBHOOK_SECRET` -- a shared secret you agree on with VFD for webhook signature verification
3. **Register webhook URLs** with VFD (done through VFD support, not a self-service dashboard):
   - **Inward credit notifications:** `POST {BASE_URL}/vfd/webhooks/inward-credit`
   - **Initial inward credit:** `POST {BASE_URL}/vfd/webhooks/initial-inward-credit`
4. VFD is in `ENABLED_PROVIDERS` by default.

### 2.4 Brevo SMTP (Email)

1. **Create account** at [app.brevo.com](https://app.brevo.com).
2. Navigate to Settings > SMTP & API.
3. Generate an **SMTP key** (this is different from the REST API key).
4. Verify your sender email address or domain under Settings > Senders & Domains.
5. Set environment variables:
   ```
   SMTP_HOST=smtp-relay.brevo.com
   SMTP_PORT=587
   SMTP_USER=your-brevo-login-email@example.com
   SMTP_PASSWORD=xsmtpsib-your-brevo-smtp-key
   SMTP_FROM=no-reply@yourdomain.com
   ```
6. The `SMTP_FROM` address must be verified in Brevo (either the specific address or the entire domain via DNS records).
7. Test by registering a new user -- they should receive a verification email.

### 2.5 Termii (SMS - Optional)

1. **Create account** at [termii.com](https://termii.com).
2. Get your API key from the dashboard.
3. Register a sender ID (alphanumeric, max 11 characters).
4. Set `TERMII_API_KEY` and `TERMII_SENDER_ID`.

---

## 3. Database Setup

### 3.1 Prerequisites

- PostgreSQL 14+ (16 recommended).
- Install `golang-migrate` CLI:
  ```bash
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```

### 3.2 Running Migrations

**Using Make (recommended for local development):**

```bash
# Apply all pending migrations (uses default local DSN)
make migrate-up

# Revert the last migration
make migrate-down

# Check current migration version
make migrate-status

# Create a new migration
make migrate-create
# (prompts for migration name)
```

The Makefile default DSN (override with `DSN=...`):
```
postgres://payflow_user:payflow_secret@localhost:5433/payflow_db?sslmode=disable
```

**Using migrate CLI directly (production):**

```bash
# Apply all migrations
migrate -database "postgres://user:pass@host:5432/payflow?sslmode=require" -path ./migrations up

# Revert last migration
migrate -database "postgres://user:pass@host:5432/payflow?sslmode=require" -path ./migrations down 1

# Check current version
migrate -database "postgres://user:pass@host:5432/payflow?sslmode=require" -path ./migrations version

# Force version (use ONLY when migration is stuck in dirty state)
migrate -database "postgres://user:pass@host:5432/payflow?sslmode=require" -path ./migrations force 27
```

### 3.3 Current Migration Version

The latest migration is **000027** (`add_payroll_job_tracking`). After running `migrate up`, verify with `migrate version` that you are at version 27 with no dirty state.

### 3.4 Seed Data (Created Automatically by Migrations)

Migration `000019_add_platform_billing.up.sql` seeds three default subscription plans:

| Plan | Tier | Monthly Price (kobo) | Monthly Price (NGN) | Max Employees | Max Payroll Runs | Features |
|---|---|---|---|---|---|---|
| Free | `free` | 0 | NGN 0 | 5 | 2 | payroll, PAYE |
| Starter | `starter` | 1,500,000 | NGN 15,000 | 50 | Unlimited (0) | payroll, PAYE, pension, NHF, reports, CSV import |
| Pro | `pro` | 5,000,000 | NGN 50,000 | Unlimited (0) | Unlimited (0) | All Starter + API access, priority support |

These are inserted with `ON CONFLICT (tier) DO NOTHING`, so re-running the migration is safe.

### 3.5 Creating the First Super Admin

There is no CLI command or migration for this. You must insert/update directly in the database.

**Option A: Register first, then promote**

1. Register a business via `POST /v1/auth/register` (this creates a user with `admin` role).
2. Promote that user to super admin:
   ```sql
   UPDATE users SET role = 'super_admin' WHERE email = 'admin@yourcompany.com';
   ```

**Option B: Promote an existing user**

```sql
UPDATE users SET role = 'super_admin' WHERE email = 'admin@yourcompany.com';
```

The `super_admin` role grants access to all `/platform/*` routes: platform stats, organization management (suspend/activate), platform settings (encrypted API key storage), and provider reconciliation triggers.

---

## 4. Webhook URLs to Register

These are the exact endpoints third-party services must call. Replace `{BASE_URL}` with your deployed server URL (e.g., `https://api.payflow.com`). All webhook endpoints are **public** (no JWT required) -- they authenticate via provider-specific signature verification.

| Provider | Purpose | Method | Exact URL | Notes |
|---|---|---|---|---|
| **Paystack** | Transfer status (success/failed/reversed) | `POST` | `{BASE_URL}/paystack/webhooks/` | Trailing slash required. Verified via `x-paystack-signature` HMAC-SHA512 header using `PAYSTACK_SECRET_KEY`. |
| **Paystack** | Billing/subscription events | `POST` | `{BASE_URL}/paystack/webhooks/billing` | Same signature verification as above. |
| **Korapay** | Deposit/charge completed | `POST` | `{BASE_URL}/korapay/webhooks/deposit` | |
| **VFD** | Inward credit notification | `POST` | `{BASE_URL}/vfd/webhooks/inward-credit` | Verified with `VFD_WEBHOOK_SECRET`. |
| **VFD** | Initial inward credit | `POST` | `{BASE_URL}/vfd/webhooks/initial-inward-credit` | Verified with `VFD_WEBHOOK_SECRET`. |

Additionally, there is an internal retrigger endpoint:

| Purpose | Method | URL | Notes |
|---|---|---|---|
| VFD webhook retrigger (replay missed webhooks) | `POST` | `{BASE_URL}/vfd/webhooks/retrigger` | Public endpoint. No auth. |

---

## 5. Post-Deployment Checklist

### Infrastructure

- [ ] PostgreSQL database provisioned and accessible
- [ ] `DB_URL` (or equivalent) environment variable set and tested with `psql`
- [ ] All migrations applied: `migrate -database "$DB_URL" -path ./migrations up` -- verify version 27, no dirty state
- [ ] Redis provisioned and `REDIS_URL` set (strongly recommended for production)
- [ ] Server port configured (`SERVER_PORT` or PaaS `PORT` injection)
- [ ] `ENABLE_AUTO_MIGRATION` is `false` in production

### Authentication and Security

- [ ] `JWT_SECRET` set -- minimum 32 characters, cryptographically random (e.g., `openssl rand -base64 48`)
- [ ] Confirm `JWT_SECRET` is at least 32 bytes (first 32 bytes used as AES-256-GCM key for platform settings encryption)
- [ ] `CORS_ALLOWED_ORIGINS` set to your actual frontend domain(s)
- [ ] `APP_URL` set to your frontend URL (used in email links)
- [ ] Rate limiting active: 100 req/sec global, 5 req/sec on `/v1/auth/*` endpoints

### Payment Providers

- [ ] At least one provider has valid credentials and is in `ENABLED_PROVIDERS`
- [ ] `ENABLED_PROVIDERS` lists only providers you intend to use
- [ ] `TRANSFER_DEFAULT_PROVIDER` matches an enabled provider with valid keys
- [ ] `TRANSFER_MIN_AMOUNT` and `TRANSFER_MAX_AMOUNT` reviewed for your use case
- [ ] Paystack webhook registered: `POST {BASE_URL}/paystack/webhooks/` (trailing slash)
- [ ] Paystack billing webhook registered: `POST {BASE_URL}/paystack/webhooks/billing`
- [ ] Korapay webhook registered: `POST {BASE_URL}/korapay/webhooks/deposit`
- [ ] VFD webhooks registered: `inward-credit` and `initial-inward-credit`
- [ ] `VFD_WEBHOOK_SECRET` set and shared with VFD
- [ ] VFD base URL set to production if going live (`https://api-apps.vfdbank.systems`)
- [ ] Paystack key is `sk_live_...` (not `sk_test_...`) if going live

### Email

- [ ] SMTP credentials configured (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`)
- [ ] `SMTP_FROM` address verified with your SMTP provider (Brevo/SendGrid)
- [ ] Test email sent (register a user and check for verification email)

### Data

- [ ] Default subscription plans exist in `subscription_plans` table (3 rows: free, starter, pro)
- [ ] First super admin created (see Section 3.5)

### Health Checks

- [ ] `GET {BASE_URL}/health` returns `{"status":"healthy"}` with database status `ok`
- [ ] `GET {BASE_URL}/health/live` returns `{"status":"alive"}` (liveness probe)
- [ ] Load balancer / PaaS configured to use `/health/live` for liveness and `/health` for readiness

### Monitoring

- [ ] Log aggregation configured (structured JSON logs by default)
- [ ] Check for log message: `"Background scheduler started"` after boot
- [ ] Check for log message: `"Daily reconciliation job scheduled"` after boot
- [ ] Check for log message: `"Weekly provider reconciliation job scheduled"` after boot
- [ ] Uptime monitoring configured on `/health/live` endpoint

---

## 6. Ongoing Operations

### 6.1 Monitor Reconciliation Alerts

PayFlow runs two automated reconciliation jobs:

- **Daily reconciliation** -- starts 1 minute after boot, then runs every 24 hours. Checks wallet balances against ledger entries. Discrepancies trigger notification emails to super admin users.
- **Weekly provider reconciliation** -- starts 5 minutes after boot, then runs every 7 days. Cross-references internal transfer records with Paystack provider records. Available for manual trigger via `GET /platform/reconciliation/provider` (super admin only).

**What to watch for:**
- Log entries containing `"Daily reconciliation failed"` or `"Weekly provider reconciliation failed"`.
- Notification emails to super admin accounts about balance mismatches.
- If the server restarts, both jobs reset their timers (1 min and 5 min respectively).

### 6.2 Check and Retry Failed Asynq Jobs

If Redis is configured, PayFlow uses Asynq for background job processing (email delivery with retry, scheduled payroll processing).

**Using the Asynq CLI:**

```bash
# Install asynq CLI
go install github.com/hibiken/asynq/tools/asynq@latest

# List queues
asynq queue ls --uri=$REDIS_URL

# List failed (archived) tasks
asynq task ls --queue=default --state=archived --uri=$REDIS_URL

# Retry all archived (failed) tasks
asynq task run-all --queue=default --state=archived --uri=$REDIS_URL

# Delete old archived tasks
asynq task delete-all --queue=default --state=archived --uri=$REDIS_URL

# List pending tasks
asynq task ls --queue=default --state=pending --uri=$REDIS_URL

# List scheduled (future) tasks
asynq task ls --queue=default --state=scheduled --uri=$REDIS_URL
```

**Without Redis:** The server uses an in-memory gocron scheduler. Jobs are lost on every restart. You will see this warning in logs: `"Using in-memory scheduler (gocron) -- jobs lost on restart"`. This is acceptable for development but not recommended for production.

### 6.3 Rotate JWT Secret

The `JWT_SECRET` serves two purposes:
1. JWT token signing (all characters).
2. AES-256-GCM encryption key for platform settings (first 32 bytes only).

**Rotation procedure:**

1. Generate a new secret: `openssl rand -base64 48`
2. Update the `JWT_SECRET` environment variable on your deployment platform.
3. Restart the server.
4. **Immediate impact:** All existing JWT tokens are invalidated. Every logged-in user must log in again. There is no graceful dual-key rotation -- it is a hard cutover.
5. **Platform settings impact:** All encrypted platform settings stored via `PUT /platform/settings/{key}` become unreadable because the encryption key changed. You must delete and re-set all platform settings via the API after rotation.

### 6.4 Rotate Encryption Key

The encryption key for platform settings and org provider settings is derived from `JWT_SECRET[:32]` (first 32 bytes). Rotating the JWT secret automatically rotates the encryption key. See Section 6.3.

There is currently no way to rotate the encryption key independently of the JWT secret. Both are tied to `JWT_SECRET` in `cmd/server/main.go` (line ~367: `encryptionKey := cfg.JWTSecret[:32]`).

### 6.5 Switch from Test to Live Mode

1. **Paystack:**
   - Replace `PAYSTACK_SECRET_KEY` from `sk_test_...` to `sk_live_...`.
   - Update webhook URLs in Paystack's **live** dashboard (test and live have separate webhook configurations in Paystack).

2. **Korapay:**
   - Replace `KORAPAY_API_KEY` and `KORAPAY_PUBLIC_KEY` with production keys from the Korapay dashboard.

3. **VFD:**
   - Change `VFD_BASE_URL` from `https://api-devapps.vfdbank.systems` to `https://api-apps.vfdbank.systems`.
   - Ensure `VFD_CONSUMER_KEY` and `VFD_CONSUMER_SECRET` are production credentials (VFD issues separate sandbox and production keys).

4. **Transfer limits:**
   - Review `TRANSFER_MIN_AMOUNT` and `TRANSFER_MAX_AMOUNT` for production-appropriate values.

5. **Safety checks:**
   - Confirm `ENABLE_AUTO_MIGRATION=false`.
   - Confirm `LOG_PRETTY=false` (JSON logs for production log aggregation).

6. Restart the server.
7. Run a small test transfer (NGN 100) to verify end-to-end connectivity with all enabled providers.

### 6.6 Add a Billing Plan

Insert directly into the database:

```sql
INSERT INTO subscription_plans (name, tier, price_monthly, max_employees, max_payroll_runs, features, paystack_plan_code)
VALUES (
    'Enterprise',
    'enterprise',
    10000000,  -- NGN 100,000 in kobo
    0,         -- 0 = unlimited employees
    0,         -- 0 = unlimited payroll runs
    '{"payroll": true, "paye": true, "pension": true, "nhf": true, "reports": true, "csv_import": true, "api_access": true, "priority_support": true, "dedicated_account_manager": true}',
    'PLN_paystack_plan_code_here'  -- Create this plan in Paystack dashboard first, then paste the plan code
);
```

Notes:
- The `tier` column has a unique constraint. Use a unique string.
- `max_employees = 0` means unlimited. Same for `max_payroll_runs`.
- `price_monthly` is stored in kobo (minor currency unit). 10000000 kobo = NGN 100,000.
- `paystack_plan_code` links to a Paystack subscription plan for automated recurring billing. Create the plan in Paystack first, then reference its code here.
- The `features` column is a JSON string. The application checks specific keys in this JSON to gate features.

### 6.7 Promote User to Super Admin

```sql
UPDATE users SET role = 'super_admin' WHERE email = 'user@example.com';
```

Verify by logging in with that account. The user should now have access to all `/platform/*` endpoints:

| Endpoint | Method | Purpose |
|---|---|---|
| `/platform/stats` | `GET` | Platform-wide statistics (total orgs, users, revenue) |
| `/platform/organizations` | `GET` | List all organizations |
| `/platform/organizations/{id}/suspend` | `POST` | Suspend an organization |
| `/platform/organizations/{id}/activate` | `POST` | Reactivate a suspended organization |
| `/platform/settings` | `GET` | List all encrypted platform settings |
| `/platform/settings/{key}` | `PUT` | Set/update an encrypted platform setting |
| `/platform/settings/{key}` | `DELETE` | Delete a platform setting |
| `/platform/reconciliation/provider` | `GET` | Trigger manual provider reconciliation |

### 6.8 Handle Failed Disbursements

When a payroll transfer or individual transfer fails:

1. **Check transfer status:**
   ```
   GET /v1/transfers/{id}
   ```
   Inspect the `status` and `provider_response` fields for the failure reason.

2. **Retry a single transfer:**
   ```
   POST /v1/transfers/{id}/retry
   ```
   Re-submits to the provider. The system tries the default provider first, then falls through the `TRANSFER_PROVIDER_FALLBACK_ORDER`.

3. **List all failed transfers:**
   ```
   GET /v1/transfers?status=failed
   ```
   Review and retry individually.

4. **Provider-level outage:** If a provider is down system-wide:
   - Option A: Wait for the provider to recover, then retry failed transfers.
   - Option B: Change `TRANSFER_DEFAULT_PROVIDER` to a different enabled provider and restart. New transfers will route to the new provider. The `TRANSFER_PROVIDER_FALLBACK_ORDER` handles this automatically for individual failures.

5. **Reverse an entire payroll run:**
   ```
   POST /v1/payroll-runs/{runID}/reverse
   ```
   Use this if a payroll run has systemic failures (e.g., wrong amounts, wrong accounts).

6. **Check Asynq queue:** Failed async transfer jobs may be sitting in the Asynq archived queue. See Section 6.2 for how to list and retry them.

### 6.9 Velocity Limits (Hardcoded)

Transfer creation is rate-limited per business (enforced by middleware):

| Operation | Per Hour | Per Day |
|---|---|---|
| Single transfer (`POST /v1/transfers/`) | 10 | 50 |
| Batch transfer (`POST /v1/transfers/batch`) | 5 | 20 |

These are hardcoded in `internal/api/router.go` in the `TransferVelocityConfig` struct. To change them, modify the values and redeploy.

Global API rate limit: 100 requests/second per IP (200 burst). Auth endpoints (`/v1/auth/*`): 5 requests/second per IP (10 burst).

### 6.10 Startup Sequence Reference

The server boots in six phases. Understanding this helps debug startup failures:

| Phase | What happens | Fatal if fails? |
|---|---|---|
| 1. Configuration & Logger | Loads config from env/.env, validates `JWT_SECRET`, configures zerolog | Yes (missing JWT_SECRET) |
| 2. Platform & Repository | Connects to PostgreSQL (primary + optional read replica), connects to Redis (optional), runs auto-migration if enabled, initializes all repositories | Yes (DB connection failure) |
| 3. External Services | Creates Korapay client, VFD client, Paystack client (if keys set), email service (direct SMTP or Asynq-backed), cache service | No (providers degrade gracefully) |
| 4. Scheduler | Creates Asynq scheduler (if Redis) or gocron fallback, creates PayrollService, resolves circular dependency | Yes (scheduler creation failure) |
| 5. API Router & HTTP Server | Builds Chi router with middleware and handlers, starts HTTP listener, registers email handler with scheduler | Yes (port binding failure) |
| 6. Graceful Shutdown | Waits for SIGINT/SIGTERM, stops scheduler (lets running jobs finish), shuts down HTTP server (30-second timeout) | N/A |

### 6.11 Platform Settings (Encrypted Key-Value Store)

Super admins can store sensitive configuration at runtime without redeployment:

```bash
# Set a setting (encrypted at rest with AES-256-GCM)
curl -X PUT {BASE_URL}/platform/settings/CUSTOM_API_KEY \
  -H "Authorization: Bearer {super_admin_jwt}" \
  -H "Content-Type: application/json" \
  -d '{"value": "secret-value-here"}'

# List all settings (values are decrypted in the response)
curl {BASE_URL}/platform/settings \
  -H "Authorization: Bearer {super_admin_jwt}"

# Delete a setting
curl -X DELETE {BASE_URL}/platform/settings/CUSTOM_API_KEY \
  -H "Authorization: Bearer {super_admin_jwt}"
```

### 6.12 Org-Level Provider Key Overrides

Individual organizations (businesses) can override the platform-level payment provider API keys. This enables multi-tenant setups where each business uses their own Paystack/Korapay/VFD credentials:

```bash
# Set an org-level key (requires admin role within that org)
curl -X PUT {BASE_URL}/v1/business/provider-keys/paystack/secret_key \
  -H "Authorization: Bearer {admin_jwt}" \
  -H "Content-Type: application/json" \
  -d '{"value": "sk_live_org_specific_key"}'

# List org keys
curl {BASE_URL}/v1/business/provider-keys \
  -H "Authorization: Bearer {admin_jwt}"

# Delete an org key (falls back to platform-level key)
curl -X DELETE {BASE_URL}/v1/business/provider-keys/paystack/secret_key \
  -H "Authorization: Bearer {admin_jwt}"
```

### 6.13 Database Maintenance

```bash
# Check current migration version
migrate -database "$DB_URL" -path ./migrations version

# Apply pending migrations
migrate -database "$DB_URL" -path ./migrations up

# Rollback last migration (one step)
migrate -database "$DB_URL" -path ./migrations down 1

# Force version (ONLY use when migration is stuck in dirty state)
migrate -database "$DB_URL" -path ./migrations force 27

# Create a new migration file
migrate create -ext sql -dir migrations -seq migration_name
```

**Backup:**
- `pg_dump -Fc $DB_URL > payflow_backup_$(date +%Y%m%d).dump`
- `pg_restore -d $DB_URL payflow_backup_20260601.dump`
- Redis data is ephemeral (cache + job queue). No backup needed -- state rebuilds on restart.

---

## Quick Reference: Minimum Viable Configuration

The absolute minimum environment variables to start the server:

```bash
DB_URL=postgres://user:pass@host:5432/payflow_db?sslmode=require
JWT_SECRET=your-cryptographically-random-string-minimum-32-characters
```

This gets you: server running, database connected, auth working (register, login, JWT issuance). No transfers, no emails, no caching, no persistent job queue. Add provider keys, SMTP, and Redis incrementally as needed.
