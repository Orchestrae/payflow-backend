# PayFlow API Reference

> Complete API reference for frontend engineers integrating with PayFlow.

**Base URL (local development):** `http://localhost:8080`
**Base URL (production):** Replace with your deployment domain.

**Related Documentation:**
- [Architecture](./ARCHITECTURE.md) -- system design and patterns
- [Payroll Guide](./PAYROLL_GUIDE.md) -- payroll workflow details
- [Wallet Guide](./WALLET_GUIDE.md) -- wallet and virtual account details
- [Deployment Guide](./DEPLOYMENT.md) -- setup, env vars, Railway

**Rate Limits:**
- Global: 100 requests/second per IP
- Auth endpoints (`/v1/auth/*`): 5 requests/second per IP
- Exceeding limits returns `429 Too Many Requests`

**Request ID:**
- Every response includes an `X-Request-ID` header for tracing
- You can pass your own `X-Request-ID` in the request to correlate with your logs

**New Endpoints (not in original collection -- update Postman manually):**

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/v1/auth/invite` | Admin | Invite user to business |
| POST | `/v1/auth/accept-invitation` | Public | Accept invite, set password |
| POST | `/v1/auth/forgot-password` | Public | Request password reset |
| POST | `/v1/auth/reset-password` | Public | Reset password with token |
| GET | `/v1/dashboard` | Any | Dashboard summary metrics |
| GET | `/v1/business/settings` | Admin | Get business settings |
| PATCH | `/v1/business/settings` | Admin | Update statutory/workflow config |
| GET | `/v1/payroll-runs/{id}/reports/paye` | Admin/Op | PAYE return CSV |
| GET | `/v1/payroll-runs/{id}/reports/pension` | Admin/Op | Pension schedule CSV |
| GET | `/v1/payroll-runs/{id}/reports/nhf` | Admin/Op | NHF schedule CSV |
| GET | `/v1/payroll-runs/{id}/reports/bank-schedule` | Admin/Op | Bank transfer CSV |
| GET | `/v1/payroll-runs/{id}/payslips/{empID}` | Admin/Op | Employee payslip PDF |
| GET | `/v1/payroll-runs/{id}/payslips` | Admin/Op | All payslips ZIP |
| GET | `/v1/payroll-runs/{id}/reports/summary` | Admin/Op | Payroll summary CSV |
| POST | `/v1/employees/import` | Admin/Op | CSV employee import |
| POST | `/v1/transfers/{id}/retry` | Any | Retry failed transfer |
| GET | `/v1/audit-logs` | Admin | Audit trail (paginated) |
| GET | `/v1/notifications` | Any | In-app notifications |
| GET | `/v1/notifications/unread-count` | Any | Unread badge count |
| PATCH | `/v1/notifications/{id}/read` | Any | Mark notification read |
| GET | `/v1/verify/bank-account` | Any | Validate bank account (Paystack) |
| GET | `/v1/me/profile` | Employee | Self-service profile |
| PATCH | `/v1/me/bank-details` | Employee | Update own bank details |
| GET | `/v1/me/payslips` | Employee | Own payslip history |
| POST | `/v1/loans` | Admin/Op | Create employee loan |
| GET | `/v1/loans` | Admin/Op | List loans |
| PATCH | `/v1/loans/{id}/cancel` | Admin/Op | Cancel active loan |
| GET | `/v1/billing/plans` | Any | List subscription plans |
| GET | `/v1/billing/subscription` | Any | Current subscription |
| POST | `/v1/billing/subscribe` | Admin | Subscribe/upgrade (Paystack URL) |
| POST | `/v1/billing/cancel` | Admin | Cancel subscription |
| GET | `/v1/billing/invoices` | Admin | Payment history |
| GET | `/platform/stats` | SuperAdmin | Platform dashboard (MRR, orgs) |
| GET | `/platform/organizations` | SuperAdmin | All organizations |
| POST | `/platform/organizations/{id}/suspend` | SuperAdmin | Suspend org |
| POST | `/platform/organizations/{id}/activate` | SuperAdmin | Activate org |

---

## Table of Contents

1. [Overview](#1-overview)
2. [Authentication](#2-authentication)
3. [Complete API Flow](#3-complete-api-flow)
   - [Step 1: Register Business](#step-1-register-business)
   - [Step 2: Login](#step-2-login)
   - [Step 3: Create Deduction Rules](#step-3-create-deduction-rules)
   - [Step 4: Create Cadres (Salary Structures)](#step-4-create-cadres-salary-structures)
   - [Step 5: Add Employees](#step-5-add-employees)
   - [Step 6: Create Payroll Run](#step-6-create-payroll-run)
   - [Step 7: Submit Payroll for Approval](#step-7-submit-payroll-for-approval)
   - [Step 8: Approve Payroll](#step-8-approve-payroll)
   - [Step 9: Process Payroll (Instant Disbursement)](#step-9-process-payroll-instant-disbursement)
   - [Step 10: Reject Payroll (Alternative Flow)](#step-10-reject-payroll-alternative-flow)
4. [Wallet and Virtual Account Management](#4-wallet-and-virtual-account-management)
   - [Create Account Holder (KYC)](#create-account-holder-kyc)
   - [Create Virtual Account](#create-virtual-account)
   - [Get Wallet](#get-wallet)
   - [Check Balance](#check-balance)
   - [Get Transactions](#get-transactions)
   - [Sandbox Credit (Testing Only)](#sandbox-credit-testing-only)
   - [Generate File Upload URL](#generate-file-upload-url)
5. [Transfers](#5-transfers)
   - [Single Transfer](#single-transfer)
   - [Batch Transfer](#batch-transfer)
   - [List Transfers](#list-transfers)
   - [Get Transfer by ID](#get-transfer-by-id)
6. [Employee Management (Full CRUD)](#6-employee-management-full-crud)
7. [Cadre Management (Full CRUD)](#7-cadre-management-full-crud)
8. [Deduction Rule Management (Full CRUD)](#8-deduction-rule-management-full-crud)
9. [Payroll Runs (Full CRUD)](#9-payroll-runs-full-crud)
10. [Error Handling](#10-error-handling)
11. [Role-Based Access Control](#11-role-based-access-control)
12. [Testing Order](#12-testing-order)
13. [Health Check](#13-health-check)

---

## 1. Overview

Payflow is a payroll management system that allows Nigerian businesses to:

- **Register a business** and onboard employees with structured salary grades (cadres)
- **Define deduction rules** (tax, pension, etc.) as percentages or flat amounts
- **Create salary structures (cadres)** with earning components (basic salary, housing, transport, etc.)
- **Run payroll** with automatic gross pay calculation, deduction application, and net pay computation
- **Approve/reject payroll** through a multi-role workflow (operator creates, approver approves)
- **Disburse salaries** via Korapay integration for bank transfers
- **Manage wallets** with virtual bank accounts for receiving and sending funds

The API follows REST conventions with JSON request/response bodies. All monetary values in the payroll system are stored as **integers in the smallest currency unit (kobo)** -- for example, NGN 500,000 is stored as `500000`. Earning component amounts in cadres use the same convention.

---

## 2. Authentication

### How It Works

Payflow uses **JWT (JSON Web Token)** authentication. After logging in, you receive a token that must be included in all subsequent requests to protected endpoints.

### Token Format

The JWT contains the following claims:

| Claim        | Description                              |
|--------------|------------------------------------------|
| `user_id`    | The authenticated user's ID              |
| `business_id`| The business the user belongs to         |
| `role`       | The user's role: `admin`, `operator`, or `approver` |

### Where to Put the Token

Include the token in the `Authorization` header using the Bearer scheme:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Required Headers for All Authenticated Requests

```
Content-Type: application/json
Authorization: Bearer <your-jwt-token>
```

### Token Lifecycle

- Tokens are returned in the login response
- Registration does NOT return a token -- you must log in after registering
- If a token expires or is invalid, the API returns `401 Unauthorized`

---

## 3. Complete API Flow

This section walks through the entire payroll lifecycle from business registration to salary disbursement.

### Step 1: Register Business

Register a new business. This creates the business, an admin user, and initiates corporate account setup.

```
POST /v1/auth/register
Content-Type: application/json
```

**Request Body:**

```json
{
  "business_name": "Acme Corp",
  "email": "admin@acmecorp.com",
  "password": "SecurePassword123",
  "rc_number": "RC123456",
  "incorporation_date": "2020-01-15T00:00:00Z",
  "director_bvn": "12345678901"
}
```

| Field               | Type     | Required | Validation                     |
|---------------------|----------|----------|--------------------------------|
| `business_name`     | string   | Yes      | -                              |
| `email`             | string   | Yes      | Must be a valid email          |
| `password`          | string   | Yes      | Minimum 8 characters           |
| `rc_number`         | string   | Yes      | Business registration number   |
| `incorporation_date`| string   | Yes      | ISO 8601 datetime              |
| `director_bvn`      | string   | Yes      | Exactly 11 digits              |

**Response (201 Created):**

```json
{
  "user": {
    "id": 1,
    "email": "admin@acmecorp.com",
    "role": "admin",
    "business_id": 1
  },
  "corporate_account": {
    "account_number": "",
    "account_name": "Acme Corp"
  }
}
```

> **Note:** The `account_number` may be empty initially if the corporate account creation with the payment provider is asynchronous. The first registered user is always assigned the `admin` role.

---

### Step 2: Login

Authenticate and receive a JWT token.

```
POST /v1/auth/login
Content-Type: application/json
```

**Request Body:**

```json
{
  "email": "admin@acmecorp.com",
  "password": "SecurePassword123"
}
```

| Field      | Type   | Required | Validation            |
|------------|--------|----------|-----------------------|
| `email`    | string | Yes      | Must be a valid email |
| `password` | string | Yes      | -                     |

**Response (200 OK):**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMSIsImJ1c2luZXNzX2lkIjoiMSIsInJvbGUiOiJhZG1pbiIsImV4cCI6MTc0MDI2MjQwMH0.abc123...",
  "user": {
    "id": 1,
    "email": "admin@acmecorp.com",
    "role": "admin",
    "business_id": 1
  }
}
```

