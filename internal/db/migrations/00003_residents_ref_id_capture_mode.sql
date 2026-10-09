-- +goose Up
ALTER TABLE residents
    DROP COLUMN aadhaar_hash,
    DROP COLUMN capture_mode,
    DROP COLUMN created_at,
    ADD COLUMN resident_ref_id VARCHAR(64) NOT NULL AFTER resident_pseudonym_id,
    ADD COLUMN dob DATE NOT NULL AFTER resident_ref_id,
    ADD UNIQUE KEY uq_residents_resident_ref_id (resident_ref_id);

ALTER TABLE captures
    ADD COLUMN capture_mode VARCHAR(20) NOT NULL
        CHECK (capture_mode IN ('SEQUENTIAL', 'SLAP')) AFTER hand;

-- +goose Down
ALTER TABLE captures
    DROP COLUMN capture_mode;

ALTER TABLE residents
    DROP INDEX uq_residents_resident_ref_id,
    DROP COLUMN dob,
    DROP COLUMN resident_ref_id,
    ADD COLUMN aadhaar_hash VARCHAR(64) UNIQUE AFTER resident_pseudonym_id,
    ADD COLUMN capture_mode VARCHAR(20) CHECK (capture_mode IN ('SEQUENTIAL', 'SLAP')),
    ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;