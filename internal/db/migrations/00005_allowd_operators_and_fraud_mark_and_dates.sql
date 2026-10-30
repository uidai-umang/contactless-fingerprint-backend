-- +goose Up
ALTER TABLE operators
    CHANGE COLUMN created_at date_created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN date_modified TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP AFTER date_created,
    ADD COLUMN is_fraud BOOLEAN NOT NULL DEFAULT FALSE AFTER status;

UPDATE operators SET date_modified = date_created;

CREATE TABLE IF NOT EXISTS allowed_operators (
    operator_ref_id VARCHAR(255) PRIMARY KEY,
    date_created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- +goose Down
DROP TABLE IF EXISTS allowed_operators;

ALTER TABLE operators
    DROP COLUMN is_fraud,
    DROP COLUMN date_modified,
    CHANGE COLUMN date_created created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