> **Save the `token` value.** You will use it in the `Authorization` header for all subsequent requests.

---

### Step 3: Create Deduction Rules

Define company-wide deduction rules (e.g., tax, pension, health insurance). These are applied automatically when payroll is calculated.

**Roles:** Admin only

```
POST /v1/deduction-rules
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "name": "Tax",
  "type": "percentage",
  "value": 7.5,
  "calculation_basis": "gross_pay"
}
```

| Field              | Type   | Required | Validation                          |
|--------------------|--------|----------|-------------------------------------|
| `name`             | string | Yes      | -                                   |
| `type`             | string | Yes      | Must be `"percentage"` or `"flat"`  |
| `value`            | number | Yes      | Percentage (e.g., 7.5) or flat amount in kobo |
| `calculation_basis`| string | Yes      | Must be `"gross_pay"` or `"basic_pay"` |

**Response (201 Created):**

```json
{
  "ID": 1,
  "CreatedAt": "2026-02-21T10:00:00Z",
  "UpdatedAt": "2026-02-21T10:00:00Z",
  "DeletedAt": null,
  "BusinessID": 1,
  "CadreID": 0,
  "Name": "Tax",
  "Type": "percentage",
  "Value": 7.5,
  "CalculationBasis": "gross_pay"
}
```

**More Examples:**

