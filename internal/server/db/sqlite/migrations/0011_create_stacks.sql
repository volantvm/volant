-- Create stacks table (unified naming convention)
-- This establishes "stacks" as the primary terminology
-- replacing any references to "deployments"

CREATE TABLE IF NOT EXISTS stacks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    compose_file TEXT,
    environment TEXT, -- JSON encoded environment variables
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,

    CHECK (status IN ('pending', 'running', 'stopped', 'failed', 'updating'))
);

-- Create indexes for common queries
CREATE INDEX idx_stacks_name ON stacks(name);
CREATE INDEX idx_stacks_status ON stacks(status);
CREATE INDEX idx_stacks_created_at ON stacks(created_at);
CREATE INDEX idx_stacks_deleted_at ON stacks(deleted_at);