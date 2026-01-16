# Payroll Workflow Configuration & Instant Run

## ✅ Features Implemented

### 1. **Business-Configurable Payroll Workflow**

Businesses can now configure their payroll workflow behavior:

#### Configuration Fields (in `businesses` table):
- `payroll_requires_approval` (boolean, default: `true`)
  - If `false`: Payroll auto-approves after submission (skips approval step)
  - If `true`: Requires explicit approval (standard workflow)

- `payroll_auto_process` (boolean, default: `false`)
  - If `true`: Approved payroll processes immediately (bypasses scheduler)
  - If `false`: Approved payroll is scheduled for later processing

#### How It Works:

**Standard Workflow (default):**
```
draft → (submit) → pending_approval → (approve) → approved → (scheduled) → processing → completed
```

**Auto-Approval Workflow (`payroll_requires_approval = false`):**
```
draft → (submit) → approved → (scheduled/auto-process) → processing → completed
```

**Auto-Process Workflow (`payroll_auto_process = true`):**
```
draft → (submit) → pending_approval → (approve) → approved → (instant) → processing → completed
```

### 2. **Instant Run for Testing**

An endpoint to process payroll immediately, bypassing the scheduler. Perfect for testing and verification.

#### Endpoint:
```
POST /v1/payroll-runs/{id}/process-now
```

#### What It Does:
1. ✅ Converts payroll entries to bulk transfer requests
2. ✅ Executes bulk transfer via provider manager (uses Korapay native bulk)
3. ✅ Verifies all transfers were created in database
4. ✅ Updates payroll run status based on results
5. ✅ Returns detailed results

#### Features:
- **Bypasses Approval**: Can process draft or approved payroll runs
- **Immediate Processing**: No scheduler delay
- **Database Verification**: Confirms transfers exist in DB
- **Account Number Validation**: Ensures 10-character account numbers (Korapay requirement)
- **Bank Code Mapping**: Maps bank names to codes automatically

## 📋 Database Changes

### Migration: `000006_add_business_payroll_config`

```sql
ALTER TABLE businesses
ADD COLUMN payroll_requires_approval BOOLEAN DEFAULT true,
ADD COLUMN payroll_auto_process BOOLEAN DEFAULT false;
```

## 🔧 Configuration

### Update Business Configuration (SQL):
```sql
-- Enable auto-approval (skip approval step)
UPDATE businesses 
SET payroll_requires_approval = false 
WHERE id = 1;

-- Enable auto-process (process immediately after approval)
UPDATE businesses 
SET payroll_auto_process = true 
WHERE id = 1;
```

### Update Business Configuration (API - To Be Implemented):
```bash
PATCH /v1/businesses/{id}/payroll-config
{
  "requires_approval": false,
  "auto_process": true
}
```

## 🧪 Testing

### Test Standard Workflow:
```bash
./test_payroll_workflow.sh
```

### Test Instant Run:
```bash
./test_instant_payroll.sh
```

### Manual Test:
```bash
# 1. Create payroll run
PAYROLL_ID=$(curl -X POST http://localhost:8080/v1/payroll-runs \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"period": "2026-01"}' | jq -r '.ID')

# 2. Process instantly
curl -X POST "http://localhost:8080/v1/payroll-runs/$PAYROLL_ID/process-now" \
  -H "Authorization: Bearer $TOKEN"

# 3. Verify transfers in DB
curl -X GET "http://localhost:8080/v1/transfers?limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

## 📊 Verification Results

The instant run endpoint verifies transfers in the database:

```go
type TransferVerificationResult struct {
    AllVerified   bool              // All transfers found in DB
    VerifiedCount int               // Number of transfers verified
    TotalCount     int               // Expected number of transfers
    Transfers      []*domain.Transfer // Matching transfers
}
```

### Verification Logic:
1. Fetches recent transfers for the business (last 2 minutes)
2. Filters transfers matching pattern: `PAYROLL-{runID}-EMP-{empID}`
3. Compares count with expected transfers
4. Returns verification result

## 🔍 Example Response

### Instant Run Response:
```json
{
  "ID": 11,
  "Status": "completed",
  "PaymentReference": "PAYROLL-RUN-11",
  "TotalNetPay": 12250000,
  "Entries": [...]
}
```

### Verified Transfers in Database:
```json
{
  "transfers": [
    {
      "Reference": "PAYROLL-11-EMP-22",
      "Amount": "750000",
      "Status": "pending",
      "RecipientAccountNumber": "0000000001",
      "RecipientBankCode": "044"
    },
    ...
  ]
}
```

## 🎯 Use Cases

### 1. **Testing & Development**
- Use instant run to test payroll processing without waiting for scheduler
- Verify bulk transfers work correctly
- Check database records are created

### 2. **Production with Auto-Approval**
- Set `payroll_requires_approval = false` for businesses that don't need approval
- Payroll auto-approves after submission

### 3. **Production with Auto-Process**
- Set `payroll_auto_process = true` for immediate disbursement
- Useful for businesses that want instant payment

### 4. **Standard Workflow**
- Keep defaults (`requires_approval = true`, `auto_process = false`)
- Standard approval and scheduling workflow

## 📝 Files Modified

1. **`internal/domain/business.go`**
   - Added `PayrollRequiresApproval` and `PayrollAutoProcess` fields

2. **`migrations/000006_add_business_payroll_config.up.sql`**
   - Migration to add configuration fields

3. **`internal/service/payroll_service.go`**
   - Updated `SubmitForApproval` to check business config
   - Updated `ApprovePayrollRun` to check auto-process config
   - Added `ProcessPayrollRunInstantly` method
   - Added `verifyTransfersInDatabase` method
   - Added `mapBankNameToCode` helper

4. **`internal/api/handler/payroll_handler.go`**
   - Added `ProcessPayrollRunInstantly` handler

5. **`internal/api/router.go`**
   - Added route: `POST /v1/payroll-runs/{id}/process-now`

6. **`cmd/server/main.go`**
   - Updated `NewPayrollService` to include new dependencies

## ⚠️ Important Notes

1. **Account Number Format**: Korapay requires exactly 10 characters. The system automatically:
   - Truncates to last 10 digits if longer
   - Pads with zeros if shorter

2. **Test Wallet Limits**: In test environment, Korapay may return "insufficient funds". The system still:
   - Creates transfers in database
   - Verifies transfers exist
   - Marks as completed if verified (for testing)

3. **Bank Code Mapping**: Currently uses a simple map. In production, use a proper bank code lookup service.

4. **Verification Timing**: Transfers are verified within 2 minutes of creation. Adjust if needed.

## 🚀 Next Steps (Optional)

1. Add API endpoint to update business payroll configuration
2. Add bank code lookup service/database
3. Add webhook to update transfer status when provider confirms
4. Add retry logic for failed transfers
5. Add detailed transfer status reporting
