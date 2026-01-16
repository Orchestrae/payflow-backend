# PayFlow API Documentation

This document provides a comprehensive overview of all PayFlow API endpoints, their functionality, expected payloads, and sample responses. The base URL for all endpoints is `/v1`.

## **Table of Contents**

1. [Authentication](#1-authentication)
2. [Employee Management](#2-employee-management)
3. [Cadre (Salary Structure) Management](#3-cadre-salary-structure-management)
4. [Deduction Rule Management](#4-deduction-rule-management)
5. [Payroll Workflow](#5-payroll-workflow)
6. [Transfers](#6-transfers)
7. [VFD Integration](#7-vfd-integration)
8. [Webhooks](#8-webhooks)
9. [Background Jobs & Scheduler](#9-background-jobs--scheduler)

---

## **1. Authentication**

These endpoints manage user and business onboarding and access to the platform.

### `POST /v1/auth/register`

**Description:** Creates a new Business and its primary Admin user in a single, atomic transaction. This is the first step for any new company joining PayFlow.

**Authorization:** Public (No token required)

**Request Body:**
```json
{
  "business_name": "Innovate Inc.",
  "email": "admin@innovate.com",
  "password": "a-very-strong-password",
  "rc_number": "RC123456",
  "incorporation_date": "2020-01-15T00:00:00Z",
  "director_bvn": "12345678901"
}
```

**Response (201 Created):**
```json
{
  "user": {
    "id": 1,
    "email": "admin@innovate.com",
    "role": "admin",
    "business_id": 1
  },
  "corporate_account": {
    "account_number": "1234567890",
    "account_name": "Innovate Inc."
  }
}
```

---

### `POST /v1/auth/login`

**Description:** Authenticates a user and returns a JSON Web Token (JWT) for accessing protected endpoints.

**Authorization:** Public

**Request Body:**
```json
{
  "email": "admin@innovate.com",
  "password": "a-very-strong-password"
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "email": "admin@innovate.com",
    "role": "admin",
    "business_id": 1
  }
}
```

**Note:** Include the token in the `Authorization: Bearer <token>` header for all subsequent protected requests.

---

## **2. Employee Management**

Endpoints for managing employee records within a business.

### `POST /v1/employees`

**Description:** Adds a new employee to the business. The service validates that the specified `cadre_id` exists and belongs to the same business before creating the employee record.

**Authorization:** `Admin` or `Operator` role required

**Request Body:**
```json
{
  "cadre_id": 1,
  "full_name": "Jane Doe",
  "email": "jane.doe@innovate.com",
  "bank_name": "Access Bank",
  "bank_account_number": "1234567890"
}
```

**Response (201 Created):**
```json
{
  "ID": 1,
  "CreatedAt": "2026-01-15T10:00:00Z",
  "UpdatedAt": "2026-01-15T10:00:00Z",
  "DeletedAt": null,
  "BusinessID": 1,
  "CadreID": 1,
  "FullName": "Jane Doe",
  "Email": "jane.doe@innovate.com",
  "BankName": "Access Bank",
  "BankAccountNumber": "1234567890",
  "IsActive": true,
  "Cadre": {
    "ID": 1,
    "Name": "Senior Software Engineer",
    "EarningComponents": [...],
    "DeductionRules": [...]
  }
}
```

---

### `GET /v1/employees`

**Description:** Retrieves a list of all employees associated with the authenticated user's business.

**Authorization:** `Admin` or `Operator` role required

**Response (200 OK):**
```json
[
  {
    "ID": 1,
    "BusinessID": 1,
    "CadreID": 1,
    "FullName": "Jane Doe",
    "Email": "jane.doe@innovate.com",
    "BankName": "Access Bank",
    "BankAccountNumber": "1234567890",
    "IsActive": true,
    "Cadre": {...}
  },
  {
    "ID": 2,
    "BusinessID": 1,
    "CadreID": 2,
    "FullName": "John Smith",
    "Email": "john.smith@innovate.com",
    "BankName": "GTBank",
    "BankAccountNumber": "0987654321",
    "IsActive": true,
    "Cadre": {...}
  }
]
```

---

### `GET /v1/employees/{employeeID}`

**Description:** Fetches the details of a single employee. The service performs a security check to ensure the employee's `business_id` matches the one in the user's JWT.

**Authorization:** `Admin` or `Operator` role required

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "CadreID": 1,
  "FullName": "Jane Doe",
  "Email": "jane.doe@innovate.com",
  "BankName": "Access Bank",
  "BankAccountNumber": "1234567890",
  "IsActive": true,
  "Cadre": {
    "ID": 1,
    "Name": "Senior Software Engineer",
    "EarningComponents": [
      {
        "ID": 1,
        "Name": "Basic Salary",
        "Amount": 500000
      },
      {
        "ID": 2,
        "Name": "Housing Allowance",
        "Amount": 150000
      }
    ],
    "DeductionRules": [...]
  }
}
```

---

### `PUT /v1/employees/{employeeID}`

**Description:** Updates an existing employee's information. All fields are optional - only provided fields will be updated.

**Authorization:** `Admin` or `Operator` role required

**Request Body:**
```json
{
  "cadre_id": 2,
  "full_name": "Jane Doe Updated",
  "email": "jane.updated@innovate.com",
  "bank_name": "GTBank",
  "bank_account_number": "9876543210",
  "is_active": true
}
```

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "CadreID": 2,
  "FullName": "Jane Doe Updated",
  "Email": "jane.updated@innovate.com",
  "BankName": "GTBank",
  "BankAccountNumber": "9876543210",
  "IsActive": true
}
```

---

### `PATCH /v1/employees/{employeeID}/deactivate`

**Description:** Deactivates an employee, marking them as inactive. This prevents them from being included in future payroll runs.

**Authorization:** `Admin` or `Operator` role required

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "CadreID": 1,
  "FullName": "Jane Doe",
  "Email": "jane.doe@innovate.com",
  "IsActive": false
}
```

---

## **3. Cadre (Salary Structure) Management**

Endpoints for defining standardized salary structures of the business.

### `POST /v1/cadres`

**Description:** Creates a new salary structure (e.g., "Senior Engineer," "Marketing Manager"). Defines a cadre with a name, earning components, and links to deduction rules.

**Authorization:** `Admin` or `Operator` role required

**Request Body:**
```json
{
  "name": "Senior Software Engineer",
  "earning_components": [
    {
      "name": "Basic Salary",
      "amount": 500000
    },
    {
      "name": "Housing Allowance",
      "amount": 150000
    }
  ],
  "deduction_rule_ids": [1, 2]
}
```

**Response (201 Created):**
```json
{
  "ID": 1,
  "CreatedAt": "2026-01-15T10:00:00Z",
  "UpdatedAt": "2026-01-15T10:00:00Z",
  "BusinessID": 1,
  "Name": "Senior Software Engineer",
  "EarningComponents": [
    {
      "ID": 1,
      "CadreID": 1,
      "Name": "Basic Salary",
      "Amount": 500000
    },
    {
      "ID": 2,
      "CadreID": 1,
      "Name": "Housing Allowance",
      "Amount": 150000
    }
  ],
  "DeductionRules": [
    {
      "ID": 1,
      "Name": "Pension Contribution",
      "Type": "percentage",
      "Value": 8.0,
      "CalculationBasis": "gross_pay"
    }
  ]
}
```

---

### `GET /v1/cadres`

**Description:** Lists all salary cadres configured for the business.

**Authorization:** `Admin` or `Operator` role required

**Response (200 OK):**
```json
[
  {
    "ID": 1,
    "BusinessID": 1,
    "Name": "Senior Software Engineer",
    "EarningComponents": [...],
    "DeductionRules": [...]
  },
  {
    "ID": 2,
    "BusinessID": 1,
    "Name": "Marketing Manager",
    "EarningComponents": [...],
    "DeductionRules": [...]
  }
]
```

---

### `GET /v1/cadres/{cadreID}`

**Description:** Fetches the details of a specific cadre including all earning components and deduction rules.

**Authorization:** `Admin` or `Operator` role required

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "Name": "Senior Software Engineer",
  "EarningComponents": [
    {
      "ID": 1,
      "CadreID": 1,
      "Name": "Basic Salary",
      "Amount": 500000
    },
    {
      "ID": 2,
      "CadreID": 1,
      "Name": "Housing Allowance",
      "Amount": 150000
    }
  ],
  "DeductionRules": [
    {
      "ID": 1,
      "BusinessID": 1,
      "CadreID": 1,
      "Name": "Pension Contribution",
      "Type": "percentage",
      "Value": 8.0,
      "CalculationBasis": "gross_pay"
    }
  ]
}
```

---

### `PUT /v1/cadres/{cadreID}`

**Description:** Updates an existing cadre's name, earning components, and deduction rules.

**Authorization:** `Admin` or `Operator` role required

**Request Body:**
```json
{
  "name": "Senior Software Engineer (Updated)",
  "earning_components": [
    {
      "name": "Basic Salary",
      "amount": 600000
    },
    {
      "name": "Housing Allowance",
      "amount": 200000
    }
  ],
  "deduction_rule_ids": [1, 2, 3]
}
```

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "Name": "Senior Software Engineer (Updated)",
  "EarningComponents": [...],
  "DeductionRules": [...]
}
```

---

### `DELETE /v1/cadres/{cadreID}`

**Description:** Deletes a cadre. Note: This will fail if there are employees still assigned to this cadre.

**Authorization:** `Admin` or `Operator` role required

**Response (200 OK):**
```json
{
  "message": "Cadre deleted successfully"
}
```

---

## **4. Deduction Rule Management**

Endpoints for managing deduction rules that can be applied to cadres.

### `POST /v1/deduction-rules`

**Description:** Creates a new deduction rule that can be applied to cadres. Supports percentage-based or flat amount deductions.

**Authorization:** `Admin` role required

**Request Body:**
```json
{
  "name": "Pension Contribution",
  "type": "percentage",
  "value": 8.0,
  "calculation_basis": "gross_pay"
}
```

**Valid Values:**
- `type`: `"percentage"` or `"flat"`
- `calculation_basis`: `"gross_pay"` or `"basic_pay"`

**Response (201 Created):**
```json
{
  "ID": 1,
  "CreatedAt": "2026-01-15T10:00:00Z",
  "UpdatedAt": "2026-01-15T10:00:00Z",
  "BusinessID": 1,
  "CadreID": 0,
  "Name": "Pension Contribution",
  "Type": "percentage",
  "Value": 8.0,
  "CalculationBasis": "gross_pay"
}
```

---

### `GET /v1/deduction-rules`

**Description:** Lists all deduction rules for the business.

**Authorization:** `Admin` role required

**Response (200 OK):**
```json
[
  {
    "ID": 1,
    "BusinessID": 1,
    "Name": "Pension Contribution",
    "Type": "percentage",
    "Value": 8.0,
    "CalculationBasis": "gross_pay"
  },
  {
    "ID": 2,
    "BusinessID": 1,
    "Name": "Health Insurance",
    "Type": "flat",
    "Value": 5000,
    "CalculationBasis": "gross_pay"
  }
]
```

---

### `PUT /v1/deduction-rules/{ruleID}`

**Description:** Updates an existing deduction rule.

**Authorization:** `Admin` role required

**Request Body:**
```json
{
  "name": "Pension Contribution (Updated)",
  "type": "percentage",
  "value": 10.0,
  "calculation_basis": "gross_pay"
}
```

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "Name": "Pension Contribution (Updated)",
  "Type": "percentage",
  "Value": 10.0,
  "CalculationBasis": "gross_pay"
}
```

---

### `DELETE /v1/deduction-rules/{ruleID}`

**Description:** Deletes a deduction rule.

**Authorization:** `Admin` role required

**Response (200 OK):**
```json
{
  "message": "Deduction rule deleted successfully"
}
```

---

## **5. Payroll Workflow**

These endpoints orchestrate the end-to-end process of running payroll.

### `POST /v1/payroll-runs`

**Description:** Initiates a new payroll run for a specified period. Calculates gross pay, deductions, and net pay for all active employees.

**Authorization:** `Admin` or `Operator` role required

**Request Body:**
```json
{
  "period": "2026-01",
  "adjustments": {
    "user1": [
      {
        "item_name": "bonus",
        "amount": 100000,
        "description": "A job well done",
        "component_type": "earnings"
      },
      {
        "item_name": "penalty",
        "amount": -50000,
        "description": "Late submission",
        "component_type": "deduction"
      }
    ],
    "jane.doe@innovate.com": [
      {
        "item_name": "performance_bonus",
        "amount": 200000,
        "description": "Outstanding Q4 performance"
      }
    ]
  }
}
```

**Request Fields:**
- `period` (optional): Payroll period in format `"YYYY-MM"` or `"YYYY-MM-DD"`. Defaults to current month if not provided.
- `adjustments` (optional): Map where:
  - **Key**: Employee identifier (can be employee ID as string or email address)
  - **Value**: Array of adjustment items for that employee

**Adjustment Item Fields:**
- `item_name` (required): Name of the adjustment item (e.g., "bonus", "penalty", "performance_bonus")
- `amount` (required): Adjustment amount. Positive values are earnings/bonuses, negative values are deductions
- `description` (optional): Description for historical tracking (e.g., "A job well done", "Late submission")
- `component_type` (optional): Either `"earnings"` or `"deduction"`. If not provided, it's auto-inferred from the amount sign (positive = earnings, negative = deduction)

**Note:** This structure allows multiple adjustments per employee with detailed tracking for historical purposes. Each adjustment is stored in the payroll run entry details with its description.

**Response (201 Created):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "Period": "2026-01-01T00:00:00Z",
  "Status": "draft",
  "TotalGrossPay": 5000000,
  "TotalDeductions": 500000,
  "TotalNetPay": 4500000,
  "ScheduledFor": "2026-01-25T09:00:00Z",
  "ProcessedAt": null,
  "PaymentReference": "",
  "RejectionReason": "",
  "Entries": [
    {
      "ID": 1,
      "PayrollRunID": 1,
      "EmployeeID": 1,
      "GrossPay": 650000,
      "TotalDeductions": 52000,
      "Bonuses": 0,
      "NetPay": 598000,
      "Employee": {
        "ID": 1,
        "FullName": "Jane Doe",
        "Email": "jane.doe@innovate.com"
      },
      "Details": [
        {
          "ID": 1,
          "Type": "earning",
          "Name": "Basic Salary",
          "Amount": 500000,
          "Description": ""
        },
        {
          "ID": 2,
          "Type": "deduction",
          "Name": "Pension Contribution",
          "Amount": 52000,
          "Description": ""
        },
        {
          "ID": 3,
          "Type": "earning",
          "Name": "bonus",
          "Amount": 100000,
          "Description": "A job well done"
        },
        {
          "ID": 4,
          "Type": "deduction",
          "Name": "penalty",
          "Amount": 50000,
          "Description": "Late submission"
        }
      ]
    }
  ]
}
```

---

### `GET /v1/payroll-runs`

**Description:** Lists all payroll runs for the business, ordered by creation date (newest first).

**Authorization:** `Admin` or `Operator` role required

**Response (200 OK):**
```json
[
  {
    "ID": 1,
    "BusinessID": 1,
    "Period": "2026-01-01T00:00:00Z",
    "Status": "draft",
    "TotalGrossPay": 5000000,
    "TotalDeductions": 500000,
    "TotalNetPay": 4500000,
    "ScheduledFor": "2026-01-25T09:00:00Z",
    "PaymentReference": "",
    "Entries": [...]
  }
]
```

---

### `GET /v1/payroll-runs/{runID}`

**Description:** Fetches the details of a specific payroll run including all entries and employee details.

**Authorization:** `Admin` or `Operator` role required

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "Period": "2026-01-01T00:00:00Z",
  "Status": "draft",
  "TotalGrossPay": 5000000,
  "TotalDeductions": 500000,
  "TotalNetPay": 4500000,
  "ScheduledFor": "2026-01-25T09:00:00Z",
  "ProcessedAt": null,
  "PaymentReference": "",
  "Entries": [...]
}
```

---

### `POST /v1/payroll-runs/{runID}/submit`

**Description:** Submits a `draft` payroll run for review and approval. Changes status from `draft` to `pending_approval`. If business has `payroll_requires_approval` set to `false`, it auto-approves.

**Authorization:** `Admin` or `Operator` role required

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "Status": "pending_approval",
  "TotalGrossPay": 5000000,
  "TotalDeductions": 500000,
  "TotalNetPay": 4500000,
  "ScheduledFor": "2026-01-25T09:00:00Z"
}
```

---

### `POST /v1/payroll-runs/{runID}/approve`

**Description:** Provides the final sign-off for a payroll run, scheduling it for payment. Changes status to `approved` and schedules a job for the `ScheduledFor` date. If business has `payroll_auto_process` enabled, processes immediately.

**Authorization:** `Admin` or `Approver` role required

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "Status": "approved",
  "TotalGrossPay": 5000000,
  "TotalDeductions": 500000,
  "TotalNetPay": 4500000,
  "ScheduledFor": "2026-01-25T09:00:00Z",
  "PaymentReference": ""
}
```

---

### `POST /v1/payroll-runs/{runID}/reject`

**Description:** Rejects a payroll run, sending it back to the Operator for corrections. Changes status to `rejected` and saves the rejection reason.

**Authorization:** `Admin` or `Approver` role required

**Request Body:**
```json
{
  "reason": "Bonus calculation for the sales team is incorrect. Please revise."
}
```

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "Status": "rejected",
  "RejectionReason": "Bonus calculation for the sales team is incorrect. Please revise.",
  "TotalGrossPay": 5000000,
  "TotalDeductions": 500000,
  "TotalNetPay": 4500000
}
```

---

### `POST /v1/payroll-runs/{runID}/process-now`

**Description:** Processes a payroll run immediately, bypassing the scheduler. Allows businesses to pay employees instantly without waiting for scheduled processing. Executes bulk transfers and verifies them in the database.

**Authorization:** `Admin` or `Operator` role required

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "Status": "completed",
  "TotalGrossPay": 5000000,
  "TotalDeductions": 500000,
  "TotalNetPay": 4500000,
  "ScheduledFor": "2026-01-25T09:00:00Z",
  "ProcessedAt": "2026-01-15T10:30:00Z",
  "PaymentReference": "PAYROLL-RUN-1",
  "Entries": [...]
}
```

---

## **6. Transfers**

Provider-agnostic transfer endpoints supporting Korapay (primary) and VFD (fallback).

### `POST /v1/transfers`

**Description:** Initiates a single transfer to a bank account. Reference is auto-generated if not provided. Uses configured default provider (Korapay) with VFD as fallback.

**Authorization:** Required (any authenticated user)

**Request Body:**
```json
{
  "amount": "1000",
  "bank_code": "044",
  "account_number": "1234567890",
  "account_name": "John Doe",
  "narration": "Payment for services",
  "reference": "TXN-001"
}
```

**Request Fields:**
- `amount` (required): Transfer amount as string
- `bank_code` (required): Bank code (e.g., "044" for Access Bank)
- `account_number` (required): Recipient account number (must be 10 characters for Korapay)
- `account_name` (required): Recipient account name
- `narration` (optional): Transfer description (defaults to "Transfer")
- `reference` (optional): Custom reference (auto-generated if not provided)

**Response (200 OK):**
```json
{
  "success": true,
  "transfer_id": 1,
  "reference": "TXN-001",
  "transaction_id": "korapay_txn_123",
  "status": "processing",
  "message": "Transfer initiated successfully",
  "provider": "korapay",
  "fee": "30.00",
  "processing_time": "245.5ms",
  "error": null
}
```

---

### `POST /v1/transfers/batch`

**Description:** Initiates a batch of transfers. Uses native bulk API if provider supports it (Korapay), otherwise processes concurrently.

**Authorization:** Required (any authenticated user)

**Request Body:**
```json
{
  "transfers": [
    {
      "amount": "1000",
      "bank_code": "044",
      "account_number": "1234567890",
      "account_name": "John Doe",
      "narration": "Payment 1"
    },
    {
      "amount": "2000",
      "bank_code": "033",
      "account_number": "0987654321",
      "account_name": "Jane Smith",
      "narration": "Payment 2"
    }
  ]
}
```

**Response (200 OK):**
```json
{
  "total_transfers": 2,
  "successful_transfers": 2,
  "failed_transfers": 0,
  "transfers": [
    {
      "success": true,
      "transfer_id": 1,
      "reference": "PAYROLL-1-EMP-1",
      "status": "processing",
      "provider": "korapay",
      "processing_time": "150ms"
    },
    {
      "success": true,
      "transfer_id": 2,
      "reference": "PAYROLL-1-EMP-2",
      "status": "processing",
      "provider": "korapay",
      "processing_time": "180ms"
    }
  ],
  "processing_time": "350ms"
}
```

---

### `GET /v1/transfers`

**Description:** Lists all transfers for the authenticated user's business with pagination.

**Authorization:** Required (any authenticated user)

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20)

**Response (200 OK):**
```json
{
  "page": 1,
  "limit": 20,
  "total": 50,
  "transfers": [
    {
      "ID": 1,
      "CreatedAt": "2026-01-15T10:00:00Z",
      "UpdatedAt": "2026-01-15T10:00:00Z",
      "BusinessID": 1,
      "Reference": "TXN-001",
      "Amount": "1000",
      "Currency": "NGN",
      "Narration": "Payment for services",
      "RecipientBankCode": "044",
      "RecipientAccountNumber": "1234567890",
      "RecipientAccountName": "John Doe",
      "Provider": "korapay",
      "Status": "completed",
      "TransactionID": "korapay_txn_123",
      "ProviderStatus": "success",
      "Fee": "30.00",
      "ProcessedAt": "2026-01-15T10:00:05Z"
    }
  ]
}
```

---

### `GET /v1/transfers/{id}`

**Description:** Fetches the details of a specific transfer.

**Authorization:** Required (any authenticated user)

**Response (200 OK):**
```json
{
  "ID": 1,
  "BusinessID": 1,
  "Reference": "TXN-001",
  "Amount": "1000",
  "Currency": "NGN",
  "Narration": "Payment for services",
  "RecipientBankCode": "044",
  "RecipientAccountNumber": "1234567890",
  "RecipientAccountName": "John Doe",
  "Provider": "korapay",
  "Status": "completed",
  "TransactionID": "korapay_txn_123",
  "ProviderStatus": "success",
  "Fee": "30.00",
  "ProcessedAt": "2026-01-15T10:00:05Z"
}
```

---

## **7. VFD Integration**

Endpoints for VFD-specific transfer operations and account enquiries.

### `GET /v1/vfd/transfers/account-enquiry`

**Description:** Enquires about a VFD account number to get account details.

**Authorization:** Required (any authenticated user)

**Query Parameters:**
- `accountNumber` (optional): Account number to enquire

**Response (200 OK):**
```json
{
  "accountNumber": "1234567890",
  "accountName": "John Doe",
  "bankCode": "044"
}
```

---

### `GET /v1/vfd/transfers/beneficiary-enquiry`

**Description:** Enquires about a beneficiary account before transfer.

**Authorization:** Required (any authenticated user)

**Query Parameters:**
- `accountNo` (required): Account number
- `bank` (required): Bank code
- `transfer_type` (required): `"intra"` or `"inter"`

**Response (200 OK):**
```json
{
  "accountNumber": "1234567890",
  "accountName": "John Doe",
  "bankCode": "044"
}
```

---

### `GET /v1/vfd/transfers/banks`

**Description:** Retrieves the list of supported banks from VFD.

**Authorization:** Required (any authenticated user)

**Response (200 OK):**
```json
{
  "banks": [
    {
      "code": "044",
      "name": "Access Bank"
    },
    {
      "code": "033",
      "name": "United Bank for Africa"
    }
  ]
}
```

---

### `POST /v1/vfd/transfers/initiate`

**Description:** Initiates a transfer via VFD API.

**Authorization:** Required (any authenticated user)

**Request Body:**
```json
{
  "fromAccount": "1234567890",
  "fromClientId": "CLIENT001",
  "fromClient": "Business Name",
  "fromSavingsId": "SAVINGS001",
  "toClientId": "CLIENT002",
  "toClient": "Recipient Name",
  "toSavingsId": "SAVINGS002",
  "toAccount": "0987654321",
  "toBank": "044",
  "signature": "generated_signature",
  "amount": "1000",
  "remark": "Payment",
  "transferType": "inter",
  "reference": "VFD-TXN-001"
}
```

**Response (200 OK):**
```json
{
  "reference": "VFD-TXN-001",
  "status": "processing",
  "message": "Transfer initiated"
}
```

---

### `GET /v1/vfd/transfers`

**Description:** Lists all VFD transfers for the business.

**Authorization:** Required (any authenticated user)

**Response (200 OK):**
```json
[
  {
    "ID": 1,
    "Reference": "VFD-TXN-001",
    "Amount": "1000",
    "Status": "completed",
    "FromAccount": "1234567890",
    "ToAccount": "0987654321"
  }
]
```

---

### `GET /v1/vfd/transfers/{id}`

**Description:** Fetches details of a specific VFD transfer.

**Authorization:** Required (any authenticated user)

**Response (200 OK):**
```json
{
  "ID": 1,
  "Reference": "VFD-TXN-001",
  "Amount": "1000",
  "Status": "completed",
  "FromAccount": "1234567890",
  "ToAccount": "0987654321",
  "CreatedAt": "2026-01-15T10:00:00Z"
}
```

---

## **8. Webhooks**

Endpoints for managing VFD webhook notifications.

### `POST /vfd/webhooks/inward-credit`

**Description:** Public endpoint for VFD to send inward credit notifications. No authentication required.

**Request Body:**
```json
{
  "accountNumber": "1234567890",
  "amount": "1000",
  "reference": "TXN-001",
  "timestamp": "2026-01-15T10:00:00Z"
}
```

**Response (200 OK):**
```json
{
  "status": "received"
}
```

---

### `GET /v1/vfd/webhooks`

**Description:** Lists all webhook notifications for the business.

**Authorization:** Required (any authenticated user)

**Response (200 OK):**
```json
[
  {
    "ID": 1,
    "AccountNumber": "1234567890",
    "Amount": "1000",
    "Reference": "TXN-001",
    "Type": "inward_credit",
    "Status": "processed",
    "CreatedAt": "2026-01-15T10:00:00Z"
  }
]
```

---

### `GET /v1/vfd/webhooks/{id}`

**Description:** Fetches details of a specific webhook notification.

**Authorization:** Required (any authenticated user)

**Response (200 OK):**
```json
{
  "ID": 1,
  "AccountNumber": "1234567890",
  "Amount": "1000",
  "Reference": "TXN-001",
  "Type": "inward_credit",
  "Status": "processed",
  "CreatedAt": "2026-01-15T10:00:00Z",
  "Payload": {...}
}
```

---

### `GET /v1/vfd/webhooks/account/{accountNumber}`

**Description:** Lists webhook notifications for a specific account number.

**Authorization:** Required (any authenticated user)

**Response (200 OK):**
```json
[
  {
    "ID": 1,
    "AccountNumber": "1234567890",
    "Amount": "1000",
    "Reference": "TXN-001",
    "Type": "inward_credit",
    "Status": "processed"
  }
]
```

---

## **9. Background Jobs & Scheduler**

This is not a direct API endpoint but a crucial backend process that underpins automation.

**Description:** The scheduler manages and executes time-based tasks without direct user interaction.

**How it Works in PayFlow:**

1. When a payroll run is approved via `POST /v1/payroll-runs/{id}/approve`, the `PayrollService` schedules a one-time job with the `gocron` scheduler. The job is set to trigger on the run's `ScheduledFor` date.

2. On the scheduled date, the scheduler wakes up and executes the job.

3. The job calls the `PayrollService.ProcessApprovedPayroll` method, which:
   - Updates the run's status to `processing`
   - Converts payroll entries to bulk transfer requests
   - Executes bulk transfers via the provider manager (Korapay/VFD)
   - Verifies transfers in the database
   - Updates status to `completed` on success or `failed` on error

4. If the payment fails, the status is updated to `failed` and the admin is notified.

This decoupling ensures that the API response for approving a payroll is instant, while the actual, potentially long-running, payment process happens reliably in the background.

---

## **Error Responses**

All endpoints may return error responses in the following format:

**Response (400 Bad Request / 401 Unauthorized / 404 Not Found / 500 Internal Server Error):**
```json
{
  "error": "Error message describing what went wrong",
  "code": "ERROR_CODE"
}
```

**Common Error Codes:**
- `VALIDATION_FAILED`: Request validation failed
- `UNAUTHORIZED`: Authentication required or invalid token
- `FORBIDDEN`: Insufficient permissions
- `NOT_FOUND`: Resource not found
- `INTERNAL_SERVER_ERROR`: Server error occurred

---

## **Authentication**

All protected endpoints require a JWT token in the `Authorization` header:

```
Authorization: Bearer <your_jwt_token>
```

The JWT token is obtained from the `POST /v1/auth/login` endpoint and contains:
- `user_id`: The authenticated user's ID
- `business_id`: The user's business ID
- `role`: The user's role (admin, operator, approver)

---

## **Notes**

1. **Amounts**: All monetary amounts are stored in the smallest currency unit (kobo for NGN). For example, ₦1,000.00 is stored as `100000`.

2. **Period Format**: Payroll periods can be specified as `"YYYY-MM"` (e.g., `"2026-01"`) or `"YYYY-MM-DD"` (e.g., `"2026-01-15"`).

3. **Bank Codes**: Common Nigerian bank codes:
   - `044`: Access Bank
   - `033`: United Bank for Africa (UBA)
   - `058`: Guaranty Trust Bank (GTB)
   - `011`: First Bank
   - `057`: Zenith Bank

4. **Transfer Providers**: The system uses Korapay as the primary provider with VFD as fallback. Provider selection is automatic based on configuration.

5. **Account Numbers**: Korapay requires exactly 10 characters for account numbers. The system automatically pads or truncates as needed.

---

## **Base URL**

All endpoints are prefixed with `/v1`:

```
http://localhost:8080/v1/...
```

For production, replace `localhost:8080` with your production domain.
