# Architecture Review: Wallet & KYC Implementation

## Issues Identified

### ❌ **Violation 1: KYC Handlers Directly Call Platform Client**

**Location:** `internal/api/handler/wallet_handler.go` (lines 393-654)

**Problem:**
- Handlers directly call `h.koraClient` (platform-specific client)
- Handlers convert request DTOs to KoraPay types directly
- No service layer abstraction
- Violates separation of concerns (HTTP layer knows about platform implementation)

**Current Pattern (WRONG):**
```
Handler → KoraPay Client (direct)
```

**Expected Pattern (CORRECT):**
```
Handler → Service → Provider Interface → Platform Implementation
```

### ❌ **Violation 2: Missing Domain Models**

**Problem:**
- KYC operations use KoraPay types (`korapay.AccountHolderCreateRequest`) in handlers
- No provider-agnostic domain models for account holder operations
- Violates DDD principle (domain should be provider-agnostic)

### ❌ **Violation 3: Missing Service Layer**

**Problem:**
- No `AccountHolderService` interface
- Business logic mixed into handlers
- No abstraction for future provider support (VFD, etc.)

### ✅ **What's Correct**

1. **Wallet Service** - Properly uses `VirtualAccountProvider` interface
2. **Transfer Service** - Properly uses `TransferProviderManager` and service abstraction
3. **Domain Models** - Wallet domain models are provider-agnostic
4. **Provider Pattern** - Virtual account provider pattern is correctly implemented

## Fixes Required

### ✅ **Fix 1: Domain Models Added**
- Added `CreateAccountHolderRequest`, `AccountHolderResult`, `AccountHolderDetails`, etc. to `domain/wallet.go`
- All models are provider-agnostic

### ✅ **Fix 2: Provider Interface Created**
- Created `AccountHolderProvider` interface in `service/provider/account_holder_provider.go`
- Follows same pattern as `VirtualAccountProvider`

### 🔄 **Fix 3: KoraPay Provider Implementation (IN PROGRESS)**
- Need to implement `AccountHolderProvider` for KoraPay
- Map domain models to KoraPay API types
- Similar to `korapayVirtualAccountProvider`

### 🔄 **Fix 4: AccountHolderService (IN PROGRESS)**
- Create `AccountHolderService` interface
- Implement service that uses `AccountHolderProvider`
- Move business logic from handlers to service

### 🔄 **Fix 5: Refactor Handlers (IN PROGRESS)**
- Remove direct `koraClient` calls
- Remove KoraPay type conversions
- Use `AccountHolderService` instead
- Handlers should only: validate, call service, format response

## Architecture Comparison

### Current (WRONG) - KYC Handlers:
```
HTTP Request
    ↓
Handler (wallet_handler.go)
    ↓ [converts to KoraPay types]
    ↓ [calls koraClient directly]
KoraPay Client
    ↓
KoraPay API
```

### Target (CORRECT) - Following Wallet Pattern:
```
HTTP Request
    ↓
Handler (wallet_handler.go)
    ↓ [validates, converts to domain]
AccountHolderService
    ↓ [business logic]
AccountHolderProvider (interface)
    ↓
KoraPay Provider Implementation
    ↓
KoraPay Client
    ↓
KoraPay API
```

## Implementation Status

- [x] Domain models created
- [x] Provider interface created
- [x] KoraPay provider implementation
- [x] AccountHolderService interface & implementation
- [x] Handler refactoring
- [x] Update main.go wiring

## ✅ **REFACTORING COMPLETE**

All architectural violations have been fixed. The KYC handlers now follow the same DDD pattern as the wallet service:

1. **Domain Models** - Provider-agnostic models in `domain/wallet.go`
2. **Provider Interface** - `AccountHolderProvider` in `service/provider/account_holder_provider.go`
3. **KoraPay Implementation** - `korapayAccountHolderProvider` in `platform/korapay/account_holder_provider.go`
4. **Service Layer** - `AccountHolderService` in `service/account_holder_service.go`
5. **Handler Refactoring** - Handlers now use service instead of direct client calls
6. **Dependency Injection** - All services wired in `main.go`

## Next Steps

1. Implement `AccountHolderProvider` for KoraPay (similar to `korapayVirtualAccountProvider`)
2. Create `AccountHolderService` interface and implementation
3. Refactor handlers to use service
4. Update `main.go` to wire everything together
5. Test end-to-end