```json
{
  "name": "Pension",
  "type": "percentage",
  "value": 8.0,
  "calculation_basis": "basic_pay"
}
```

```json
{
  "name": "Health Insurance",
  "type": "flat",
  "value": 15000,
  "calculation_basis": "gross_pay"
}
```

---

### Step 4: Create Cadres (Salary Structures)

Cadres represent salary grades/levels. Each cadre defines a set of earning components that determine gross pay.

**Roles:** Admin, Operator

```
POST /v1/cadres
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "name": "Senior Engineer",
  "earning_components": [
    { "name": "Basic Salary", "amount": 500000 },
    { "name": "Housing", "amount": 150000 },
    { "name": "Transport", "amount": 50000 }
  ],
  "deduction_rule_ids": [1, 2]
}
```

| Field                | Type   | Required | Description                                    |
|----------------------|--------|----------|------------------------------------------------|
| `name`               | string | Yes      | Name of the salary grade                       |
| `earning_components` | array  | Yes      | Array of `{ name, amount }` objects            |
| `deduction_rule_ids` | array  | No       | IDs of deduction rules to attach to this cadre |

Each earning component:

| Field    | Type   | Required | Description                              |
|----------|--------|----------|------------------------------------------|
| `name`   | string | Yes      | Component name (e.g., "Basic Salary")    |
| `amount` | int64  | Yes      | Amount in kobo (smallest currency unit)  |

**Response (201 Created):**

```json
{
  "ID": 1,
  "CreatedAt": "2026-02-21T10:05:00Z",
  "UpdatedAt": "2026-02-21T10:05:00Z",
  "DeletedAt": null,
  "BusinessID": 1,
  "Name": "Senior Engineer",
  "EarningComponents": [
    { "ID": 1, "CreatedAt": "...", "UpdatedAt": "...", "DeletedAt": null, "CadreID": 1, "Name": "Basic Salary", "Amount": 500000 },
    { "ID": 2, "CreatedAt": "...", "UpdatedAt": "...", "DeletedAt": null, "CadreID": 1, "Name": "Housing", "Amount": 150000 },
    { "ID": 3, "CreatedAt": "...", "UpdatedAt": "...", "DeletedAt": null, "CadreID": 1, "Name": "Transport", "Amount": 50000 }
  ],
  "DeductionRules": [],
  "Employees": null
}
```

> **Note:** The gross pay for this cadre is the sum of all earning components: 500,000 + 150,000 + 50,000 = NGN 700,000.

---

### Step 5: Add Employees

Add employees to the business. Each employee must be assigned to a cadre, which determines their salary.

**Roles:** Admin, Operator

```
POST /v1/employees
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "full_name": "John Doe",
  "email": "john@acmecorp.com",
  "cadre_id": 1,
  "bank_name": "GTBank",
  "bank_code": "058",
  "bank_account_number": "0123456789"
}
```

| Field                 | Type   | Required | Description                        |
|-----------------------|--------|----------|------------------------------------|
| `full_name`           | string | Yes      | Employee's full name               |
| `email`               | string | Yes      | Must be a valid email              |
| `cadre_id`            | uint   | Yes      | ID of the cadre (salary grade)     |
| `bank_name`           | string | Yes      | Name of the employee's bank        |
| `bank_code`           | string | No       | Bank code (e.g., "058" for GTBank) |
| `bank_account_number` | string | Yes      | Bank account number for salary     |

**Response (201 Created):**

```json
{
  "ID": 1,
  "CreatedAt": "2026-02-21T10:10:00Z",
  "UpdatedAt": "2026-02-21T10:10:00Z",
  "DeletedAt": null,
  "BusinessID": 1,
  "CadreID": 1,
  "FullName": "John Doe",
  "Email": "john@acmecorp.com",
  "BankName": "GTBank",
  "BankCode": "058",
  "BankAccountNumber": "0123456789",
  "IsActive": true,
  "Cadre": null
}
```

---

### Step 6: Create Payroll Run

Create a new payroll run for a given period. The system automatically calculates gross pay, applies deductions, and computes net pay for all active employees.

**Roles:** Admin, Operator

```
POST /v1/payroll-runs
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "period": "2026-02"
}
```

| Field         | Type   | Required | Description                                                     |
|---------------|--------|----------|-----------------------------------------------------------------|
| `period`      | string | No       | Format: `"YYYY-MM"` or `"YYYY-MM-DD"`. Defaults to current month if omitted |
| `adjustments` | object | No       | Per-employee adjustments (bonuses, penalties, etc.)              |

**With adjustments:**

```json
{
  "period": "2026-02",
  "adjustments": {
    "1": [
      {
        "item_name": "Performance Bonus",
        "amount": 100000,
        "description": "Q4 2025 performance bonus"
      }
    ],
    "john@acmecorp.com": [
      {
        "item_name": "Late Penalty",
        "amount": -5000,
        "description": "Late attendance deduction",
        "component_type": "deduction"
      }
    ]
  }
}
```

Adjustment keys can be **employee ID (as string)** or **employee email**. Each adjustment item:

| Field            | Type   | Required | Description                                                |
|------------------|--------|----------|------------------------------------------------------------|
| `item_name`      | string | Yes      | Name of the adjustment (e.g., "Performance Bonus")         |
| `amount`         | int64  | Yes      | Positive for earnings, negative for deductions (in kobo)   |
| `description`    | string | No       | Optional description for audit trail                       |
| `component_type` | string | No       | `"earnings"` or `"deduction"` (auto-inferred from amount sign if not provided) |

**Response (201 Created):**

