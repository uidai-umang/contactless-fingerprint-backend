package repository

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"

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
	err := r.db.QueryRow(`
		SELECT camera_spec_id, fingerprint_hash, camera_id, lens_facing,
		       hardware_level, sensor_physical_size_mm, sensor_active_array_size,
		       pixel_array_size, focal_length_mm, aperture,
		       min_focus_distance_diopters, hyperfocal_distance_diopters,
		       has_flash, has_ois, max_digital_zoom, sensor_orientation,
		       supports_raw, af_modes, ae_modes, awb_modes, created_at
		FROM camera_specs
		WHERE fingerprint_hash = $1
	`, hash).Scan(
		&spec.CameraSpecID, &spec.FingerprintHash, &spec.CameraID, &spec.LensFacing,
		&spec.HardwareLevel, &spec.SensorPhysicalSizeMM, &spec.SensorActiveArraySize,
		&spec.PixelArraySize, &spec.FocalLengthMM, &spec.Aperture,
		&spec.MinFocusDistanceDiopters, &spec.HyperfocalDistanceDiopters,
		&spec.HasFlash, &spec.HasOIS, &spec.MaxDigitalZoom, &spec.SensorOrientation,
		&spec.SupportsRaw, pq.Array(&spec.AfModes), pq.Array(&spec.AeModes),
		pq.Array(&spec.AwbModes), &spec.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return spec, nil
}

// Insert creates a new camera_specs row. Returns ErrDuplicateCameraSpec
// if fingerprint_hash already exists — callers should FindByFingerprintHash
// first to avoid hitting this in the normal path; this guards the race
// where two devices with identical hardware register concurrently.
func (r *CameraSpecRepository) Insert(spec model.CameraSpec) (*model.CameraSpec, error) {
	result := &model.CameraSpec{}
	err := r.db.QueryRow(`
		INSERT INTO camera_specs (
			fingerprint_hash, camera_id, lens_facing, hardware_level,
			sensor_physical_size_mm, sensor_active_array_size, pixel_array_size,
			focal_length_mm, aperture, min_focus_distance_diopters,
			hyperfocal_distance_diopters, has_flash, has_ois, max_digital_zoom,
			sensor_orientation, supports_raw, af_modes, ae_modes, awb_modes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19
		)
		RETURNING camera_spec_id, fingerprint_hash, camera_id, lens_facing,
		          hardware_level, sensor_physical_size_mm, sensor_active_array_size,
		          pixel_array_size, focal_length_mm, aperture,
		          min_focus_distance_diopters, hyperfocal_distance_diopters,
		          has_flash, has_ois, max_digital_zoom, sensor_orientation,
		          supports_raw, af_modes, ae_modes, awb_modes, created_at
	`,
		spec.FingerprintHash, spec.CameraID, spec.LensFacing, spec.HardwareLevel,
		spec.SensorPhysicalSizeMM, spec.SensorActiveArraySize, spec.PixelArraySize,
		spec.FocalLengthMM, spec.Aperture, spec.MinFocusDistanceDiopters,
		spec.HyperfocalDistanceDiopters, spec.HasFlash, spec.HasOIS, spec.MaxDigitalZoom,
		spec.SensorOrientation, spec.SupportsRaw,
		pq.Array(spec.AfModes), pq.Array(spec.AeModes), pq.Array(spec.AwbModes),
	).Scan(
		&result.CameraSpecID, &result.FingerprintHash, &result.CameraID, &result.LensFacing,
		&result.HardwareLevel, &result.SensorPhysicalSizeMM, &result.SensorActiveArraySize,
		&result.PixelArraySize, &result.FocalLengthMM, &result.Aperture,
		&result.MinFocusDistanceDiopters, &result.HyperfocalDistanceDiopters,
		&result.HasFlash, &result.HasOIS, &result.MaxDigitalZoom, &result.SensorOrientation,
		&result.SupportsRaw, pq.Array(&result.AfModes), pq.Array(&result.AeModes),
		pq.Array(&result.AwbModes), &result.CreatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, ErrDuplicateCameraSpec
		}
		return nil, err
	}
	return result, nil
}
