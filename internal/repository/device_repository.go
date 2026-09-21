package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

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
		WHERE android_id = ?
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
	result := &model.Device{
		DeviceID:             uuid.New().String(),
		OperatorID:           d.OperatorID,
		AndroidID:            d.AndroidID,
		DeviceFingerprint:    d.DeviceFingerprint,
		DeviceModel:          d.DeviceModel,
		DeviceManufacturer:   d.DeviceManufacturer,
		OSVersion:            d.OSVersion,
		PlayIntegrityStatus:  d.PlayIntegrityStatus,
		CameraSpecID:         d.CameraSpecID,
		AndroidSDKVersion:    d.AndroidSDKVersion,
		AndroidSecurityPatch: d.AndroidSecurityPatch,
		SOCModel:             d.SOCModel,
		RAMTotalMB:           d.RAMTotalMB,
		RegisteredAt:         time.Now().UTC(),
	}

	_, err := r.db.Exec(`
		INSERT INTO devices (
			device_id, operator_id, android_id, device_fingerprint, device_model,
			device_manufacturer, os_version, play_integrity_status,
			camera_spec_id, android_sdk_version, android_security_patch,
			soc_model, ram_total_mb, registered_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`,
		result.DeviceID, result.OperatorID, result.AndroidID, result.DeviceFingerprint, result.DeviceModel,
		result.DeviceManufacturer, result.OSVersion, result.PlayIntegrityStatus,
		result.CameraSpecID, result.AndroidSDKVersion, result.AndroidSecurityPatch,
		result.SOCModel, result.RAMTotalMB, result.RegisteredAt,
	)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number == 1062 {
				return nil, ErrDuplicateDevice
			}
			if mysqlErr.Number == 1452 {
				return nil, &ErrForeignKeyViolation{Field: parseFKField(mysqlErr.Message)}
			}
		}
		return nil, err
	}
	return result, nil
}