```json
{
  "ID": 1,
  "CreatedAt": "2026-02-21T10:15:00Z",
  "UpdatedAt": "2026-02-21T10:15:00Z",
  "DeletedAt": null,
  "BusinessID": 1,
  "Period": "2026-02-01T00:00:00Z",
  "Status": "draft",
  "TotalGrossPay": 700000,
  "TotalDeductions": 52500,
  "TotalNetPay": 647500,
  "ScheduledFor": "0001-01-01T00:00:00Z",
  "ProcessedAt": null,
  "PaymentReference": "",
  "RejectionReason": "",
  "Entries": [
    {
      "ID": 1,
      "PayrollRunID": 1,
      "EmployeeID": 1,
      "GrossPay": 700000,
      "TotalDeductions": 52500,
      "Bonuses": 0,
      "NetPay": 647500,
      "Employee": null,
      "Details": [
        { "Type": "earning", "Name": "Basic Salary", "Amount": 500000 },
        { "Type": "earning", "Name": "Housing", "Amount": 150000 },
        { "Type": "earning", "Name": "Transport", "Amount": 50000 },
        { "Type": "deduction", "Name": "Tax", "Amount": 52500 }
      ]
    }
  ]
}
```

> **Note:** The payroll run starts in `"draft"` status. It must go through the approval workflow before it can be processed.

---

### Step 7: Submit Payroll for Approval

Move a payroll run from `draft` to `pending_approval` status.

**Roles:** Admin, Operator

```
POST /v1/payroll-runs/{runID}/submit
Authorization: Bearer <token>
```

**No request body required.**

**Response (200 OK):**

```json
{
  "ID": 1,
  "Status": "pending_approval",
  "TotalGrossPay": 700000,
  "TotalDeductions": 52500,
  "TotalNetPay": 647500,
  "..."
}
```

---

### Step 8: Approve Payroll

Approve a pending payroll run. This moves it to `approved` status, making it eligible for processing.

**Roles:** Admin, Approver

```
POST /v1/payroll-runs/{runID}/approve
Authorization: Bearer <token>
```

**No request body required.**

**Response (200 OK):**

```json
{
  "ID": 1,
  "Status": "approved",
  "..."
}
```

---

### Step 9: Process Payroll (Instant Disbursement)

Process an approved payroll run immediately. This triggers the actual bank transfers to all employees.

**Roles:** Admin, Operator

```
POST /v1/payroll-runs/{runID}/process-now
Authorization: Bearer <token>
```

**No request body required.**

**Response (200 OK):**

```json
{
  "ID": 1,
  "Status": "processing",
  "PaymentReference": "PAY-20260221-abc123",
  "..."
}
```

> **Payroll Status Flow:**
> ```
> draft --> pending_approval --> approved --> processing --> completed
>                                    |
>                                    +--> rejected (see Step 10)
>
> processing --> failed (if transfer errors occur)
> ```

---

### Step 10: Reject Payroll (Alternative Flow)

Reject a payroll run that is pending approval. Requires a reason.

**Roles:** Admin, Approver

```
POST /v1/payroll-runs/{runID}/reject
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "reason": "Incorrect bonus amounts for the engineering team"
}
```

| Field    | Type   | Required | Description               |
|----------|--------|----------|---------------------------|
| `reason` | string | Yes      | Reason for the rejection  |

**Response (200 OK):**

```json
{
  "ID": 1,
  "Status": "rejected",
  "RejectionReason": "Incorrect bonus amounts for the engineering team",
  "..."
}
```

---

## 4. Wallet and Virtual Account Management

Payflow integrates with Korapay for wallet and virtual account functionality. This allows businesses to receive funds into a virtual bank account and disburse payments.

### Create Account Holder (KYC)

Before creating a virtual account, you must create an account holder with KYC information.

```
POST /v1/wallets/account-holders
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "first_name": "Tolu",
  "last_name": "Ade",
  "use_case": "Business",
  "type": "individual",
  "date_of_birth": "1990-05-15",
  "nationality": "NG",
  "occupation": "Software Engineer",
  "email": "tolu@acmecorp.com",
  "phone": "+2348133443000",
  "bank_id_number": "12345678901",
  "source_of_inflow": "business_income",
  "identification": {
    "type": "national_id",
    "number": "A12345678",
    "country": "NG"
  },
  "address": {
    "country": "NG",
    "address": "123 Main Street",
    "state": "Lagos",
    "city": "Ikeja"
  },
  "employment": {
    "status": "employer",
    "employer": "Acme Corp",
    "description": "Tech company"
  }
}
```

| Field              | Type   | Required | Description                                             |
|--------------------|--------|----------|---------------------------------------------------------|
| `first_name`       | string | Yes      | -                                                       |
| `last_name`        | string | Yes      | -                                                       |
| `use_case`         | string | Yes      | `"Personal"` or `"Business"`                            |
| `type`             | string | Yes      | `"individual"` or `"business"`                          |
| `date_of_birth`    | string | Yes      | Format: `YYYY-MM-DD`                                    |
| `nationality`      | string | Yes      | Country code (e.g., `"NG"`)                             |
| `occupation`       | string | No       | -                                                       |
| `email`            | string | Yes      | Must be a valid email                                   |
| `phone`            | string | Yes      | International format (e.g., `"+2348133443000"`)         |
| `bank_id_number`   | string | No       | BVN or similar                                          |
| `source_of_inflow` | string | Yes      | e.g., `"salary"`, `"business_income"`, `"bank_statement"` |
| `identification`   | object | No       | ID document details (see below)                         |
| `address`          | object | No       | Physical address details (see below)                    |
| `employment`       | object | No       | Employment details (see below)                          |
| `metadata`         | object | No       | Arbitrary key-value pairs                               |

**Response (201 Created):** Returns the created account holder object from Korapay.

---

### Create Virtual Account

Create a virtual bank account for the business to receive payments.

