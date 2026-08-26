-- Stores national capture targets per demographic dimension/key
CREATE TABLE IF NOT EXISTS quota_targets (
    dimension VARCHAR(20) CHECK (dimension IN ('GENDER', 'AGE_GROUP')),
    key VARCHAR(20) NOT NULL,
    target_count INTEGER CHECK (target_count >= 0),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (dimension, key)
);

-- Records each time an operator captures a resident against a full/near-full quota bracket
CREATE TABLE IF NOT EXISTS quota_overrides (
    override_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES sessions(session_id),
    resident_pseudonym_id UUID NOT NULL REFERENCES residents(resident_pseudonym_id),
    operator_id UUID NOT NULL REFERENCES operators(operator_id),
    dimension VARCHAR(20) CHECK (dimension IN ('GENDER', 'AGE_GROUP')),
    key VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quota_overrides_dimension_key ON quota_overrides(dimension, key);
CREATE INDEX IF NOT EXISTS idx_quota_overrides_operator ON quota_overrides(operator_id);

-- Seed placeholder national targets
INSERT INTO quota_targets (dimension, key, target_count) VALUES
    ('GENDER', 'MALE', 150000),
    ('GENDER', 'FEMALE', 150000),
    ('GENDER', 'OTHER', 1000),
    ('AGE_GROUP', '5-17', 60000),
    ('AGE_GROUP', '18-40', 120000),
    ('AGE_GROUP', '41-60', 80000),
    ('AGE_GROUP', '60+', 40000)
ON CONFLICT DO NOTHING;
