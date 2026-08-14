CREATE TABLE IF NOT EXISTS camera_specs (
    camera_spec_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fingerprint_hash VARCHAR(64) UNIQUE NOT NULL,
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
    af_modes INTEGER[],
    ae_modes INTEGER[],
    awb_modes INTEGER[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE devices ADD COLUMN IF NOT EXISTS camera_spec_id UUID REFERENCES camera_specs(camera_spec_id);
ALTER TABLE devices ADD COLUMN IF NOT EXISTS android_sdk_version INT;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS android_security_patch VARCHAR(20);
ALTER TABLE devices ADD COLUMN IF NOT EXISTS soc_model VARCHAR(255);
ALTER TABLE devices ADD COLUMN IF NOT EXISTS ram_total_mb INT;

ALTER TABLE captures ADD COLUMN IF NOT EXISTS device_id UUID REFERENCES devices(device_id);
ALTER TABLE captures ADD COLUMN IF NOT EXISTS camera_focus_mode VARCHAR(30);
ALTER TABLE captures ADD COLUMN IF NOT EXISTS camera_focus_distance_diopters FLOAT;
ALTER TABLE captures ADD COLUMN IF NOT EXISTS camera_af_state VARCHAR(30);
ALTER TABLE captures ADD COLUMN IF NOT EXISTS camera_ae_state VARCHAR(30);
ALTER TABLE captures ADD COLUMN IF NOT EXISTS camera_awb_state VARCHAR(30);
ALTER TABLE captures ADD COLUMN IF NOT EXISTS camera_iso INT;
ALTER TABLE captures ADD COLUMN IF NOT EXISTS camera_exposure_time_ns BIGINT;
ALTER TABLE captures ADD COLUMN IF NOT EXISTS camera_frame_duration_ns BIGINT;
ALTER TABLE captures ADD COLUMN IF NOT EXISTS camera_zoom_ratio FLOAT;
ALTER TABLE captures ADD COLUMN IF NOT EXISTS camera_flash_state VARCHAR(20);
ALTER TABLE captures ADD COLUMN IF NOT EXISTS camera_rotation INT;

CREATE INDEX IF NOT EXISTS idx_devices_camera_spec ON devices(camera_spec_id);
CREATE INDEX IF NOT EXISTS idx_captures_device ON captures(device_id);