```
POST /v1/wallets/virtual-account
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "account_name": "Acme Corp Payroll",
  "customer_name": "Acme Corp",
  "customer_email": "finance@acmecorp.com",
  "bvn": "12345678901",
  "permanent": true
}
```

| Field               | Type   | Required | Description                                    |
|---------------------|--------|----------|------------------------------------------------|
| `account_name`      | string | Yes      | Name for the virtual account                   |
| `account_reference` | string | No       | Custom reference (auto-generated if omitted)   |
| `customer_name`     | string | Yes      | Customer/business name                         |
| `customer_email`    | string | No       | Email for notifications                        |
| `bvn`               | string | Yes      | Exactly 11 digits                              |
| `nin`               | string | No       | National ID Number                             |
| `bank_code`         | string | No       | Preferred bank code (provider may assign)      |
| `permanent`         | bool   | No       | Defaults to `true`                             |

**Response (201 Created):** Returns the virtual account details including account number and bank.

---

### Get Wallet

Retrieve the wallet associated with the authenticated business.

```
GET /v1/wallets
Authorization: Bearer <token>
```

**Response (200 OK):** Returns the wallet object with virtual account details.

---

### Check Balance

Get the current wallet balance.

```
GET /v1/wallets/balance
Authorization: Bearer <token>
```

**Response (200 OK):**

```json
{
  "balance": 5000000,
  "currency": "NGN"
}
```

> **Note:** Balance is in the smallest currency unit (kobo). 5000000 kobo = NGN 50,000.

---

### Get Transactions

List wallet transactions with pagination.

```
GET /v1/wallets/transactions?page=1&limit=10
Authorization: Bearer <token>
```

| Query Param | Type | Default | Description              |
|-------------|------|---------|--------------------------|
| `page`      | int  | 1       | Page number              |
| `limit`     | int  | 10      | Items per page           |

**Response (200 OK):**

```json
{
  "transactions": [...],
  "total": 25,
  "page": 1,
  "limit": 10
}
```

---

### Sandbox Credit (Testing Only)

Credit a virtual account in sandbox/test mode. Only available when using Korapay test API keys.

**Roles:** Admin only

```
POST /v1/wallets/sandbox/credit
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "account_number": "1234567890",
  "amount": 1000000,
  "currency": "NGN"
}
```

| Field            | Type   | Required | Description                                   |
|------------------|--------|----------|-----------------------------------------------|
| `account_number` | string | Yes      | Virtual account number to credit              |
| `amount`         | int    | Yes      | Amount in main currency unit (NGN, not kobo)  |
| `currency`       | string | No       | Defaults to `"NGN"`                           |

**Response (200 OK):**

```json
{
  "status": true,
  "message": "Virtual bank account credited successfully"
}
```

---

### Generate File Upload URL

Generate a pre-signed URL for uploading KYC documents.

```
POST /v1/wallets/files/generate-upload-url
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "reference": "doc-unique-ref-001",
  "purpose": "kyc_document",
  "content_type": "image/jpeg"
}
```

| Field          | Type   | Required | Description                                        |
|----------------|--------|----------|----------------------------------------------------|
| `reference`    | string | Yes      | Your unique reference for the file                 |
| `purpose`      | string | Yes      | e.g., `"kyc_document"`, `"proof_of_address"`       |
| `content_type` | string | Yes      | MIME type: `"image/jpeg"`, `"application/pdf"`, `"image/png"` |

**Response (200 OK):** Returns the upload URL and file reference.

---

## 5. Transfers

The provider-agnostic transfer API uses Korapay as the primary payment provider. These endpoints allow sending money from the business wallet to any bank account.

### Single Transfer

```
POST /v1/transfers
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "amount": "50000",
  "bank_code": "058",
  "account_number": "0123456789",
  "account_name": "John Doe",
  "narration": "February salary payment"
}
```

| Field            | Type   | Required | Description                                    |
|------------------|--------|----------|------------------------------------------------|
| `amount`         | string | Yes      | Amount to transfer (in main currency unit)     |
| `bank_code`      | string | Yes      | Destination bank code                          |
| `account_number` | string | Yes      | Destination account number                     |
| `account_name`   | string | Yes      | Destination account holder name                |
| `narration`      | string | No       | Transfer description (defaults to "Transfer")  |
| `reference`      | string | No       | Custom reference (auto-generated if omitted)   |

**Response (200 OK):**

```json
{
  "success": true,
  "transfer_id": 1,
  "reference": "TRF-20260221-abc123",
  "transaction_id": "KPY-txn-12345",
  "status": "success",
  "message": "Transfer completed successfully",
  "provider": "korapay",
  "fee": "50",
  "processing_time": "2.5s"
}
```

---

### Batch Transfer

Send up to 100 transfers in a single request.

