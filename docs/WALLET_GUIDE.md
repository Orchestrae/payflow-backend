# Wallet Guide

Comprehensive documentation for PayFlow's wallet system, covering virtual account management, KYC, webhook integration, and atomic balance operations.

---

## Table of Contents

- [Virtual Account Management](#virtual-account-management)
- [Account Holder and KYC](#account-holder-and-kyc)
- [File Upload](#file-upload)
- [Sandbox Testing](#sandbox-testing)
- [Webhook Integration](#webhook-integration)
  - [Korapay Deposit Webhooks](#korapay-deposit-webhooks)
  - [VFD Credit Webhooks](#vfd-credit-webhooks)
- [Wallet Balance Operations](#wallet-balance-operations)
- [Currency](#currency)
- [Cross-References](#cross-references)

---

## Virtual Account Management

### Create Virtual Bank Account

`POST /v1/wallets/virtual-account`

Creates a virtual bank account via the Korapay provider.

**Request Body:**

| Field           | Type   | Required | Description                          |
|-----------------|--------|----------|--------------------------------------|
| account_name    | string | Yes      | Display name for the account         |
| customer_name   | string | Yes      | Full name of the account owner       |
| customer_email  | string | Yes      | Email address of the account owner   |
| bvn             | string | Yes      | Bank Verification Number (11 digits) |
| nin             | string | No       | National Identification Number       |

**Response Fields:**

| Field              | Type   | Description                              |
|--------------------|--------|------------------------------------------|
| account_number     | string | The generated virtual account number     |
| bank_code          | string | Bank code for the virtual account        |
| bank_name          | string | Human-readable bank name                 |
| account_reference  | string | Unique reference for the virtual account |

### Get Wallet Details

`GET /v1/wallets/`

Returns the full wallet record including balance, locked balance, and associated virtual account information.

### Get Current Balance

`GET /v1/wallets/balance`

Returns the current available balance for the authenticated business wallet.

### Get Transaction History

`GET /v1/wallets/transactions`

Returns a paginated list of wallet transactions (deposits, withdrawals, locks, unlocks).

---

## Account Holder and KYC

These endpoints manage Korapay account holders and their KYC verification status.

### Create Account Holder

`POST /v1/wallets/account-holders`

Creates a new account holder record with the payment provider.

### Get Account Holder Details

`GET /v1/wallets/account-holders/{reference}/details`

Retrieves the current details and KYC status for an account holder identified by their reference.

### Update KYC Information

`PATCH /v1/wallets/account-holders/{reference}/update-kyc`

Submits updated KYC information for an existing account holder. The `{reference}` path parameter is the account holder's unique reference.

---

## File Upload

### Generate Signed Upload URL

`POST /v1/wallets/files/generate-upload-url`

Generates a pre-signed upload URL for uploading KYC documents (identity cards, utility bills, etc.) directly to the provider's storage. The client uses the returned URL to upload the file via a subsequent PUT request.

---

## Sandbox Testing

### Simulate Deposit

`POST /v1/wallets/sandbox/credit`

**Admin only.** Simulates a deposit into the business wallet. Available only in sandbox/test environments. Useful for testing payroll processing without requiring real bank transfers.

---

## Webhook Integration

PayFlow receives asynchronous notifications from payment providers via webhooks. All webhook endpoints perform cryptographic signature verification before processing.

### Korapay Deposit Webhooks

**Endpoint:** `POST /korapay/webhooks/deposit`

**Signature Verification:**

- Header: `x-korapay-signature`
- Algorithm: HMAC-SHA256
- Signed payload: the `data` field from the webhook body
- Secret: Korapay secret key
- Verification is mandatory. Requests with invalid or missing signatures are rejected.

**Event Handling:**

- Only `charge.success` events are processed.
- The handler extracts the `account_reference` from `virtual_bank_account_details` in the webhook payload.
- A matching wallet is looked up by account reference, and a deposit is recorded atomically.

### VFD Credit Webhooks

VFD Bank sends credit notifications through multiple webhook endpoints.

| Endpoint                               | Description                              |
|----------------------------------------|------------------------------------------|
| `POST /vfd/webhooks/inward-credit`     | Settled inward credit transactions       |
| `POST /vfd/webhooks/initial-inward-credit` | Pre-settlement credit notifications  |
| `POST /vfd/webhooks/retrigger`         | Re-trigger a previously received webhook |

**Signature Verification:**

- Header: `X-VFD-Signature`
- Algorithm: HMAC-SHA256
- Secret: Configured via `VFD_WEBHOOK_SECRET` environment variable
- Verification is applied when `VFD_WEBHOOK_SECRET` is configured. If the secret is not set, signature verification is skipped.

---

## Wallet Balance Operations

All balance mutations are performed atomically at the database level to prevent race conditions and ensure consistency.

### LockBalance

Atomically locks a specified amount from the wallet's available balance. The lock is enforced in the SQL `WHERE` clause itself, guaranteeing that the available balance (balance minus locked_balance) is sufficient before the update proceeds.

```
UPDATE wallets
SET locked_balance = locked_balance + amount
WHERE id = ? AND (balance - locked_balance) >= amount
```

If the available balance is insufficient, the update affects zero rows and the operation returns an error.

### UnlockBalance

Atomically decrements the locked balance. Uses `GREATEST` to prevent the locked balance from going negative in edge cases.

```
UPDATE wallets
SET locked_balance = GREATEST(locked_balance - amount, 0)
WHERE id = ?
```

### RecordDeposit

Atomically increments the wallet balance. Enforces idempotency using the transaction reference -- duplicate deposits with the same reference are silently ignored.

### RecordWithdrawal

Atomically decrements both the total balance and the locked balance. A database-level `CHECK` constraint prevents the balance from going negative.

```
UPDATE wallets
SET balance = balance - amount, locked_balance = locked_balance - amount
WHERE id = ?
```

---

## Currency

All monetary values are stored as `int64` in the smallest currency unit (kobo for NGN).

| Kobo Value | Naira Equivalent |
|------------|------------------|
| 500000     | NGN 5,000        |
| 100        | NGN 1            |
| 50         | NGN 0.50         |

---

## Cross-References

- [Architecture](./ARCHITECTURE.md)
- [API Reference](./API_REFERENCE.md)
- [Payroll Guide](./PAYROLL_GUIDE.md)
