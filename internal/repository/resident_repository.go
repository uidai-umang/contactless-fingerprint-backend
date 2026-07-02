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

	// Try to find existing residnet by aadhaar_hash
	query := ` 
	SELECT resident_pseudonym_id, aadhaar_hash, age_group, gender, skin_tone, created_at
	FROM residents
	WHERE aadhaar_hash = $1
	`
	err := r.db.QueryRow(query, req.AadhaarHash).Scan(
		&resident.ResidentPseudonymID,
		&resident.AadhaarHash,
		&resident.AgeGroup,
		&resident.Gender,
		&resident.SkinTone,
		&resident.CreatedAt,
	)

	if err == sql.ErrNoRows {
		// If no existing resident found, create a new one
		insertQuery := `
		INSERT INTO residents (aadhaar_hash, age_group, gender, skin_tone)
		VALUES ($1, $2, $3, $4)
		RETURNING resident_pseudonym_id, aadhaar_hash, age_group, skin_tone, created_at
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
			&resident.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		return resident, nil
	}

	if err != nil {
		return nil, err
	}

	return resident, nil
}

// DeleteByAadhaarHash wipes all data for a resident — used in dev/test only
func (r *ResidentRepository) DeleteByAadhaarHash(aadhaarHash string) error {
	_, err := r.db.Exec(
		`DELETE FROM residents WHERE aadhaar_hash = $1`,
		aadhaarHash,
	)
	return err
}
