# Migration Reconciliation Plan

## Problem Statement

We have a hybrid migration setup:
- **Auto-migration**: Creates tables on startup (if enabled)
- **Traditional migrations**: SQL files that must be run manually

This creates potential inconsistencies between:
- **Development** (auto-migration enabled) - tables created automatically
- **Production** (auto-migration disabled) - tables created via migrations only

## Identified Gaps

### Missing Migration Scripts

1. ❌ **`vfd_webhook_notifications`** - No migration script
   - **Impact**: HIGH - Actively used by webhook service
   - **Status**: Migration created (000009)
   
2. ❌ **`transfer_records`** - No migration script
   - **Impact**: LOW - Repository marked as deprecated
   - **Status**: Migration created (000009) for backward compatibility

### Complete Coverage Now ✅

All tables now have migration scripts:
- ✅ Core tables (000001) - businesses, users, cadres, employees, payroll tables
- ✅ Schema changes (000002-000007) - deleted_at, timestamps, config, descriptions
- ✅ Transfers (000005) - transfers table
- ✅ Wallets (000008) - business_wallets, wallet_transactions
- ✅ VFD tables (000009) - vfd_webhook_notifications, transfer_records

## Reconciliation Strategy

### For Existing Databases (Development/Production)

If tables already exist from auto-migration:

1. **Migration files use `CREATE TABLE IF NOT EXISTS`**
   - Safe to run - won't fail if tables exist
   - Will create missing tables if they don't exist

2. **Column changes use `ADD COLUMN IF NOT EXISTS`**
   - Safe to run - won't fail if columns exist
   - Ensures all columns are present

3. **Constraints may need adjustment**
   - Some constraints might need `IF NOT EXISTS` or manual checking

### For New Databases

1. **Disable auto-migration**: `ENABLE_AUTO_MIGRATION=false`
2. **Run all migrations**: `make migrate-up`
3. **All tables created in order** - Guaranteed consistency

## Verification Steps

### 1. Check Migration Coverage

All domain models should have corresponding migration scripts:

**Auto-Migration Models → Migration Status:**
- ✅ `Business` → `businesses` (000001)
- ✅ `User` → `users` (000001)
- ✅ `Cadre` → `cadres` (000001)
- ✅ `EarningComponent` → `earning_components` (000001)
- ✅ `DeductionRule` → `deduction_rules` (000001)
- ✅ `Employee` → `employees` (000001)
- ✅ `PayrollRun` → `payroll_runs` (000001)
- ✅ `PayrollRunEntry` → `payroll_run_entries` (000001)
- ✅ `PayrollRunEntryDetail` → `payroll_run_entry_details` (000001)
- ✅ `Transfer` → `transfers` (000005)

**Migration-Only Models (not in auto-migration):**
- ✅ `BusinessWallet` → `business_wallets` (000008)
- ✅ `WalletTransaction` → `wallet_transactions` (000008)
- ✅ `VFDWebhookNotification` → `vfd_webhook_notifications` (000009) ⭐ NEW
- ✅ `TransferRecord` → `transfer_records` (000009) ⭐ NEW

### 2. Verify Table Structure

Compare migration scripts with domain models:
- Column names match GORM tags
- Data types match (BIGINT for int64, VARCHAR for string, etc.)
- Constraints match (NOT NULL, UNIQUE, FOREIGN KEY)
- Indexes match GORM index definitions

### 3. Test Migration Up/Down

```bash
# Test migration up
make migrate-up

# Verify all tables exist
psql $DB_URL -c "\dt"

# Test migration down (last migration)
make migrate-down

# Verify tables removed
psql $DB_URL -c "\dt"
```

## Best Practices Going Forward

### ✅ DO:
1. **Always create migration scripts** for new tables
2. **Use `IF NOT EXISTS`** for safety (CREATE TABLE, ADD COLUMN)
3. **Include both .up.sql and .down.sql** for reversibility
4. **Test migrations** in dev before production
5. **Document schema changes** in migration comments

### ❌ DON'T:
1. **Don't rely on auto-migration in production**
2. **Don't modify existing migration files** - create new ones
3. **Don't mix auto-migration and traditional migrations** for same tables
4. **Don't skip down migrations** - they're needed for rollbacks

## Migration Order

Current migration sequence ensures dependencies are created in order:

1. **000001**: Core tables (businesses, users, cadres, employees, payroll)
2. **000002**: Add deleted_at columns (soft deletes)
3. **000003**: Add missing timestamp columns
4. **000004**: Fix circular dependency (admin_id FK)
5. **000005**: Add transfers table
6. **000006**: Add business payroll config
7. **000007**: Add description to payroll entry details
8. **000008**: Add wallet tables
9. **000009**: Add VFD tables ⭐ NEW

## Reconciliation Checklist

- [x] Created migration 000009 for VFD tables
- [x] Verified all tables have migration scripts
- [x] Updated MIGRATION_STRATEGY.md with best practices
- [x] Created MIGRATION_AUDIT.md for reference
- [x] Auto-migration disabled by default
- [ ] Test migrations in clean database
- [ ] Verify table structures match domain models
- [ ] Document any remaining inconsistencies

## Next Steps

1. **Run migrations on a clean database** to verify completeness
2. **Compare table structures** (migration vs auto-migration vs domain models)
3. **Fix any structural mismatches** if found
4. **Document final migration strategy** for team
