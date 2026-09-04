CREATE TABLE IF NOT EXISTS user_role (
    id SERIAL NOT NULL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP,
    user_id INT NOT NULL REFERENCES users (id),
    role_id INT NOT NULL REFERENCES "role" (id),
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_role_user_role ON user_role (user_id, role_id);
