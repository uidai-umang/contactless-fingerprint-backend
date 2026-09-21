-- MySQL schema. IDs are CHAR(36) UUID strings generated in Go (google/uuid),
-- not DB-generated -- MySQL has no built-in uuid_generate_v4() equivalent,
-- and app-generated IDs let us skip RETURNING (MySQL doesn't support it).

-- Stores operator accounts
CREATE TABLE IF NOT EXISTS operators (
    operator_id CHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone_number VARCHAR(15) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP NULL
) ENGINE=InnoDB;

-- Stores camera hardware specifications, deduped by fingerprint hash.
-- One row covers every device sharing the same physical camera module --
-- static properties (sensor size, focal length, etc.) never change per
-- device or per capture, so they live here once, not repeated elsewhere.
CREATE TABLE IF NOT EXISTS camera_specs (
    camera_spec_id CHAR(36) PRIMARY KEY,
    fingerprint_hash VARCHAR(64) UNIQUE NOT NULL,  -- hash of manufacturer+model+
                                                     -- hardware_level+sensor_size+
                                                     -- focal_length+aperture+camera_id
    camera_id VARCHAR(10),
    lens_facing VARCHAR(10),
    hardware_level VARCHAR(20),
    sensor_physical_size_mm VARCHAR(30),
    sensor_active_array_size VARCHAR(30),
    pixel_array_size VARCHAR(30),
    focal_length_mm FLOAT,
    aperture FLOAT,
    min_focus_distance_diopters FLOAT,
    hyperfocal_distance_diopters FLOAT,
    has_flash BOOLEAN,
    has_ois BOOLEAN,
    max_digital_zoom FLOAT,
    sensor_orientation INT,
    supports_raw BOOLEAN,
    af_modes JSON,   -- MySQL has no array type; stored as e.g. [1,2,3]
    ae_modes JSON,
    awb_modes JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- Stores devices registered to an operator
CREATE TABLE IF NOT EXISTS devices (
    device_id CHAR(36) PRIMARY KEY,
    operator_id CHAR(36) REFERENCES operators(operator_id),
    android_id VARCHAR(20) UNIQUE NOT NULL,
    device_fingerprint VARCHAR(64),
    device_model VARCHAR(255),
    device_manufacturer VARCHAR(255),
    os_version VARCHAR(50),
    play_integrity_status VARCHAR(20),
    is_flagged BOOLEAN DEFAULT FALSE,
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMP NULL,
    camera_spec_id CHAR(36) REFERENCES camera_specs(camera_spec_id),
    android_sdk_version INT,
    android_security_patch VARCHAR(20),
    soc_model VARCHAR(255),
    ram_total_mb INT
) ENGINE=InnoDB;

-- Stores resident pseudonym records -- no PII stored
CREATE TABLE IF NOT EXISTS residents (
    resident_pseudonym_id CHAR(36) PRIMARY KEY,
    aadhaar_hash VARCHAR(64) UNIQUE NOT NULL,
    age_group VARCHAR(20) CHECK (age_group IN ('5-17', '18-40', '41-60', '60+')),
    gender VARCHAR(10) CHECK (gender IN ('MALE', 'FEMALE', 'OTHER')),
    skin_tone VARCHAR(50),
    capture_mode VARCHAR(20) CHECK (capture_mode IN ('SEQUENTIAL', 'SLAP')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- Resident consent -- no session concept, tied to resident + operator directly
CREATE TABLE IF NOT EXISTS consents (
    consent_id CHAR(36) PRIMARY KEY,
    resident_pseudonym_id CHAR(36) NOT NULL REFERENCES residents(resident_pseudonym_id),
    consented BOOLEAN NOT NULL,
    language_shown VARCHAR(50),
    operator_id CHAR(36) NOT NULL REFERENCES operators(operator_id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- One capture record per finger per resident. Image lives in CEPH/S3, only the key is stored here.
CREATE TABLE IF NOT EXISTS captures (
    capture_id CHAR(36) PRIMARY KEY,
    resident_pseudonym_id CHAR(36) NOT NULL REFERENCES residents(resident_pseudonym_id),
    operator_id CHAR(36) NOT NULL REFERENCES operators(operator_id),
    finger_type VARCHAR(20) CHECK (finger_type IN (
        'LEFT_THUMB', 'LEFT_INDEX', 'LEFT_MIDDLE', 'LEFT_RING', 'LEFT_LITTLE',
        'RIGHT_THUMB', 'RIGHT_INDEX', 'RIGHT_MIDDLE', 'RIGHT_RING', 'RIGHT_LITTLE',
        'LEFT_SLAP', 'RIGHT_SLAP'
    )),
    hand VARCHAR(5) CHECK (hand IN ('LEFT', 'RIGHT')),
    nfiq2_score FLOAT,
    blur_score FLOAT,
    brightness_score FLOAT,
    glare_score FLOAT,
    attempt_count INTEGER DEFAULT 1,
    degraded_flag BOOLEAN DEFAULT false,
    ceph_object_key VARCHAR(500),   -- path to encrypted image in CEPH/S3
    image_checksum VARCHAR(64),     -- SHA-256 of original image for integrity check
    wrapped_dek_ref VARCHAR(255),   -- reference to HSM-managed decryption key
    camera_model VARCHAR(255),
    camera_resolution VARCHAR(50),
    device_model VARCHAR(255),
    upload_status VARCHAR(20) DEFAULT 'PENDING' CHECK (upload_status IN ('PENDING', 'UPLOADED', 'FAILED')),
    upload_attempts INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    uploaded_at TIMESTAMP NULL,
    device_id CHAR(36) REFERENCES devices(device_id),
    camera_focus_mode VARCHAR(30),
    camera_focus_distance_diopters FLOAT,
    camera_af_state VARCHAR(30),
    camera_ae_state VARCHAR(30),
    camera_awb_state VARCHAR(30),
    camera_iso INT,
    camera_exposure_time_ns BIGINT,
    camera_frame_duration_ns BIGINT,
    camera_zoom_ratio FLOAT,
    camera_flash_state VARCHAR(20),
    camera_rotation INT,
    capture_strategy VARCHAR(20),
    focus_type VARCHAR(50),
    -- Generated column + unique index below reproduce Postgres's partial
    -- unique index (`WHERE upload_status = 'UPLOADED'`) -- MySQL has no
    -- partial index, but a unique index ignores multiple NULLs, so this
    -- column is only non-NULL (and thus only enforced unique) for UPLOADED rows.
    uploaded_finger_key VARCHAR(60) GENERATED ALWAYS AS (
        CASE WHEN upload_status = 'UPLOADED' THEN CONCAT(resident_pseudonym_id, ':', finger_type) END
    ) STORED,
    UNIQUE KEY unique_uploaded_finger_per_resident (uploaded_finger_key)
) ENGINE=InnoDB;

-- Append-only audit trail -- no UPDATE or DELETE ever allowed on this table
CREATE TABLE IF NOT EXISTS audit_logs (
    log_id CHAR(36) PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    operator_id CHAR(36),
    device_id CHAR(36),
    payload_hash VARCHAR(64),
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- Indexes for frequently queried foreign keys
CREATE INDEX idx_captures_resident ON captures(resident_pseudonym_id);
CREATE INDEX idx_captures_operator ON captures(operator_id);
CREATE INDEX idx_audit_logs_operator ON audit_logs(operator_id);
CREATE INDEX idx_devices_camera_spec ON devices(camera_spec_id);
CREATE INDEX idx_captures_device ON captures(device_id);

-- Stores national capture targets per demographic dimension/key.
-- `key` is backtick-quoted throughout this file and in Go queries --
-- it's a reserved word in MySQL (wasn't in Postgres).
CREATE TABLE IF NOT EXISTS quota_targets (
    dimension VARCHAR(20) CHECK (dimension IN ('GENDER', 'AGE_GROUP')),
    `key` VARCHAR(20) NOT NULL,
    target_count INTEGER CHECK (target_count >= 0),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (dimension, `key`)
) ENGINE=InnoDB;

-- Records each time an operator captures a resident against a full/near-full quota bracket
CREATE TABLE IF NOT EXISTS quota_overrides (
    override_id CHAR(36) PRIMARY KEY,
    resident_pseudonym_id CHAR(36) NOT NULL REFERENCES residents(resident_pseudonym_id),
    operator_id CHAR(36) NOT NULL REFERENCES operators(operator_id),
    dimension VARCHAR(20) CHECK (dimension IN ('GENDER', 'AGE_GROUP')),
    `key` VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE INDEX idx_quota_overrides_dimension_key ON quota_overrides(dimension, `key`);
CREATE INDEX idx_quota_overrides_operator ON quota_overrides(operator_id);