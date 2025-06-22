BEGIN;

-- Drop the deferred foreign key constraint
ALTER TABLE businesses DROP CONSTRAINT IF EXISTS fk_admin_user;

-- Make admin_id NOT NULL again
ALTER TABLE businesses ALTER COLUMN admin_id SET NOT NULL;

-- Add back the immediate foreign key constraint
ALTER TABLE businesses ADD CONSTRAINT fk_admin_user 
    FOREIGN KEY (admin_id) REFERENCES users(id) ON DELETE RESTRICT;

COMMIT;
