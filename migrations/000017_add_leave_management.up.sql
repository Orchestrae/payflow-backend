-- Leave types
CREATE TABLE IF NOT EXISTS leave_types (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    business_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    default_days INT NOT NULL DEFAULT 20,
    requires_approval BOOLEAN DEFAULT true
);
CREATE INDEX IF NOT EXISTS idx_leave_types_business ON leave_types(business_id) WHERE deleted_at IS NULL;

-- Leave requests
CREATE TABLE IF NOT EXISTS leave_requests (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    employee_id BIGINT NOT NULL,
    business_id BIGINT NOT NULL,
    leave_type_id BIGINT NOT NULL,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    days INT NOT NULL,
    reason VARCHAR(500),
    status VARCHAR(20) DEFAULT 'pending',
    approved_by_id BIGINT
);
CREATE INDEX IF NOT EXISTS idx_leave_requests_employee ON leave_requests(employee_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leave_requests_business ON leave_requests(business_id, created_at DESC) WHERE deleted_at IS NULL;

-- Leave balances
CREATE TABLE IF NOT EXISTS leave_balances (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    employee_id BIGINT NOT NULL,
    leave_type_id BIGINT NOT NULL,
    year INT NOT NULL,
    entitled INT NOT NULL DEFAULT 20,
    used INT DEFAULT 0,
    remaining INT NOT NULL DEFAULT 20,
    UNIQUE(employee_id, leave_type_id, year)
);
