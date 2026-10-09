package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"contactless-fingerprint-backend/internal/model"
)

type ResidentRepository struct {
	db *sql.DB
}

// Constructor function to create a new instance of ResidentRepository
func NewResidentRepository(db *sql.DB) *ResidentRepository {
	return &ResidentRepository{db: db}
}

// FindByRefID looks a resident up by the Mitra resident ref id.
// dob is deliberately not selected -- it is stored but never read back.
func (r *ResidentRepository) FindByRefID(refID string) (*model.Resident, error) {
	resident := &model.Resident{ResidentRefID: refID}

	err := r.db.QueryRow(`
		SELECT resident_pseudonym_id, COALESCE(age_group, ''), COALESCE(gender, '')
		FROM residents
		WHERE resident_ref_id = ?
	`, refID).Scan(
		&resident.ResidentPseudonymID,
		&resident.AgeGroup,
		&resident.Gender,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return resident, nil
}

// FindOrCreateByRefID returns the existing resident for refID, or creates one.
// First write wins: if the resident already exists, the incoming dob / gender /
// ageGroup are ignored and the stored values are returned.
func (r *ResidentRepository) FindOrCreateByRefID(refID string, dob time.Time, gender, ageGroup string) (*model.Resident, error) {
	existing, err := r.FindByRefID(refID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	// ID is generated here in Go -- MySQL has no RETURNING clause.
	resident := &model.Resident{
		ResidentPseudonymID: uuid.New().String(),
		ResidentRefID:       refID,
		AgeGroup:            ageGroup,
		Gender:              gender,
	}

	_, err = r.db.Exec(`
		INSERT INTO residents (resident_pseudonym_id, resident_ref_id, dob, gender, age_group)
		VALUES (?, ?, ?, ?, ?)
	`,
		resident.ResidentPseudonymID,
		refID,
		dob.Format("2006-01-02"),
		gender,
		ageGroup,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			// Another request inserted the same ref id between our SELECT and
			// INSERT -- return that row instead of failing.
			return r.FindByRefID(refID)
		}
		return nil, err
	}

	return resident, nil
}

// LockResidentTx takes a row lock on the resident for the rest of the
// transaction. This is what serialises two near-simultaneous captures for the
// same resident (e.g. the images of a slap batch), so both can't pass the
// capture-mode check before either has inserted. Returns ErrNotFound if the
// resident doesn't exist.
func (r *ResidentRepository) LockResidentTx(tx *sql.Tx, residentPseudonymID string) error {
	var id string
	err := tx.QueryRow(`
		SELECT resident_pseudonym_id FROM residents
		WHERE resident_pseudonym_id = ?
		FOR UPDATE
	`, residentPseudonymID).Scan(&id)

	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}

// DeleteByRefID wipes a resident -- used in dev/test only
func (r *ResidentRepository) DeleteByRefID(refID string) error {
	_, err := r.db.Exec(
		`DELETE FROM residents WHERE resident_ref_id = ?`,
		refID,
	)
	return err
}
