# Migration Strategy

## Overview

This project uses a **hybrid migration approach** with a clear separation between development and production:

- **Development/Local**: Auto-migration enabled (optional) for convenience
- **Production**: Traditional migrations only (recommended, safer)

## Current State

### Auto-Migration (GORM)
- **Enabled by**: `ENABLE_AUTO_MIGRATION=true` in `.env`
- **What it creates**: Existing domain models (Business, User, Cadre, Employee, PayrollRun, Transfer, etc.)
- **When it runs**: On every server startup (if enabled)
- **Location**: `internal/platform/database/postgres.go` → `AutoMigrateAll()`

### Traditional Migrations (golang-migrate)
- **What it creates**: All schema changes, including:
  - Initial table structure
  - Column additions/changes
  - New tables (e.g., `business_wallets`, `wallet_transactions`)
  - Indexes and constraints
- **When it runs**: Manually via `make migrate-up` or `migrate` CLI
- **Location**: `migrations/` directory

## ⚠️ Important Notes

### Wallet Tables Are Migration-Only
- `business_wallets` and `wallet_transactions` are **NOT** in auto-migration
- They are **only** created via traditional migration `000008_add_wallet_tables.up.sql`
- This is intentional to ensure consistency across environments

### Safety Recommendations

1. **Development**: 
   - Auto-migration can be enabled for convenience
   - Still run migrations manually to test: `make migrate-up`

2. **Production**: 
   - **ALWAYS** disable auto-migration: `ENABLE_AUTO_MIGRATION=false`
   - **ONLY** use traditional migrations
   - This ensures version control, rollbacks, and consistency

## Configuration

### Enable Auto-Migration (Local Dev Only)
```env
ENABLE_AUTO_MIGRATION=true
```

### Disable Auto-Migration (Production Recommended)
```env
ENABLE_AUTO_MIGRATION=false
```

## Migration Workflow

### 1. Create a New Migration
```bash
make migrate-create
# Enter migration name when prompted
```

### 2. Write Migration Files
- `migrations/XXXXX_migration_name.up.sql` - What to do
- `migrations/XXXXX_migration_name.down.sql` - How to undo it

### 3. Apply Migrations
```bash
# Set DSN in .env or environment
export DSN="postgres://user:password@localhost:5432/dbname?sslmode=disable"

# Apply all pending migrations
make migrate-up

# Or use migrate CLI directly
migrate -database "$DSN" -path ./migrations up
```

### 4. Rollback if Needed
```bash
# Rollback last migration
make migrate-down

# Or rollback specific number
migrate -database "$DSN" -path ./migrations down 1
```

## Best Practices

1. **Always create both `.up.sql` and `.down.sql` files** - Ensures reversible migrations
2. **Test migrations in dev before production** - Run both up and down
3. **Never modify existing migration files** - Create new migrations for changes
4. **Use transactions in migrations** - Wrap changes in `BEGIN; ... COMMIT;` where appropriate
5. **Version control all migrations** - All migration files should be committed to git

## Migration Status

Current migrations:
- `000001_create_initial_tables` - Core tables (businesses, users, cadres, employees, etc.)
- `000002_add_deleted_at_columns` - Soft delete support
- `000003_add_missing_timestamp_columns` - Additional timestamps
- `000004_fix_circular_dependency` - Foreign key fixes
- `000005_add_transfers_table` - Transfer table (redundant with auto-migration)
- `000006_add_business_payroll_config` - Payroll configuration fields
- `000007_add_description_to_payroll_entry_details` - Payroll entry details enhancement
- `000008_add_wallet_tables` - **Wallet tables (migration-only, not in auto-migration)**

## Reconciliation Strategy

If you're moving from auto-migration to traditional migrations:

1. **For existing databases**: 
   - Auto-migration already created tables
   - Traditional migrations will check for existence and skip if already present (using `IF NOT EXISTS`)
   - No conflicts expected

2. **For new databases**:
   - Disable auto-migration: `ENABLE_AUTO_MIGRATION=false`
   - Run all migrations: `make migrate-up`
   - Tables will be created in order from migration files

3. **Future new tables**:
   - **Always** create traditional migration first
   - **Optionally** add to auto-migration if you want dev convenience (not recommended)
   - **Never** rely on auto-migration alone in production

## Troubleshooting

### "Table already exists" errors
- If table exists from auto-migration, migration will fail
- Solution: Add `IF NOT EXISTS` to migration, or drop and recreate

### "Migration version mismatch"
- Check current migration version: `migrate version -database "$DSN"`
- Force to specific version if needed: `migrate force -database "$DSN" -version X`

### Mixing auto-migration and traditional migrations
- **Recommendation**: Choose one approach per environment
- **Development**: Auto-migration OK (for convenience)
- **Production**: Traditional migrations only (for safety)
