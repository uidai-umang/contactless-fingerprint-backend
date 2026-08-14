package repository

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"contactless-fingerprint-backend/internal/model"
)

var ErrDuplicateDevice = errors.New("device with this android_id already registered")

type DeviceRepository struct {
	db *sql.DB
}

func NewDeviceRepository(db *sql.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// FindByAndroidID looks up an existing device registration.
// Returns (nil, nil) if not found.
func (r *DeviceRepository) FindByAndroidID(androidID string) (*model.Device, error) {
	d := &model.Device{}
	err := r.db.QueryRow(`
		SELECT device_id, operator_id, android_id, device_fingerprint,
		       device_model, device_manufacturer, os_version, play_integrity_status,
		       is_flagged, camera_spec_id, android_sdk_version, android_security_patch,
		       soc_model, ram_total_mb, registered_at, last_seen_at
		FROM devices
		WHERE android_id = $1
	`, androidID).Scan(
		&d.DeviceID, &d.OperatorID, &d.AndroidID, &d.DeviceFingerprint,
		&d.DeviceModel, &d.DeviceManufacturer, &d.OSVersion, &d.PlayIntegrityStatus,
		&d.IsFlagged, &d.CameraSpecID, &d.AndroidSDKVersion, &d.AndroidSecurityPatch,
		&d.SOCModel, &d.RAMTotalMB, &d.RegisteredAt, &d.LastSeenAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

// Insert creates a new device row, referencing an already-resolved
// camera_spec_id (the caller — DeviceService — is responsible for the
// find-or-create camera_specs lookup before calling this).
func (r *DeviceRepository) Insert(d model.Device) (*model.Device, error) {
	result := &model.Device{}
	err := r.db.QueryRow(`
		INSERT INTO devices (
			operator_id, android_id, device_fingerprint, device_model,
			device_manufacturer, os_version, play_integrity_status,
			camera_spec_id, android_sdk_version, android_security_patch,
			soc_model, ram_total_mb
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		RETURNING device_id, operator_id, android_id, device_fingerprint,
		          device_model, device_manufacturer, os_version, play_integrity_status,
		          is_flagged, camera_spec_id, android_sdk_version, android_security_patch,
		          soc_model, ram_total_mb, registered_at, last_seen_at
	`,
		d.OperatorID, d.AndroidID, d.DeviceFingerprint, d.DeviceModel,
		d.DeviceManufacturer, d.OSVersion, d.PlayIntegrityStatus,
		d.CameraSpecID, d.AndroidSDKVersion, d.AndroidSecurityPatch,
		d.SOCModel, d.RAMTotalMB,
	).Scan(
		&result.DeviceID, &result.OperatorID, &result.AndroidID, &result.DeviceFingerprint,
		&result.DeviceModel, &result.DeviceManufacturer, &result.OSVersion, &result.PlayIntegrityStatus,
		&result.IsFlagged, &result.CameraSpecID, &result.AndroidSDKVersion, &result.AndroidSecurityPatch,
		&result.SOCModel, &result.RAMTotalMB, &result.RegisteredAt, &result.LastSeenAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return nil, ErrDuplicateDevice
			}
			if pqErr.Code == "23503" {
				return nil, &ErrForeignKeyViolation{Field: parseFKField(pqErr.Constraint)}
			}
		}
		return nil, err
	}
	return result, nil
}
