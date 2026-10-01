-- +goose Up
CREATE TABLE IF NOT EXISTS refresh_tokens (
    token_id    CHAR(36) PRIMARY KEY,
    operator_id CHAR(36) NOT NULL REFERENCES operators(operator_id),
    token_hash  CHAR(64) NOT NULL UNIQUE,
    expires_at  TIMESTAMP NOT NULL,
    revoked_at  TIMESTAMP NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE INDEX idx_refresh_tokens_operator ON refresh_tokens(operator_id);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;