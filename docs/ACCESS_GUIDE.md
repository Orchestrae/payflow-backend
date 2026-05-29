# PayFlow Access Guide

How to access each interface, create accounts, and test the full system.

## Live URLs

| Service | URL |
|---------|-----|
| **Frontend (Admin Dashboard)** | https://payflowio.vercel.app |
| **Backend API** | https://payflow-api-production-bf5e.up.railway.app |
| **Health Check** | https://payflow-api-production-bf5e.up.railway.app/health |
| **Local Frontend Dev** | http://localhost:5173 |
| **Local Backend Dev** | http://localhost:8080 |

## Repositories

| Repo | URL |
|------|-----|
| Backend | https://github.com/Orchestrae/payflow-backend |
| Frontend | https://github.com/Orchestrae/payflow-frontend |

---

## 1. Super Admin (Platform Owner)

The super admin manages the entire PayFlow platform — views all organizations, MRR, and can suspend/activate orgs.

### Create Super Admin Account

Super admin must be created directly in the database (not via API, for security):

```sql
-- Connect to Railway Postgres
-- Insert super admin user (BusinessID = 0, not tied to any org)
INSERT INTO users (email, password_hash, role, business_id, is_verified, created_at, updated_at)
VALUES (
    'superadmin@payflow.com',
    -- Generate hash: use bcrypt with cost 14
    '$2a$14$YOUR_BCRYPT_HASH_HERE',
    'super_admin',
    0,
    true,
    NOW(),
    NOW()
);
```

Or generate the hash locally:

```bash
# In a Go playground or script:
go run -e 'import "golang.org/x/crypto/bcrypt"; h, _ := bcrypt.GenerateFromPassword([]byte("YourSuperAdminPassword!"), 14); fmt.Println(string(h))'
```

### Login

```bash
curl -X POST https://payflow-api-production-bf5e.up.railway.app/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"superadmin@payflow.com","password":"YourSuperAdminPassword!"}'
```

### Available Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/platform/stats` | MRR, total orgs, signups, plan distribution |
| GET | `/platform/organizations` | All organizations with details |
| POST | `/platform/organizations/{id}/suspend` | Suspend an org |
| POST | `/platform/organizations/{id}/activate` | Reactivate an org |

---

## 2. Business Admin (Organization Owner)

The business admin is the first user created when an organization registers. They manage employees, payroll, transfers, and settings.

### Register a New Business

```bash
curl -X POST https://payflow-api-production-bf5e.up.railway.app/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "business_name": "Your Company Name",
    "email": "admin@yourcompany.com",
    "password": "SecurePassword123!",
    "rc_number": "RC100001",
    "incorporation_date": "2020-01-01T00:00:00Z",
    "director_bvn": "12345678901"
  }'
```

### Login

```bash
curl -X POST https://payflow-api-production-bf5e.up.railway.app/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@yourcompany.com","password":"SecurePassword123!"}'
```

Or login via the frontend at **https://payflowio.vercel.app**.

### What Admin Can Do

- Manage employees (CRUD, CSV import, deactivate)
- Manage cadres/salary structures
- Configure deduction rules
- Run payroll (create, submit, approve, process)
- Download reports (PAYE, pension, NHF, bank schedule, payslips)
- Manage transfers (single, batch, retry)
- View wallet and transactions
- Configure business settings (toggle statutory deductions)
- Invite team members (operators, approvers)
- View audit logs
- Manage subscription and billing

### Test Account (already created)

```
Email:    admin@payflowtest.com
Password: TestAdmin123!
Business: PayFlow Test Corp
Plan:     Free (5 employees max)
```

---

## 3. Operator

Operators handle day-to-day payroll operations — creating employees, running payroll, managing transfers.

### Create an Operator

An admin invites operators via the API:

```bash
curl -X POST https://payflow-api-production-bf5e.up.railway.app/v1/auth/invite \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-jwt-token>" \
  -d '{"email":"operator@yourcompany.com","role":"operator"}'
```

The operator receives an invitation email with a link to set their password.

