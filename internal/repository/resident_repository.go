package repository

import (
	"database/sql"

	"contactless-fingerprint-backend/internal/model"
)

type ResidentRepository struct {
	db *sql.DB
}

// Constructor function to create a new instance of ResidentRepository
func NewResidentRepository(db *sql.DB) *ResidentRepository {
	return &ResidentRepository{db: db}
}

func (r *ResidentRepository) FindOrCreateByAadhaarHash(req model.ResidentLookupRequest) (*model.Resident, error) {
	resident := &model.Resident{}
	var captureMode sql.NullString

	// Try to find existing residnet by aadhaar_hash
	query := `
	SELECT resident_pseudonym_id, aadhaar_hash, age_group, gender, skin_tone, capture_mode, created_at
	FROM residents
	WHERE aadhaar_hash = $1
	`
	err := r.db.QueryRow(query, req.AadhaarHash).Scan(
		&resident.ResidentPseudonymID,
		&resident.AadhaarHash,
		&resident.AgeGroup,
		&resident.Gender,
		&resident.SkinTone,
		&captureMode,
		&resident.CreatedAt,
	)

	if err == sql.ErrNoRows {
		// If no existing resident found, create a new one
		insertQuery := `
		INSERT INTO residents (aadhaar_hash, age_group, gender, skin_tone)
		VALUES ($1, $2, $3, $4)
		RETURNING resident_pseudonym_id, aadhaar_hash, age_group, gender, skin_tone, capture_mode, created_at
		`
		err = r.db.QueryRow(insertQuery,
			req.AadhaarHash,
			req.AgeGroup,
			req.Gender,
			req.SkinTone,
		).Scan(
			&resident.ResidentPseudonymID,
			&resident.AadhaarHash,
			&resident.AgeGroup,
			&resident.Gender,
			&resident.SkinTone,
			&captureMode,
			&resident.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		resident.CaptureMode = captureMode.String
		return resident, nil
	}

	if err != nil {
		return nil, err
	}

	resident.CaptureMode = captureMode.String
	return resident, nil
}

// LockCaptureModeTx fetches the resident's capture_mode with a row lock, for
// use inside a transaction that will also decide/set it. The lock is what
// prevents two near-simultaneous first-captures (e.g. the first two images
// of a slap batch upload landing together) from both reading capture_mode
// as unset and racing to set it. Returns (nil, nil) if unset, or
// ErrNotFound if the resident doesn't exist.
func (r *ResidentRepository) LockCaptureModeTx(tx *sql.Tx, residentPseudonymID string) (*string, error) {
	var captureMode sql.NullString

	err := tx.QueryRow(`
		SELECT capture_mode FROM residents
		WHERE resident_pseudonym_id = $1
		FOR UPDATE
	`, residentPseudonymID).Scan(&captureMode)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !captureMode.Valid {
		return nil, nil
	}
	return &captureMode.String, nil
}

// SetCaptureModeTx sets the resident's capture_mode. Only ever called once
// per resident, immediately after LockCaptureModeTx returns nil (unset),
// within the same transaction/lock.
func (r *ResidentRepository) SetCaptureModeTx(tx *sql.Tx, residentID, mode string) error {
	_, err := tx.Exec(`
		UPDATE residents SET capture_mode = $1
		WHERE resident_pseudonym_id = $2
	`, mode, residentID)
	return err
}

// DeleteByAadhaarHash wipes all data for a resident — used in dev/test only
func (r *ResidentRepository) DeleteByAadhaarHash(aadhaarHash string) error {
	_, err := r.db.Exec(
		`DELETE FROM residents WHERE aadhaar_hash = $1`,
		aadhaarHash,
	)
	return err
}
