# Testing Single Transfer with Korapay as Primary Provider

## Setup

1. **Set Environment Variables:**
```bash
export TRANSFER_DEFAULT_PROVIDER=korapay
export TRANSFER_PROVIDER_FALLBACK_ORDER=vfd
export KORAPAY_API_KEY=your_korapay_api_key
export KORAPAY_BASE_URL=https://api.korapay.com/merchant/api/v1
```

2. **Start the Server:**
```bash
make run
```

## Testing Approach

Since Korapay doesn't support account enquiry, we have two options:

### Option 1: Provide Account Details in Request (Recommended for Testing)

This skips the account enquiry step and goes directly to transfer with Korapay.

**API Endpoint:** `POST /v1/bulk-transfers/single`

**Request Example:**
```json
{
  "from_account_number": "1234567890",
  "to_account_number": "0987654321",
  "to_bank_code": "033",
  "amount": "1000.00",
  "remark": "Test transfer with Korapay",
  "transfer_type": "inter",
  "reference": "TEST-KORA-1234567890",
  "from_account_details": {
    "accountNo": "1234567890",
    "accountBalance": "50000.00",
    "accountId": "acc-123",
    "client": "Test Company",
    "clientId": "client-123",
    "savingsProductName": "Current Account"
  },
  "to_account_details": {
    "name": "John Doe",
    "clientId": "client-456",
    "bvn": "12345678901",
    "account": {
      "number": "0987654321",
      "id": "acc-456"
    },
    "status": "active",
    "currency": "NGN",
    "bank": "033"
  }
}
```

### Option 2: Test Fallback Behavior

Without account details, the system will:
1. Try Korapay for account enquiry → **Fails** (not supported)
2. Fallback to VFD for account enquiry → **Succeeds**
3. Try Korapay for transfer → **Should succeed**

## Expected Behavior

When Korapay is the default provider:
- **Account Enquiry:** Will fail with Korapay, fallback to VFD
- **Beneficiary Enquiry:** Will fail with Korapay, fallback to VFD  
- **Transfer Initiation:** Will use Korapay API directly

## Testing with cURL

```bash
# 1. Get JWT token (login first)
TOKEN=$(curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"your-email","password":"your-password"}' | jq -r '.token')

# 2. Test single transfer with account details provided
curl -X POST http://localhost:8080/v1/bulk-transfers/single \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "from_account_number": "1234567890",
    "to_account_number": "0987654321",
    "to_bank_code": "033",
    "amount": "1000.00",
    "remark": "Test transfer",
    "transfer_type": "inter",
    "reference": "TEST-KORA-123",
    "from_account_details": {
      "accountNo": "1234567890",
      "client": "Test Company",
      "clientId": "client-123",
      "accountId": "acc-123"
    },
    "to_account_details": {
      "name": "John Doe",
      "account": {"number": "0987654321", "id": "acc-456"},
      "bank": "033"
    }
  }'
```

## Verifying Provider Usage

Check the server logs to see which provider is being used:
- Look for: `"Attempting transfer" "provider"="korapay"`
- Look for: `"Transfer succeeded" "provider"="korapay"`

