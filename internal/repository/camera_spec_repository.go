package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"contactless-fingerprint-backend/internal/model"
)

var ErrCameraSpecNotFound = errors.New("camera spec not found")
var ErrDuplicateCameraSpec = errors.New("camera spec with this fingerprint hash already exists")

type CameraSpecRepository struct {
	db *sql.DB
}

func NewCameraSpecRepository(db *sql.DB) *CameraSpecRepository {
	return &CameraSpecRepository{db: db}
}

// FindByFingerprintHash looks up an existing camera_specs row by its
// hardware fingerprint hash. Returns (nil, nil) if not found — NOT an
// error — so callers can distinguish "needs insert" from "query failed".
func (r *CameraSpecRepository) FindByFingerprintHash(hash string) (*model.CameraSpec, error) {
	spec := &model.CameraSpec{}
	var afModesJSON, aeModesJSON, awbModesJSON []byte

	err := r.db.QueryRow(`
		SELECT camera_spec_id, fingerprint_hash, camera_id, lens_facing,
		       hardware_level, sensor_physical_size_mm, sensor_active_array_size,
		       pixel_array_size, focal_length_mm, aperture,
		       min_focus_distance_diopters, hyperfocal_distance_diopters,
		       has_flash, has_ois, max_digital_zoom, sensor_orientation,
		       supports_raw, af_modes, ae_modes, awb_modes, created_at
		FROM camera_specs
		WHERE fingerprint_hash = ?
	`, hash).Scan(
		&spec.CameraSpecID, &spec.FingerprintHash, &spec.CameraID, &spec.LensFacing,
		&spec.HardwareLevel, &spec.SensorPhysicalSizeMM, &spec.SensorActiveArraySize,
		&spec.PixelArraySize, &spec.FocalLengthMM, &spec.Aperture,
		&spec.MinFocusDistanceDiopters, &spec.HyperfocalDistanceDiopters,
		&spec.HasFlash, &spec.HasOIS, &spec.MaxDigitalZoom, &spec.SensorOrientation,
		&spec.SupportsRaw, &afModesJSON, &aeModesJSON, &awbModesJSON, &spec.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := unmarshalModes(afModesJSON, aeModesJSON, awbModesJSON, spec); err != nil {
		return nil, err
	}
	return spec, nil
}

// Insert creates a new camera_specs row. Returns ErrDuplicateCameraSpec
// if fingerprint_hash already exists — callers should FindByFingerprintHash
// first to avoid hitting this in the normal path; this guards the race
// where two devices with identical hardware register concurrently.
func (r *CameraSpecRepository) Insert(spec model.CameraSpec) (*model.CameraSpec, error) {
	// MySQL's JSON columns need real JSON, not a Go slice — af_modes/ae_modes/
	// awb_modes were array columns in Postgres (via pq.Array); here they're
	// marshalled by hand on the way in and out.
	afModesJSON, err := json.Marshal(spec.AfModes)
	if err != nil {
		return nil, err
	}
	aeModesJSON, err := json.Marshal(spec.AeModes)
	if err != nil {
		return nil, err
	}
	awbModesJSON, err := json.Marshal(spec.AwbModes)
	if err != nil {
		return nil, err
	}

	result := &model.CameraSpec{
		CameraSpecID:               uuid.New().String(),
		FingerprintHash:            spec.FingerprintHash,
		CameraID:                   spec.CameraID,
		LensFacing:                 spec.LensFacing,
		HardwareLevel:              spec.HardwareLevel,
		SensorPhysicalSizeMM:       spec.SensorPhysicalSizeMM,
		SensorActiveArraySize:      spec.SensorActiveArraySize,
		PixelArraySize:             spec.PixelArraySize,
		FocalLengthMM:              spec.FocalLengthMM,
		Aperture:                   spec.Aperture,
		MinFocusDistanceDiopters:   spec.MinFocusDistanceDiopters,
		HyperfocalDistanceDiopters: spec.HyperfocalDistanceDiopters,
		HasFlash:                   spec.HasFlash,
		HasOIS:                     spec.HasOIS,
		MaxDigitalZoom:             spec.MaxDigitalZoom,
		SensorOrientation:          spec.SensorOrientation,
		SupportsRaw:                spec.SupportsRaw,
		AfModes:                    spec.AfModes,
		AeModes:                    spec.AeModes,
		AwbModes:                   spec.AwbModes,
		CreatedAt:                  time.Now().UTC(),
	}

	_, err = r.db.Exec(`
		INSERT INTO camera_specs (
			camera_spec_id, fingerprint_hash, camera_id, lens_facing, hardware_level,
			sensor_physical_size_mm, sensor_active_array_size, pixel_array_size,
			focal_length_mm, aperture, min_focus_distance_diopters,
			hyperfocal_distance_diopters, has_flash, has_ois, max_digital_zoom,
			sensor_orientation, supports_raw, af_modes, ae_modes, awb_modes, created_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`,
		result.CameraSpecID, spec.FingerprintHash, spec.CameraID, spec.LensFacing, spec.HardwareLevel,
		spec.SensorPhysicalSizeMM, spec.SensorActiveArraySize, spec.PixelArraySize,
		spec.FocalLengthMM, spec.Aperture, spec.MinFocusDistanceDiopters,
		spec.HyperfocalDistanceDiopters, spec.HasFlash, spec.HasOIS, spec.MaxDigitalZoom,
		spec.SensorOrientation, spec.SupportsRaw, afModesJSON, aeModesJSON, awbModesJSON, result.CreatedAt,
	)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			return nil, ErrDuplicateCameraSpec
		}
		return nil, err
	}
	return result, nil
}

func unmarshalModes(afJSON, aeJSON, awbJSON []byte, spec *model.CameraSpec) error {
	if len(afJSON) > 0 {
		if err := json.Unmarshal(afJSON, &spec.AfModes); err != nil {
			return err
		}
	}
	if len(aeJSON) > 0 {
		if err := json.Unmarshal(aeJSON, &spec.AeModes); err != nil {
			return err
		}
	}
	if len(awbJSON) > 0 {
		if err := json.Unmarshal(awbJSON, &spec.AwbModes); err != nil {
			return err
		}
	}
	return nil
}
