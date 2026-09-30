-- +goose Up
ALTER TABLE operators
    DROP COLUMN name,
    DROP COLUMN email,
    DROP COLUMN phone_number,
    DROP COLUMN last_login_at,
    ADD COLUMN operator_ref_id VARCHAR(255) UNIQUE AFTER operator_id;

-- +goose Down
ALTER TABLE operators
    DROP COLUMN operator_ref_id,
    ADD COLUMN name VARCHAR(255) NOT NULL,
    ADD COLUMN email VARCHAR(255) UNIQUE NOT NULL,
    ADD COLUMN phone_number VARCHAR(15) UNIQUE NOT NULL,
    ADD COLUMN last_login_at TIMESTAMP NULL;