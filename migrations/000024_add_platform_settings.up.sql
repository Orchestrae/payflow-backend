CREATE TABLE IF NOT EXISTS platform_settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) UNIQUE NOT NULL,
    encrypted_value TEXT NOT NULL DEFAULT '',
    description VARCHAR(500) DEFAULT '',
    category VARCHAR(50) DEFAULT '',
    is_set BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_platform_settings_category ON platform_settings(category);
