-- ENABLE UUID GENERATION
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Stores ASK (AADHAAR SEVA KENDRA) centres information
CREATE TABLE IF NOT EXISTS centres (
    center_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    city VARCHAR(255) NOT NULL,
    state VARCHAR(255) NOT NULL,
    region VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Stores operator accounts linked to a centre
CREATE TABLE IF NOT EXISTS operators (
    operator_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    center_id UUID REFERENCES centres(center_id),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone_number VARCHAR(15) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP
);

-- Stores devices registered to an operator
CREATE TABLE IF NOT EXISTS devices (
    device_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operator_id UUID REFERENCES operators(operator_id),
    android_id VARCHAR(20) UNIQUE NOT NULL,
    device_fingerprint VARCHAR(64),
    device_model VARCHAR(255),
    device_manufacturer VARCHAR(255),
    os_version VARCHAR(50),
    play_integrity_status VARCHAR(20),
    is_flagged BOOLEAN DEFAULT FALSE,
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMP
);

-- Stores resident pseudonym records — no PII stored
CREATE TABLE IF NOT EXISTS residents (
    resident_pseudonym_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    aadhaar_hash VARCHAR(64) UNIQUE NOT NULL,
    age_group VARCHAR(20) CHECK (age_group IN ('5-17', '18-40', '41-60', '60+')),
    gender VARCHAR(10) CHECK (gender IN ('MALE', 'FEMALE', 'OTHER')),
    skin_tone VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Represents one data collection session per resident per operator
CREATE TABLE IF NOT EXISTS sessions (
    session_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operator_id UUID REFERENCES operators(operator_id),
    device_id UUID REFERENCES devices(device_id),
    center_id UUID REFERENCES centres(center_id),
    resident_pseudonym_id UUID REFERENCES residents(resident_pseudonym_id),
    status VARCHAR(20) DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'COMPLETED', 'ABANDONED', 'TIMED_OUT')),
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    close_reason VARCHAR(255)
);

-- Stores resident consent per session — append only, never modified
CREATE TABLE IF NOT EXISTS consents (
    consent_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES sessions(session_id),
    resident_pseudonym_id UUID NOT NULL REFERENCES residents(resident_pseudonym_id),
    consented BOOLEAN NOT NULL,
    language_shown VARCHAR(50),
    operator_id UUID NOT NULL REFERENCES operators(operator_id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Stores one fingerprint capture record per finger per session
-- Image itself is stored in CEPH, only the reference key is stored here
CREATE TABLE IF NOT EXISTS captures (
    capture_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES sessions(session_id),
    resident_pseudonym_id UUID NOT NULL REFERENCES residents(resident_pseudonym_id),
    operator_id UUID NOT NULL REFERENCES operators(operator_id),
    finger_type VARCHAR(20) CHECK (finger_type IN (
        'LEFT_THUMB', 'LEFT_INDEX', 'LEFT_MIDDLE', 'LEFT_RING', 'LEFT_LITTLE',
        'RIGHT_THUMB', 'RIGHT_INDEX', 'RIGHT_MIDDLE', 'RIGHT_RING', 'RIGHT_LITTLE'
    )),
    hand VARCHAR(5) CHECK (hand IN ('LEFT', 'RIGHT')),
    nfiq2_score FLOAT,
    blur_score FLOAT,
    brightness_score FLOAT,
    glare_score FLOAT,
    attempt_count INTEGER DEFAULT 1,
    degraded_flag BOOLEAN DEFAULT false,
    ceph_object_key VARCHAR(500),   -- path to encrypted image in CEPH
    image_checksum VARCHAR(64),     -- SHA-256 of original image for integrity check
    wrapped_dek_ref VARCHAR(255),   -- reference to HSM-managed decryption key
    camera_model VARCHAR(255),
    camera_resolution VARCHAR(50),
    device_model VARCHAR(255),
    upload_status VARCHAR(20) DEFAULT 'PENDING' CHECK (upload_status IN ('PENDING', 'UPLOADED', 'FAILED')),
    upload_attempts INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    uploaded_at TIMESTAMP
);

-- Append-only audit trail — no UPDATE or DELETE ever allowed on this table
CREATE TABLE IF NOT EXISTS audit_logs (
    log_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(50) NOT NULL,
    operator_id UUID,
    session_id UUID,
    device_id UUID,
    payload_hash VARCHAR(64),
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for frequently queried foreign keys
CREATE INDEX IF NOT EXISTS idx_captures_session ON captures(session_id);
CREATE INDEX IF NOT EXISTS idx_captures_resident ON captures(resident_pseudonym_id);
CREATE INDEX IF NOT EXISTS idx_sessions_operator ON sessions(operator_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_session ON audit_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_operator ON audit_logs(operator_id);



