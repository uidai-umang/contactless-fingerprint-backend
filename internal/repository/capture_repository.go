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

func (r *CaptureRepository) ExistsUploaded(residentPseudonymID, fingerType string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM captures
			WHERE resident_pseudonym_id = $1
			  AND finger_type = $2
			  AND upload_status = 'UPLOADED'
		)
	`, residentPseudonymID, fingerType).Scan(&exists)
	return exists, err
}

func (r *CaptureRepository) ExistsUploadedTx(tx *sql.Tx, residentPseudonymID, fingerType string) (bool, error) {
	var exists bool
	err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM captures
			WHERE resident_pseudonym_id = $1
			  AND finger_type = $2
			  AND upload_status = 'UPLOADED'
		)
	`, residentPseudonymID, fingerType).Scan(&exists)
	return exists, err
}

func (r *CaptureRepository) Insert(req model.CaptureRequest, cephKey string) (*model.Capture, error) {
	return r.insert(r.db, req, cephKey)
}

func (r *CaptureRepository) InsertTx(tx *sql.Tx, req model.CaptureRequest, cephKey string) (*model.Capture, error) {
	return r.insert(tx, req, cephKey)
}

type queryRower interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

func (r *CaptureRepository) insert(q queryRower, req model.CaptureRequest, cephKey string) (*model.Capture, error) {
	capture := &model.Capture{}

	query := `
		INSERT INTO captures (
			resident_pseudonym_id, operator_id,
			finger_type, hand, nfiq2_score, blur_score,
			brightness_score, glare_score, attempt_count,
			degraded_flag, ceph_object_key, image_checksum,
			camera_model, camera_resolution, device_model,
			upload_status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, 'UPLOADED'
		)
		RETURNING capture_id, resident_pseudonym_id,
		          operator_id, finger_type, hand, nfiq2_score,
		          blur_score, brightness_score, glare_score,
		          attempt_count, degraded_flag, ceph_object_key,
		          image_checksum, camera_model, camera_resolution,
		          device_model, upload_status, created_at
	`

	err := q.QueryRow(query,
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

func (r *CaptureRepository) GetByResidentID(residentID string) ([]model.Capture, error) {
	rows, err := r.db.Query(`
		SELECT capture_id, resident_pseudonym_id,
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

// GenerateCephKey builds the CEPH storage path: {residentID}/{fingerType}_{timestamp}.jp2
func GenerateCephKey(residentID, fingerType string) string {
	timestamp := time.Now().UTC().Format("20060102T150405")
	return fmt.Sprintf("/sitaa-clf/%s/%s_%s.jp2", residentID, fingerType, timestamp)
}