```
POST /v1/transfers/batch
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**

```json
{
  "transfers": [
    {
      "amount": "50000",
      "bank_code": "058",
      "account_number": "0123456789",
      "account_name": "John Doe",
      "narration": "February salary"
    },
    {
      "amount": "45000",
      "bank_code": "044",
      "account_number": "9876543210",
      "account_name": "Jane Smith",
      "narration": "February salary"
    }
  ]
}
```

| Field       | Type  | Required | Validation                                |
|-------------|-------|----------|-------------------------------------------|
| `transfers` | array | Yes      | Min 1, Max 100 transfer objects           |

**Response (200 OK):**

```json
{
  "total_transfers": 2,
  "successful_transfers": 2,
  "failed_transfers": 0,
  "transfers": [
    {
      "success": true,
      "transfer_id": 2,
      "reference": "TRF-20260221-def456",
      "status": "success",
      "provider": "korapay",
      "processing_time": "1.8s"
    },
    {
      "success": true,
      "transfer_id": 3,
      "reference": "TRF-20260221-ghi789",
      "status": "success",
      "provider": "korapay",
      "processing_time": "2.1s"
    }
  ],
  "processing_time": "4.2s"
}
```

---

### List Transfers

```
GET /v1/transfers?page=1&limit=20
Authorization: Bearer <token>
```

| Query Param | Type | Default | Description              |
|-------------|------|---------|--------------------------|
| `page`      | int  | 1       | Page number              |
| `limit`     | int  | 20      | Items per page (max 100) |

**Response (200 OK):**

```json
{
  "transfers": [
    {
      "id": 1,
      "reference": "TRF-20260221-abc123",
      "amount": "50000",
      "status": "success",
      "..."
    }
  ],
  "total": 15,
  "page": 1,
  "limit": 20
}
```

---

### Get Transfer by ID

```
GET /v1/transfers/{id}
Authorization: Bearer <token>
```

**Response (200 OK):** Returns the full transfer object with all details.

---

## 6. Employee Management (Full CRUD)

All employee endpoints require **Admin** or **Operator** role.

| Method  | Endpoint                                | Description             |
|---------|-----------------------------------------|-------------------------|
| POST    | `/v1/employees`                         | Create employee         |
| GET     | `/v1/employees`                         | List all employees      |
| GET     | `/v1/employees/{employeeID}`            | Get employee by ID      |
| PUT     | `/v1/employees/{employeeID}`            | Update employee         |
| PATCH   | `/v1/employees/{employeeID}/deactivate` | Deactivate employee     |

### Update Employee

```
PUT /v1/employees/{employeeID}
Content-Type: application/json
Authorization: Bearer <token>
```

```json
{
  "full_name": "John Doe Jr.",
  "email": "john.doe@acmecorp.com",
  "cadre_id": 2,
  "bank_name": "First Bank",
  "bank_code": "011",
  "bank_account_number": "1234567890",
  "is_active": true
}
```

All fields are optional for update -- only provide the fields you want to change.

### Deactivate Employee

```
PATCH /v1/employees/{employeeID}/deactivate
Authorization: Bearer <token>
```

**Response: 204 No Content** (no body)

> Deactivated employees are excluded from future payroll runs.

---

## 7. Cadre Management (Full CRUD)

Cadre endpoints require **Admin** or **Operator** role.

| Method | Endpoint                  | Description        |
|--------|---------------------------|--------------------|
| POST   | `/v1/cadres`              | Create cadre       |
| GET    | `/v1/cadres`              | List all cadres    |
| GET    | `/v1/cadres/{cadreID}`    | Get cadre by ID    |
| PUT    | `/v1/cadres/{cadreID}`    | Update cadre       |
| DELETE | `/v1/cadres/{cadreID}`    | Delete cadre       |

### Update Cadre

```
PUT /v1/cadres/{cadreID}
Content-Type: application/json
Authorization: Bearer <token>
```

```json
{
  "name": "Senior Engineer (Updated)",
  "earning_components": [
    { "name": "Basic Salary", "amount": 550000 },
    { "name": "Housing", "amount": 175000 },
    { "name": "Transport", "amount": 60000 }
  ],
  "deduction_rule_ids": [1, 2]
}
```

### Delete Cadre

```
DELETE /v1/cadres/{cadreID}
Authorization: Bearer <token>
```

**Response: 204 No Content** (no body)

---

## 8. Deduction Rule Management (Full CRUD)

Deduction rule endpoints require **Admin** role.

| Method | Endpoint                        | Description            |
|--------|---------------------------------|------------------------|
| POST   | `/v1/deduction-rules`           | Create deduction rule  |
| GET    | `/v1/deduction-rules`           | List all rules         |
| PUT    | `/v1/deduction-rules/{ruleID}`  | Update rule            |
| DELETE | `/v1/deduction-rules/{ruleID}`  | Delete rule            |

### Update Deduction Rule

```
PUT /v1/deduction-rules/{ruleID}
Content-Type: application/json
Authorization: Bearer <token>
```

```json
{
  "name": "Tax (Updated)",
  "type": "percentage",
  "value": 10.0,
  "calculation_basis": "gross_pay"
}
```

### Delete Deduction Rule

```
DELETE /v1/deduction-rules/{ruleID}
Authorization: Bearer <token>
```

**Response: 204 No Content** (no body)

---

## 9. Payroll Runs (Full CRUD)

| Method | Endpoint                                   | Roles                      | Description                  |
|--------|--------------------------------------------|----------------------------|------------------------------|
| POST   | `/v1/payroll-runs`                         | Admin, Operator            | Create payroll run           |
| GET    | `/v1/payroll-runs`                         | Admin, Operator, Approver  | List payroll runs            |
| GET    | `/v1/payroll-runs/{runID}`                 | Admin, Operator, Approver  | Get payroll run details      |
| POST   | `/v1/payroll-runs/{runID}/submit`          | Admin, Operator            | Submit for approval          |
| POST   | `/v1/payroll-runs/{runID}/approve`         | Admin, Approver            | Approve payroll              |
| POST   | `/v1/payroll-runs/{runID}/reject`          | Admin, Approver            | Reject payroll               |
| POST   | `/v1/payroll-runs/{runID}/process-now`     | Admin, Operator            | Process immediately          |

### Payroll Status Values

| Status             | Description                                            |
|--------------------|--------------------------------------------------------|
| `draft`            | Initial state. Payroll has been created and calculated |
| `pending_approval` | Submitted and waiting for an approver to review        |
| `approved`         | Approved and ready for processing/disbursement         |
| `processing`       | Bank transfers are currently being executed             |
| `completed`        | All transfers completed successfully                   |
| `rejected`         | Rejected by an approver with a reason                  |
| `failed`           | Transfer processing encountered errors                 |

---

## 10. Error Handling

All error responses follow this format:

```json
{
  "error": "Human-readable error message",
  "details": {}
}
```

The `details` field is optional and only present for certain error types (e.g., transfer amount validation).

### HTTP Status Codes

| Status Code | Error Constant            | Meaning                                              |
|-------------|---------------------------|------------------------------------------------------|
| `400`       | `validation failed`       | Request body failed validation (missing/invalid fields) |
| `401`       | `unauthorized`            | Missing, invalid, or expired JWT token               |
| `403`       | `forbidden: insufficient permissions` | User's role does not have access to this endpoint |
| `404`       | `resource not found`      | The requested resource (employee, cadre, payroll run, etc.) does not exist |
| `409`       | `resource conflict or duplicate` | Trying to create a resource that already exists (e.g., duplicate email) |
| `500`       | `internal server error`   | Unexpected server error (details are not exposed to clients) |
| `502`       | `payment gateway operation failed` | The external payment provider (Korapay) returned an error |

### Transfer Amount Errors (400)

Transfer amount validation errors include additional details:

```json
{
  "error": "transfer amount 50 is below the minimum allowed amount of 100",
  "details": {
    "amount": "50",
    "min_amount": "100",
    "max_amount": "0"
  }
}
```

### Common Error Scenarios

| Scenario                                  | Status | Error Message                         |
|-------------------------------------------|--------|---------------------------------------|
| Missing `Authorization` header            | 401    | `unauthorized`                        |
| Token with `Bearer` prefix missing        | 401    | `unauthorized`                        |
| Expired JWT token                         | 401    | `unauthorized`                        |
| Operator tries to create deduction rule   | 403    | `forbidden: insufficient permissions` |
| Approver tries to create employee         | 403    | `forbidden: insufficient permissions` |
| Get employee with non-existent ID         | 404    | `resource not found`                  |
| Register with already-used email          | 409    | `resource conflict or duplicate`      |
| Invalid JSON in request body              | 400    | `validation failed`                   |
| Missing required field                    | 400    | `validation failed`                   |
| Process payroll while Korapay is down     | 502    | `payment gateway operation failed`    |

---

## 11. Role-Based Access Control

Payflow has three user roles with different permissions:

### Admin

Full access to everything. The first user created when a business registers is always an admin.

| Resource         | Create | Read | Update | Delete | Special Actions                    |
|------------------|--------|------|--------|--------|------------------------------------|
| Deduction Rules  | Yes    | Yes  | Yes    | Yes    | -                                  |
| Cadres           | Yes    | Yes  | Yes    | Yes    | -                                  |
| Employees        | Yes    | Yes  | Yes    | Yes    | Deactivate                         |
| Payroll Runs     | Yes    | Yes  | -      | -      | Submit, Approve, Reject, Process   |
| Transfers        | Yes    | Yes  | -      | -      | Single, Batch                      |
| Wallets          | Yes    | Yes  | -      | -      | Sandbox Credit                     |

### Operator

Day-to-day operations: managing employees, cadres, and payroll creation/submission.

| Resource         | Create | Read | Update | Delete | Special Actions          |
|------------------|--------|------|--------|--------|--------------------------|
| Deduction Rules  | No     | No   | No     | No     | -                        |
| Cadres           | Yes    | Yes  | Yes    | Yes    | -                        |
| Employees        | Yes    | Yes  | Yes    | Yes    | Deactivate               |
| Payroll Runs     | Yes    | Yes  | -      | -      | Submit, Process          |
| Transfers        | Yes    | Yes  | -      | -      | Single, Batch            |
| Wallets          | Yes    | Yes  | -      | -      | -                        |

### Approver

Review and approve/reject payroll runs. Read-only access to payroll data.

| Resource         | Create | Read | Update | Delete | Special Actions          |
|------------------|--------|------|--------|--------|--------------------------|
| Deduction Rules  | No     | No   | No     | No     | -                        |
| Cadres           | No     | No   | No     | No     | -                        |
| Employees        | No     | No   | No     | No     | -                        |
| Payroll Runs     | No     | Yes  | -      | -      | Approve, Reject          |
| Transfers        | Yes    | Yes  | -      | -      | Single, Batch            |
| Wallets          | Yes    | Yes  | -      | -      | -                        |

### Quick Reference: Endpoint Permissions

```
Public (no auth):
  POST /v1/auth/register
  POST /v1/auth/login
  GET  /health

