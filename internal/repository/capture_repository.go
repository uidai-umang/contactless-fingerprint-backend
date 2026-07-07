package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"contactless-fingerprint-backend/internal/model"
)

type CaptureRepository struct {
	db *sql.DB
}

func NewCaptureRepository(db *sql.DB) *CaptureRepository {
	return &CaptureRepository{db: db}
}

// ExistsUploaded reports whether a capture with UPLOADED status already exists
// for the given (session_id, resident_pseudonym_id, finger_type) combination.
func (r *CaptureRepository) ExistsUploaded(sessionID, residentPseudonymID, fingerType string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM captures
			WHERE session_id = $1
			  AND resident_pseudonym_id = $2
			  AND finger_type = $3
			  AND upload_status = 'UPLOADED'
		)
	`, sessionID, residentPseudonymID, fingerType).Scan(&exists)
	return exists, err
}

// Insert saves a single capture record and returns it with generated capture_id.
// Returns *ErrForeignKeyViolation if any referenced FK does not exist.
func (r *CaptureRepository) Insert(req model.CaptureRequest, cephKey string) (*model.Capture, error) {
	capture := &model.Capture{}

	query := `
		INSERT INTO captures (
			session_id, resident_pseudonym_id, operator_id,
			finger_type, hand, nfiq2_score, blur_score,
			brightness_score, glare_score, attempt_count,
			degraded_flag, ceph_object_key, image_checksum,
			camera_model, camera_resolution, device_model,
			upload_status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, 'UPLOADED'
		)
		RETURNING capture_id, session_id, resident_pseudonym_id,
		          operator_id, finger_type, hand, nfiq2_score,
		          blur_score, brightness_score, glare_score,
		          attempt_count, degraded_flag, ceph_object_key,
		          image_checksum, camera_model, camera_resolution,
		          device_model, upload_status, created_at
	`

	err := r.db.QueryRow(query,
		req.SessionID,
		req.ResidentPseudonymID,
		req.OperatorID,
		req.FingerType,
		req.Hand,
		req.Nfiq2Score,
		req.BlurScore,
		req.BrightnessScore,
		req.GlareScore,
		req.AttemptCount,
		req.DegradedFlag,
		cephKey,
		req.ImageChecksum,
		req.CameraModel,
		req.CameraResolution,
		req.DeviceModel,
	).Scan(
		&capture.CaptureID,
		&capture.SessionID,
		&capture.ResidentPseudonymID,
		&capture.OperatorID,
		&capture.FingerType,
		&capture.Hand,
		&capture.Nfiq2Score,
		&capture.BlurScore,
		&capture.BrightnessScore,
		&capture.GlareScore,
		&capture.AttemptCount,
		&capture.DegradedFlag,
		&capture.CephObjectKey,
		&capture.ImageChecksum,
		&capture.CameraModel,
		&capture.CameraResolution,
		&capture.DeviceModel,
		&capture.UploadStatus,
		&capture.CreatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23503" {
				return nil, &ErrForeignKeyViolation{Field: parseFKField(pqErr.Constraint)}
			}
			if pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "unique_uploaded_finger_per_resident") {
				return nil, ErrDuplicateCapture
			}
		}
		return nil, err
	}

	return capture, nil
}

// GetByResidentID returns all captures for a resident
func (r *CaptureRepository) GetByResidentID(residentID string) ([]model.Capture, error) {
	rows, err := r.db.Query(`
		SELECT capture_id, session_id, resident_pseudonym_id,
		       operator_id, finger_type, hand, nfiq2_score,
		       blur_score, brightness_score, glare_score,
		       attempt_count, degraded_flag, upload_status, created_at
		FROM captures
		WHERE resident_pseudonym_id = $1
		ORDER BY created_at ASC
	`, residentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var captures []model.Capture
	for rows.Next() {
		c := model.Capture{}
		err := rows.Scan(
			&c.CaptureID,
			&c.SessionID,
			&c.ResidentPseudonymID,
			&c.OperatorID,
			&c.FingerType,
			&c.Hand,
			&c.Nfiq2Score,
			&c.BlurScore,
			&c.BrightnessScore,
			&c.GlareScore,
			&c.AttemptCount,
			&c.DegradedFlag,
			&c.UploadStatus,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		captures = append(captures, c)
	}

	return captures, nil
}

// GetPendingByResidentID returns captures with PENDING upload status
func (r *CaptureRepository) GetPendingByResidentID(residentID string) ([]model.Capture, error) {
	rows, err := r.db.Query(`
		SELECT capture_id, finger_type, hand, upload_status
		FROM captures
		WHERE resident_pseudonym_id = $1
		AND upload_status = 'PENDING'
	`, residentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var captures []model.Capture
	for rows.Next() {
		c := model.Capture{}
		err := rows.Scan(
			&c.CaptureID,
			&c.FingerType,
			&c.Hand,
			&c.UploadStatus,
		)
		if err != nil {
			return nil, err
		}
		captures = append(captures, c)
	}

	return captures, nil
}

// GenerateCephKey builds the CEPH object storage path for an image
func GenerateCephKey(centreID, residentID, sessionID, fingerType string) string {
	timestamp := time.Now().UTC().Format("20060102T150405")
	return fmt.Sprintf("/sitaa-clf/%s/%s/%s/%s_%s.enc",
		centreID, residentID, sessionID, fingerType, timestamp)
}

