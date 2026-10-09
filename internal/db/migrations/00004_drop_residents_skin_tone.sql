-- +goose Up
ALTER TABLE residents
    DROP COLUMN skin_tone;

-- +goose Down
ALTER TABLE residents
    ADD COLUMN skin_tone VARCHAR(50);