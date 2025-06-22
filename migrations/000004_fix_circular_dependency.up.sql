BEGIN;

-- Drop the existing foreign key constraint
ALTER TABLE businesses DROP CONSTRAINT IF EXISTS fk_admin_user;

-- Make admin_id nullable temporarily to allow business creation without user
ALTER TABLE businesses ALTER COLUMN admin_id DROP NOT NULL;

-- Add a deferred foreign key constraint that will be checked at transaction commit
ALTER TABLE businesses ADD CONSTRAINT fk_admin_user 
    FOREIGN KEY (admin_id) REFERENCES users(id) ON DELETE RESTRICT 
    DEFERRABLE INITIALLY DEFERRED;

COMMIT;