### Accept Invitation

```bash
curl -X POST https://payflow-api-production-bf5e.up.railway.app/v1/auth/accept-invitation \
  -H "Content-Type: application/json" \
  -d '{"token":"<invite-token-from-email>","password":"OperatorPassword123!"}'
```

### What Operator Can Do

- Manage employees and cadres
- Create and submit payroll runs
- Process payroll (instant)
- Manage transfers
- Download reports and payslips
- Cannot: approve/reject payroll, manage deduction rules, change settings, invite users

---

## 4. Approver

Approvers review and approve/reject payroll runs submitted by operators.

### Create an Approver

```bash
curl -X POST https://payflow-api-production-bf5e.up.railway.app/v1/auth/invite \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-jwt-token>" \
  -d '{"email":"approver@yourcompany.com","role":"approver"}'
```

### What Approver Can Do

- View payroll runs
- Approve payroll runs
- Reject payroll runs (with reason)
- Cannot: create employees, modify cadres, process payroll, manage transfers

---

## 5. Employee (Self-Service)

Employees can view their own payslips and update their bank details.

### Create Employee Account

An admin creates an employee self-service account:

```bash
# First, the employee must exist in the system (created by admin/operator)
# Then create their self-service user account (future implementation)
```

### What Employee Can Do

- `GET /v1/me/profile` — view own employee profile
- `PATCH /v1/me/bank-details` — update bank name, code, account number
- `GET /v1/me/payslips` — view own payslip history
- Cannot: access any other business data

---

## Subscription Plans

| Plan | Price | Max Employees | Max Payroll Runs/mo |
|------|-------|---------------|---------------------|
| **Free** | NGN 0 | 5 | 2 |
| **Starter** | NGN 15,000/mo | 50 | Unlimited |
| **Pro** | NGN 50,000/mo | Unlimited | Unlimited |

### View Plans

```bash
curl https://payflow-api-production-bf5e.up.railway.app/v1/billing/plans \
  -H "Authorization: Bearer <jwt-token>"
```

### Upgrade

```bash
curl -X POST https://payflow-api-production-bf5e.up.railway.app/v1/billing/subscribe \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <jwt-token>" \
  -d '{"tier":"starter"}'
```

Returns a Paystack payment URL to complete the subscription.

---

## Password Reset

```bash
# Request reset email
curl -X POST https://payflow-api-production-bf5e.up.railway.app/v1/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@yourcompany.com"}'

# Reset with token from email
curl -X POST https://payflow-api-production-bf5e.up.railway.app/v1/auth/reset-password \
  -H "Content-Type: application/json" \
  -d '{"token":"<token-from-email>","new_password":"NewPassword123!"}'
```

---

## Quick Test Flow

1. Register: `POST /v1/auth/register` → creates business + admin
2. Login: `POST /v1/auth/login` → get JWT token
3. Create deduction rules: `POST /v1/deduction-rules`
4. Create cadre: `POST /v1/cadres` (with earning components + component_type)
5. Add employees: `POST /v1/employees` (or CSV import)
6. Enable statutory: `PATCH /v1/business/settings {"pension_enabled": true, "paye_enabled": true}`
7. Create payroll: `POST /v1/payroll-runs {"period": "2026-01"}`
8. Submit: `POST /v1/payroll-runs/{id}/submit`
9. Approve: `POST /v1/payroll-runs/{id}/approve`
10. Process: `POST /v1/payroll-runs/{id}/process-now`
11. Download payslip: `GET /v1/payroll-runs/{id}/payslips/{empId}`
12. Download reports: `GET /v1/payroll-runs/{id}/reports/paye`

---

## Local Development

```bash
# Start infrastructure
make up

# Run migrations
make migrate-up

# Start backend
JWT_SECRET=dev-secret-32-chars-minimum-here make run

# Start frontend (separate terminal)
cd frontend && npm run dev
```

Backend: http://localhost:8080
Frontend: http://localhost:5173
MailHog: http://localhost:8025 (email testing UI)
