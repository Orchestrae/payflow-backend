# Migration Audit & Reconciliation Plan

## Current State Analysis

### Tables in Auto-Migration (`AutoMigrateAll()`)
Located in: `internal/platform/database/postgres.go:158-169`

1. ✅ `Business` → `businesses` table
2. ✅ `User` → `users` table  
3. ✅ `Cadre` → `cadres` table
4. ✅ `EarningComponent` → `earning_components` table
5. ✅ `DeductionRule` → `deduction_rules` table
6. ✅ `Employee` → `employees` table
7. ✅ `PayrollRun` → `payroll_runs` table
8. ✅ `PayrollRunEntry` → `payroll_run_entries` table
9. ✅ `PayrollRunEntryDetail` → `payroll_run_entry_details` table
10. ✅ `Transfer` → `transfers` table

### Tables in Traditional Migrations

**From `000001_create_initial_tables.up.sql`:**
- ✅ `businesses` - FULL MIGRATION
- ✅ `users` - FULL MIGRATION
- ✅ `deduction_rules` - FULL MIGRATION
- ✅ `cadres` - FULL MIGRATION
- ✅ `cadre_deduction_rules` - JOIN TABLE (only in migration)
- ✅ `earning_components` - FULL MIGRATION
- ✅ `employees` - FULL MIGRATION
- ✅ `payroll_runs` - FULL MIGRATION
- ✅ `payroll_run_entries` - FULL MIGRATION
- ✅ `payroll_run_entry_details` - FULL MIGRATION

**From `000002_add_deleted_at_columns.up.sql`:**
- ✅ Adds `deleted_at` columns to all above tables

**From `000003_add_missing_timestamp_columns.up.sql`:**
- ✅ Adds `created_at`, `updated_at` to `earning_components`
- ✅ Adds `created_at`, `updated_at` to `payroll_run_entries`
- ✅ Adds `created_at`, `updated_at` to `payroll_run_entry_details`

**From `000004_fix_circular_dependency.up.sql`:**
- ✅ Fixes `businesses.admin_id` foreign key constraint

**From `000005_add_transfers_table.up.sql`:**
- ✅ `transfers` - FULL MIGRATION (redundant with auto-migration)

**From `000006_add_business_payroll_config.up.sql`:**
- ✅ Adds `payroll_requires_approval`, `payroll_auto_process` to `businesses`

**From `000007_add_description_to_payroll_entry_details.up.sql`:**
- ✅ Adds `description` column to `payroll_run_entry_details`

**From `000008_add_wallet_tables.up.sql`:**
- ✅ `business_wallets` - **MIGRATION ONLY** (not in auto-migration) ✅
- ✅ `wallet_transactions` - **MIGRATION ONLY** (not in auto-migration) ✅

### ⚠️ MISSING FROM TRADITIONAL MIGRATIONS

#### 1. `vfd_webhook_notifications` table
- **Status**: ❌ **NO MIGRATION SCRIPT**
- **Created by**: Auto-migration only (if enabled)
- **Location**: `internal/repository/postgres/vfd_webhook_repo.go:11-28`
- **Impact**: If auto-migration is disabled, this table won't exist!

#### 2. `vfd_transfer_records` table (or similar name)
- **Status**: ❌ **NO MIGRATION SCRIPT**
- **Created by**: Auto-migration only (if enabled)
- **Location**: `internal/repository/postgres/vfd_transfer_repo.go:11-37`
- **Note**: This repository is marked as deprecated (use `TransferRepository` instead)
- **Impact**: Low (deprecated), but still a gap

### ✅ Tables Correctly Covered

All core tables have migration scripts:
- ✅ `businesses` - Migration 000001
- ✅ `users` - Migration 000001
- ✅ `cadres` - Migration 000001
- ✅ `earning_components` - Migration 000001
- ✅ `employees` - Migration 000001
- ✅ `payroll_runs` - Migration 000001
- ✅ `payroll_run_entries` - Migration 000001
- ✅ `payroll_run_entry_details` - Migration 000001
- ✅ `transfers` - Migration 000005
- ✅ `business_wallets` - Migration 000008
- ✅ `wallet_transactions` - Migration 000008

## Reconciliation Plan

### Option 1: Create Missing Migrations (Recommended)

Create migration scripts for the missing tables to ensure they exist in production.

**Migration 000009**: Add VFD webhook notifications table
**Migration 000010**: Add VFD transfer records table (optional, if still needed)

### Option 2: Add Missing Tables to Existing Migration

Since these are older tables, we could add them to a "catch-up" migration.

### Option 3: Remove/Deprecate Unused Tables

If VFD tables are truly deprecated and not used, we can document that they're legacy.

## Recommended Action Plan

1. ✅ **Current state is mostly safe** - Core tables are covered
2. ⚠️ **Create migration for `vfd_webhook_notifications`** - This is actively used
3. ⚠️ **Create migration for `vfd_transfer_records`** - Only if still needed (marked deprecated)
4. ✅ **Wallet tables are migration-only** - This is correct and intentional
5. ✅ **Enable auto-migration flag** - Disabled by default for production safety

## Next Steps

1. Audit actual database to see what tables exist
2. Create migration scripts for missing tables
3. Test migration up/down
4. Update documentation