Admin only:
  POST/GET/PUT/DELETE /v1/deduction-rules
  POST /v1/wallets/sandbox/credit

Admin + Operator:
  POST/GET/PUT/DELETE /v1/cadres
  POST/GET/PUT/PATCH  /v1/employees
  POST   /v1/payroll-runs
  POST   /v1/payroll-runs/{id}/submit
  POST   /v1/payroll-runs/{id}/process-now

Admin + Approver:
  POST /v1/payroll-runs/{id}/approve
  POST /v1/payroll-runs/{id}/reject

Admin + Operator + Approver:
  GET /v1/payroll-runs
  GET /v1/payroll-runs/{id}

All Authenticated:
  POST/GET /v1/transfers
  POST     /v1/transfers/batch
  GET      /v1/transfers/{id}
  POST     /v1/wallets/virtual-account
  GET      /v1/wallets
  GET      /v1/wallets/balance
  GET      /v1/wallets/transactions
  POST     /v1/wallets/account-holders
  GET      /v1/wallets/account-holders/{ref}/details
  PATCH    /v1/wallets/account-holders/{ref}/update-kyc
  POST     /v1/wallets/files/generate-upload-url
```

---

## 12. Testing Order

Follow this order to build up data dependencies correctly when testing the API from scratch:

### Phase 1: Setup

1. **Register Business** -- `POST /v1/auth/register`
2. **Login** -- `POST /v1/auth/login` (save the token)
3. **Health Check** -- `GET /health` (verify server is running)

### Phase 2: Configuration

4. **Create Deduction Rules** -- `POST /v1/deduction-rules`
   - Create at least one rule (e.g., Tax at 7.5%)
   - Optionally create more (Pension, Health Insurance, etc.)
5. **List Deduction Rules** -- `GET /v1/deduction-rules` (verify they were created)

### Phase 3: Salary Structures

6. **Create Cadres** -- `POST /v1/cadres`
   - Create at least one cadre with earning components
   - Optionally link deduction rules via `deduction_rule_ids`
7. **List Cadres** -- `GET /v1/cadres` (verify, note the cadre IDs)

### Phase 4: Employees

8. **Create Employees** -- `POST /v1/employees`
   - Assign each employee to a cadre using `cadre_id`
   - Include valid bank details
9. **List Employees** -- `GET /v1/employees` (verify)

### Phase 5: Payroll Lifecycle

10. **Create Payroll Run** -- `POST /v1/payroll-runs`
    - Specify the period (e.g., `"2026-02"`)
    - Optionally include adjustments
    - Verify that gross pay, deductions, and net pay are calculated correctly
11. **List Payroll Runs** -- `GET /v1/payroll-runs` (verify status is `"draft"`)
12. **Get Payroll Run Details** -- `GET /v1/payroll-runs/{id}` (inspect entries)
13. **Submit for Approval** -- `POST /v1/payroll-runs/{id}/submit`
14. **Approve Payroll** -- `POST /v1/payroll-runs/{id}/approve`
15. **Process Payroll** -- `POST /v1/payroll-runs/{id}/process-now`

### Phase 6: Wallet Setup (Optional -- for Korapay integration)

16. **Create Account Holder** -- `POST /v1/wallets/account-holders`
17. **Create Virtual Account** -- `POST /v1/wallets/virtual-account`
18. **Check Balance** -- `GET /v1/wallets/balance`
19. **Sandbox Credit** -- `POST /v1/wallets/sandbox/credit` (test mode only)
20. **Get Transactions** -- `GET /v1/wallets/transactions`

### Phase 7: Transfers (Optional -- for direct transfers)

21. **Single Transfer** -- `POST /v1/transfers`
22. **Batch Transfer** -- `POST /v1/transfers/batch`
23. **List Transfers** -- `GET /v1/transfers`
24. **Get Transfer** -- `GET /v1/transfers/{id}`

### Alternative Flows to Test

- **Reject payroll** -- After step 13, use `POST /v1/payroll-runs/{id}/reject` instead of approving
- **Update employee** -- After step 8, use `PUT /v1/employees/{id}`
- **Deactivate employee** -- Use `PATCH /v1/employees/{id}/deactivate`, then run a new payroll to confirm exclusion
- **Update cadre** -- Modify earning components with `PUT /v1/cadres/{id}`, then create a new payroll to see updated amounts

---

## 13. Health Check

Verify the server and database are operational.

```
GET /health
```

**No authentication required.**

**Response (200 OK):**

```json
{
  "status": "healthy",
  "message": "Server is running. All systems operational.",
  "server": "ok",
  "database": {
    "status": "ok",
    "message": "connected"
  }
}
```

**Response (503 Service Unavailable):**

```json
{
  "status": "unhealthy",
  "message": "Server is running but database is unreachable.",
  "server": "ok",
  "database": {
    "status": "unavailable",
    "message": "connection refused"
  }
}
```

---

## Quick cURL Reference

Here is a complete set of copy-paste-ready cURL commands for the core flow:

```bash
# 1. Register
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "business_name": "Acme Corp",
    "email": "admin@acmecorp.com",
    "password": "SecurePassword123",
    "rc_number": "RC123456",
    "incorporation_date": "2020-01-15T00:00:00Z",
    "director_bvn": "12345678901"
  }'

