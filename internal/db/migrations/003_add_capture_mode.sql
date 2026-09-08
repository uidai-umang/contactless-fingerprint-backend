ALTER TABLE residents ADD COLUMN IF NOT EXISTS capture_mode VARCHAR(20)
    CHECK (capture_mode IN ('SEQUENTIAL', 'SLAP'));
