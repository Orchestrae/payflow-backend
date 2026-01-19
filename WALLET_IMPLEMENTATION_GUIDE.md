# Wallet Implementation Guide

## Overview

This document provides a comprehensive guide to the wallet and virtual account implementation, including all endpoints, request/response formats, and testing instructions.

## Table of Contents

1. [Virtual Account Management](#virtual-account-management)
2. [Wallet Operations](#wallet-operations)
3. [Account Holder / KYC Management](#account-holder--kyc-management)
4. [File Upload for KYC Documents](#file-upload-for-kyc-documents)
5. [Sandbox Testing](#sandbox-testing)
6. [Webhook Integration](#webhook-integration)
7. [Testing Instructions](#testing-instructions)

---

## Virtual Account Management

### 1. Create Virtual Account

Creates a virtual bank account for a business to receive payments.

**Endpoint:** `POST /v1/wallets/virtual-account`  
**Authentication:** Required (JWT)  
**Role:** Any authenticated user

**Request Body:**
```json
{
  "account_name": "Steph James",
  "account_reference": "your-account-reference-001",  // Optional - auto-generated if not provided
  "customer_name": "Don Alpha",
  "customer_email": "don_alpha@email.com",
  "bvn": "11111111111",  // Required, exactly 11 digits
  "nin": "12345678901",  // Optional
  "bank_code": "000",     // Optional - provider may assign
  "permanent": true       // Optional - defaults to true
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "Virtual bank account created successfully",
  "data": {
    "account_name": "Steph James",
    "account_number": "1110031765",
    "bank_code": "000",
    "bank_name": "Test Bank",
    "customer": {
      "name": "Don Alpha",
      "email": "don_alpha@email.com"
    },
    "account_reference": "your-account-reference-001",
    "unique_id": "KPY-VA-yrJtnFgVesLeKgM",
    "account_status": "active",
    "created_at": "2026-01-18T02:27:20.390Z",
    "currency": "NGN"
  }
}
```

**cURL Example:**
```bash
curl -X POST http://localhost:8080/v1/wallets/virtual-account \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "account_name": "Steph James",
    "account_reference": "acc-ref-001",
    "customer_name": "Don Alpha",
    "customer_email": "don_alpha@email.com",
    "bvn": "11111111111",
    "permanent": true
  }'
```

---

## Wallet Operations

### 2. Get Wallet Details

Retrieves wallet details including balance and virtual account information.

**Endpoint:** `GET /v1/wallets`  
**Authentication:** Required (JWT)  
**Role:** Any authenticated user

**Response (200 OK):**
```json
{
  "id": 1,
  "business_id": 1,
  "balance": 100000,  // Balance in kobo (NGN 1,000.00)
  "locked_balance": 5000,  // Locked balance in kobo (NGN 50.00)
  "currency": "NGN",
  "virtual_account_number": "1110031765",
  "virtual_account_bank_code": "000",
  "virtual_account_bank_name": "Test Bank",
  "virtual_account_reference": "your-account-reference-001",
  "virtual_account_unique_id": "KPY-VA-yrJtnFgVesLeKgM",
  "virtual_account_status": "active",
  "provider": "korapay",
  "balance_updated_at": "2026-01-18T02:27:20.390Z",
  "created_at": "2026-01-18T02:27:20.390Z",
  "updated_at": "2026-01-18T02:27:20.390Z"
}
```

**cURL Example:**
```bash
curl -X GET http://localhost:8080/v1/wallets \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 3. Get Wallet Balance

Retrieves current wallet balance.

**Endpoint:** `GET /v1/wallets/balance`  
**Authentication:** Required (JWT)  
**Role:** Any authenticated user

**Response (200 OK):**
```json
{
  "balance": 100000,  // Balance in kobo (NGN 1,000.00)
  "currency": "NGN"
}
```

**cURL Example:**
```bash
curl -X GET http://localhost:8080/v1/wallets/balance \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 4. Get Wallet Transactions

Retrieves transaction history for the wallet with pagination.

**Endpoint:** `GET /v1/wallets/transactions?page=1&limit=10`  
**Authentication:** Required (JWT)  
**Role:** Any authenticated user

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 10)

**Response (200 OK):**
```json
{
  "transactions": [
    {
      "id": 1,
      "business_id": 1,
      "transaction_type": "deposit",
      "amount": 100000,  // Amount in kobo (NGN 1,000.00)
      "balance_before": 0,
      "balance_after": 100000,
      "currency": "NGN",
      "reference": "DEP-001",
      "provider_reference": "KPY-PAY-4l5O8mxmgX2kijp",
      "description": "Payment by Don Alpha",
      "status": "completed",
      "processed_at": "2026-01-18T02:27:20.390Z",
      "created_at": "2026-01-18T02:27:20.390Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 10
}
```

**Transaction Types:**
- `deposit`: Money deposited into wallet
- `withdrawal`: Money withdrawn from wallet (transfers)
- `fee`: Processing fees deducted
- `lock`: Balance locked for pending transaction
- `unlock`: Balance unlocked after transaction

**cURL Example:**
```bash
curl -X GET "http://localhost:8080/v1/wallets/transactions?page=1&limit=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

---

## Account Holder / KYC Management

### 5. Create Account Holder

Creates an account holder for KYC onboarding (required before creating virtual accounts in production).

**Endpoint:** `POST /v1/wallets/account-holders`  
**Authentication:** Required (JWT)  
**Role:** Any authenticated user

**Request Body:**
```json
{
  "first_name": "Sarah",
  "last_name": "Doe",
  "use_case": "Personal",
  "type": "individual",
  "date_of_birth": "1988-04-04",
  "nationality": "NG",
  "occupation": "Entrepreneur",
  "email": "sarah.doe@example.com",
  "phone": "+2348133443000",
  "bank_id_number": "12332435200",  // BVN
  "source_of_inflow": "bank_statement",
  "source_of_inflow_document": {
    "reference": "KPY-FILE-202509091525ocHUp9oxwrt4lICRJ63S35963"
  },
  "selfie": {
    "reference": "KPY-FILE-202509091525ocHUp9oxwrt4lICRJ63S35963"
  },
  "identification": {
    "type": "passport",
    "number": "A11111111",
    "document_front": {
      "reference": "KPY-FILE-202509091525ocHUp9oxwrt4lICRJ63S35963"
    },
    "issued_date": "2010-01-01",
    "expiry_date": "2030-01-01",
    "country": "NG"
  },
  "proof_of_address": {
    "type": "bank_statement",
    "document": {
      "reference": "KPY-FILE-202509091525ocHUp9oxwrt4lICRJ63S35963"
    }
  },
  "address": {
    "country": "NG",
    "zip": "12345",
    "address": "Freedom Way St",
    "state": "Lagos",
    "city": "Lagos"
  },
  "employment": {
    "status": "employer",
    "employer": "Google",
    "description": "I am an employer"
  },
  "metadata": {
    "key1": "value1"
  }
}
```

**Response (201 Created):**
```json
{
  "reference": "KPY-AH-CGAXuc6jZwDA8TJ",
  "email": "sarah.doe@example.com",
  "status": "pending",
  "metadata": {
    "key1": "value1"
  }
}
```

**cURL Example:**
```bash
curl -X POST http://localhost:8080/v1/wallets/account-holders \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Sarah",
    "last_name": "Doe",
    "use_case": "Personal",
    "type": "individual",
    "date_of_birth": "1988-04-04",
    "nationality": "NG",
    "email": "sarah.doe@example.com",
    "phone": "+2348133443000",
    "bank_id_number": "12332435200",
    "source_of_inflow": "bank_statement"
  }'
```

### 6. Get Account Holder Details

Retrieves account holder details by reference.

**Endpoint:** `GET /v1/wallets/account-holders/{reference}/details`  
**Authentication:** Required (JWT)  
**Role:** Any authenticated user

**Response (200 OK):**
```json
{
  "reference": "KPY-AH-WveCyyYAvKh107Y",
  "account_type": "individual",
  "first_name": "Sarah",
  "last_name": "Doe",
  "email": "israel.ban12@example.com",
  "phone_number": "+2348121775311",
  "occupation": "Software Engineering",
  "status": "pending",
  "metadata": {
    "key1": "value1"
  },
  "date_created": "2024-03-22T13:27:17.000Z",
  "country": "NG",
  "date_of_birth": "1988-04-04T00:00:00.000Z",
  "address": {
    "zip": "12345",
    "city": "Lagos",
    "state": "Lagos",
    "address": "Freedom Way St",
    "country": "NG"
  },
  "documents": {
    "identification_front": "...",  // Base64 encoded
    "identification_back": "...",
    "proof_of_address": "...",
    "selfie": "...",
    "source_of_inflow": "..."
  }
}
```

**cURL Example:**
```bash
curl -X GET http://localhost:8080/v1/wallets/account-holders/KPY-AH-WveCyyYAvKh107Y/details \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 7. Update Account Holder KYC

Updates account holder KYC information.

**Endpoint:** `PATCH /v1/wallets/account-holders/{reference}/update-kyc`  
**Authentication:** Required (JWT)  
**Role:** Any authenticated user

**Request Body:**
```json
{
  "first_name": "Sarah",
  "last_name": "Doe",
  "source_of_inflow": "bank_statement",
  "source_of_inflow_document": {
    "reference": "KPY-FILE-202509091525ocHUp9oxwrt4lICRJ63S35963"
  },
  "selfie": {
    "reference": "KPY-FILE-202509091525ocHUp9oxwrt4lICRJ63S35963"
  },
  "identification": {
    "type": "passport",
    "number": "A11111111",
    "document_front": {
      "reference": "KPY-FILE-202509091525ocHUp9oxwrt4lICRJ63S35963"
    },
    "issued_date": "2010-01-01",
    "expiry_date": "2030-01-01",
    "country": "NG"
  },
  "proof_of_address": {
    "type": "bank_statement",
    "document": {
      "reference": "KPY-FILE-202509091525ocHUp9oxwrt4lICRJ63S35963"
    }
  }
}
```

**Response (200 OK):**
```json
{
  "reference": "KPY-AH-WveCyyYAvKh107Y",
  "first_name": "Sarah",
  "last_name": "Doe",
  "status": "pending"
}
```

**cURL Example:**
```bash
curl -X PATCH http://localhost:8080/v1/wallets/account-holders/KPY-AH-WveCyyYAvKh107Y/update-kyc \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Sarah",
    "last_name": "Doe",
    "source_of_inflow": "bank_statement"
  }'
```

---

## File Upload for KYC Documents

### 8. Generate File Upload URL

Generates a pre-signed S3 URL for uploading KYC documents.

**Endpoint:** `POST /v1/wallets/files/generate-upload-url`  
**Authentication:** Required (JWT)  
**Role:** Any authenticated user

**Request Body:**
```json
{
  "reference": "REF-J999222D338SFSwwww",  // Your unique reference
  "purpose": "kyc_document",  // kyc_document, proof_of_address, etc.
  "content_type": "image/jpeg"  // image/jpeg, application/pdf, image/png
}
```

**Response (200 OK):**
```json
{
  "korapay_reference": "KPY-FILE-202508291414WUPh6CmvLAU052iYVynj14842",
  "owner_reference": "REF-JGSJHVVV122200",
  "purpose": "kyc_document",
  "upload_url": "https://kpy-exported-files-staging.s3.amazonaws.com/...",
  "upload_url_expires": "2025-08-29T14:14:14.849Z"
}
```

**Workflow:**
1. Call this endpoint to get an upload URL
2. Upload the file to the returned S3 URL using `PUT` or `POST`
3. Use the `korapay_reference` in account holder creation/update requests

**cURL Example:**
```bash
# Step 1: Generate upload URL
curl -X POST http://localhost:8080/v1/wallets/files/generate-upload-url \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reference": "REF-MY-DOC-001",
    "purpose": "kyc_document",
    "content_type": "image/jpeg"
  }'

# Step 2: Upload file to the returned upload_url (from Step 1)
curl -X PUT "UPLOAD_URL_FROM_STEP_1" \
  -H "Content-Type: image/jpeg" \
  --data-binary @/path/to/document.jpg
```

---

## Sandbox Testing

### 9. Sandbox Credit (Testing Only)

Credits a virtual account in sandbox environment for testing purposes. **This endpoint only works in sandbox/test mode.**

**Endpoint:** `POST /v1/wallets/sandbox/credit`  
**Authentication:** Required (JWT)  
**Role:** Admin only

**Request Body:**
```json
{
  "account_number": "1110000395",
  "amount": 100,  // Amount in main currency unit (NGN 100.00)
  "currency": "NGN"  // Optional, defaults to NGN
}
```

**Response (200 OK):**
```json
{
  "status": true,
  "message": "Virtual bank account credited successfully"
}
```

**Important Notes:**
- Only available in sandbox/test environment
- Admin role required
- Automatically records deposit in wallet balance
- Use for testing deposit flows without real money

**cURL Example:**
```bash
curl -X POST http://localhost:8080/v1/wallets/sandbox/credit \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "account_number": "1110000395",
    "amount": 100,
    "currency": "NGN"
  }'
```

---

## Webhook Integration

### 10. KoraPay Deposit Webhook

Receives deposit notifications from KoraPay when money is deposited into a virtual account.

**Endpoint:** `POST /korapay/webhooks/deposit`  
**Authentication:** None (public endpoint, but should verify signature in production)

**Webhook Payload (KoraPay format):**
```json
{
  "reference": "KPY-PAY-4l5O8mxmgX2kijp",
  "account_reference": "your-account-reference-001",
  "account_number": "1110031765",
  "amount": "1000.00",
  "currency": "NGN",
  "status": "success",
  "description": "Payment by Don Alpha",
  "created_at": "2026-01-18T02:27:20.390Z",
  "payer_bank_account": {
    "account_number": "1007408761",
    "account_name": "Steph James",
    "bank_name": "ZENITH BANK"
  }
}
```

**Response (200 OK):**
```json
{
  "status": "success"
}
```

**Handler Behavior:**
- Parses webhook payload
- Looks up business by `account_reference`
- Records deposit in wallet
- Updates wallet balance
- Creates wallet transaction record

**Note:** Webhook format may vary - adjust `parseDepositNotification` in `wallet_handler.go` based on actual KoraPay webhook structure.

---

## Testing Instructions

### Prerequisites

1. **Environment Setup:**
   ```bash
   # Set KoraPay credentials in .env
   KORAPAY_API_KEY=sk_test_...
   KORAPAY_BASE_URL=https://api.korapay.com  # or sandbox URL
   ```

2. **Start Server:**
   ```bash
   make run
   # or
   go run cmd/server/main.go
   ```

3. **Get JWT Token:**
   ```bash
   # Register/Login to get JWT token
   curl -X POST http://localhost:8080/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{
       "email": "admin@example.com",
       "password": "password"
     }'
   ```

### Test Flow: Complete Wallet Setup

#### Step 1: Generate File Upload URLs (for KYC documents)

```bash
# Generate URL for passport
curl -X POST http://localhost:8080/v1/wallets/files/generate-upload-url \
  -H "Authorization: Bearer YOUR_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "reference": "PASSPORT-001",
    "purpose": "kyc_document",
    "content_type": "image/jpeg"
  }'

# Upload file to the returned upload_url (save korapay_reference)
# Use PUT request with the file as binary data
```

#### Step 2: Create Account Holder (KYC)

```bash
curl -X POST http://localhost:8080/v1/wallets/account-holders \
  -H "Authorization: Bearer YOUR_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "John",
    "last_name": "Doe",
    "use_case": "Personal",
    "type": "individual",
    "date_of_birth": "1990-01-01",
    "nationality": "NG",
    "email": "john.doe@example.com",
    "phone": "+2348133443000",
    "bank_id_number": "11111111111",
    "source_of_inflow": "bank_statement",
    "identification": {
      "type": "passport",
      "number": "A12345678",
      "document_front": {
        "reference": "KPY-FILE-xxx"  // From Step 1
      }
    }
  }'
```

#### Step 3: Create Virtual Account

```bash
curl -X POST http://localhost:8080/v1/wallets/virtual-account \
  -H "Authorization: Bearer YOUR_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "account_name": "John Doe Business",
    "account_reference": "my-account-ref-001",
    "customer_name": "John Doe",
    "customer_email": "john.doe@example.com",
    "bvn": "11111111111",
    "permanent": true
  }'
```

#### Step 4: Get Wallet Details

```bash
curl -X GET http://localhost:8080/v1/wallets \
  -H "Authorization: Bearer YOUR_JWT"
```

#### Step 5: Credit Wallet (Sandbox Only)

```bash
curl -X POST http://localhost:8080/v1/wallets/sandbox/credit \
  -H "Authorization: Bearer YOUR_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "account_number": "1110000395",  // From Step 3
    "amount": 5000,
    "currency": "NGN"
  }'
```

#### Step 6: Check Balance

```bash
curl -X GET http://localhost:8080/v1/wallets/balance \
  -H "Authorization: Bearer YOUR_JWT"
```

#### Step 7: View Transactions

```bash
curl -X GET "http://localhost:8080/v1/wallets/transactions?page=1&limit=10" \
  -H "Authorization: Bearer YOUR_JWT"
```

### Testing with Real Transfers

After crediting the wallet, you can test transfers using the transfer endpoints:

```bash
# Make a transfer (balance will be checked automatically)
curl -X POST http://localhost:8080/v1/transfers \
  -H "Authorization: Bearer YOUR_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "recipient_name": "Jane Doe",
    "recipient_account": "1234567890",
    "recipient_bank_code": "058",
    "amount": "1000.00",
    "currency": "NGN",
    "narration": "Test payment"
  }'
```

---

## Error Handling

All endpoints return standardized error responses:

**Error Response Format:**
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {}  // Optional additional details
  }
}
```

**Common Error Codes:**
- `VALIDATION_FAILED`: Request validation failed
- `NOT_FOUND`: Resource not found
- `FORBIDDEN`: Insufficient permissions
- `PAYMENT_GATEWAY_FAILED`: KoraPay API error
- `INSUFFICIENT_BALANCE`: Not enough balance for transfer
- `WALLET_NOT_FOUND`: Wallet does not exist

---

## Integration with Transfer Service

The wallet system is integrated with the transfer service:

1. **Balance Checking:** Before transfers, balance is automatically checked
2. **Balance Locking:** Amount is locked during transfer processing
3. **Withdrawal Recording:** Successful transfers are recorded as withdrawals
4. **Balance Unlocking:** On transfer failure, locked balance is unlocked

This ensures:
- No overdrafts
- Accurate balance tracking
- Complete audit trail

---

## Notes

1. **Amounts:** All amounts are stored in the smallest currency unit (kobo for NGN). API accepts amounts in main currency unit (e.g., `100` = NGN 100.00) or kobo (specified in request).

2. **Balance Locking:** When a transfer is initiated, the amount is locked. If the transfer succeeds, it's deducted. If it fails, it's unlocked.

3. **Transaction Types:**
   - `deposit`: Money received
   - `withdrawal`: Money sent (transfers)
   - `fee`: Processing fees
   - `lock`: Temporary lock for pending transactions
   - `unlock`: Release of locked balance

4. **Sandbox Mode:** Sandbox credit only works when `KORAPAY_BASE_URL` contains "sandbox", "test", or "staging".

5. **Production:** For production, ensure:
   - Webhook signature verification (currently placeholder)
   - Proper error handling and logging
   - Rate limiting on webhook endpoints
   - Monitoring and alerting

---

## API Summary

| Endpoint | Method | Auth | Role | Description |
|----------|--------|------|------|-------------|
| `/v1/wallets/virtual-account` | POST | ✅ | Any | Create virtual account |
| `/v1/wallets` | GET | ✅ | Any | Get wallet details |
| `/v1/wallets/balance` | GET | ✅ | Any | Get balance |
| `/v1/wallets/transactions` | GET | ✅ | Any | Get transactions |
| `/v1/wallets/account-holders` | POST | ✅ | Any | Create account holder |
| `/v1/wallets/account-holders/{ref}/details` | GET | ✅ | Any | Get account holder |
| `/v1/wallets/account-holders/{ref}/update-kyc` | PATCH | ✅ | Any | Update KYC |
| `/v1/wallets/files/generate-upload-url` | POST | ✅ | Any | Generate upload URL |
| `/v1/wallets/sandbox/credit` | POST | ✅ | Admin | Sandbox credit |
| `/korapay/webhooks/deposit` | POST | ❌ | Public | Deposit webhook |

---

## Support

For issues or questions:
1. Check error responses for detailed messages
2. Review server logs for debugging
3. Verify KoraPay API credentials
4. Ensure sandbox mode is correctly configured