# 2. Login (save the token)
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@acmecorp.com",
    "password": "SecurePassword123"
  }'

# Set token variable (replace with actual token from login response)
TOKEN="eyJhbG..."

# 3. Create Deduction Rule
curl -X POST http://localhost:8080/v1/deduction-rules \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Tax",
    "type": "percentage",
    "value": 7.5,
    "calculation_basis": "gross_pay"
  }'

# 4. Create Cadre
curl -X POST http://localhost:8080/v1/cadres \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Senior Engineer",
    "earning_components": [
      { "name": "Basic Salary", "amount": 500000 },
      { "name": "Housing", "amount": 150000 },
      { "name": "Transport", "amount": 50000 }
    ]
  }'

# 5. Create Employee
curl -X POST http://localhost:8080/v1/employees \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "full_name": "John Doe",
    "email": "john@acmecorp.com",
    "cadre_id": 1,
    "bank_name": "GTBank",
    "bank_code": "058",
    "bank_account_number": "0123456789"
  }'

# 6. Create Payroll Run
curl -X POST http://localhost:8080/v1/payroll-runs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "period": "2026-02"
  }'

# 7. Submit for Approval
curl -X POST http://localhost:8080/v1/payroll-runs/1/submit \
  -H "Authorization: Bearer $TOKEN"

# 8. Approve Payroll
curl -X POST http://localhost:8080/v1/payroll-runs/1/approve \
  -H "Authorization: Bearer $TOKEN"

# 9. Process Payroll
curl -X POST http://localhost:8080/v1/payroll-runs/1/process-now \
  -H "Authorization: Bearer $TOKEN"

# 10. Check Balance
curl http://localhost:8080/v1/wallets/balance \
  -H "Authorization: Bearer $TOKEN"

# 11. Single Transfer
curl -X POST http://localhost:8080/v1/transfers \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "amount": "50000",
    "bank_code": "058",
    "account_number": "0123456789",
    "account_name": "John Doe",
    "narration": "Salary payment"
  }'
```
