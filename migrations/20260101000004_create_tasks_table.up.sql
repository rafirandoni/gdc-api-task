CREATE TABLE IF NOT EXISTS tasks (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP,
    user_id BIGINT,
    idempotency_key VARCHAR(150) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(16) NOT NULL DEFAULT 'BACKLOG' CHECK (
        status IN (
            'BACKLOG',
            'ON_PROGRESS',
            'ON_REVIEW',
            'FEEDBACK',
            'FINISHED'
        )
    )
);

CREATE INDEX IF NOT EXISTS idx_tasks_idempotency ON tasks (idempotency_key);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks (status);
