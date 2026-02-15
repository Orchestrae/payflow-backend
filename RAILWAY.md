# Railway Deployment

## Temporary auto-fixes (in code)
- **Auto-migration** – Enabled automatically when DATABASE_URL contains `railway` (no manual migrate needed)
- **JWT_SECRET** – Uses a temporary default if not set (⚠️ set JWT_SECRET in Railway for production)
- **PORT** – Read automatically from Railway

## Required: One manual step

**DATABASE_URL** – Railway will not inject it until you add the variable reference:

1. Add **Postgres** – Create a Postgres database in your Railway project.
2. **Link it** – In your app service: **Variables** → **Add Variable Reference** → Select your Postgres service → `DATABASE_URL`
   - Or when creating Postgres, use **Connect** / **Add to service** if Railway offers it.

That’s the only Railway-side setup needed.
