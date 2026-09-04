CREATE TABLE
    IF NOT EXISTS task_logs (
        id UUID NOT NULL PRIMARY KEY,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP,
        task_id UUID NOT NULL,
        user_id BIGINT,
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

CREATE INDEX IF NOT EXISTS idx_task_logs_task_id ON task_logs (task_id);