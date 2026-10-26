package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type ConsentRepository struct {
	db *sql.DB
}

func NewConsentRepository(db *sql.DB) *ConsentRepository {
	return &ConsentRepository{db: db}
}

// latest returns the resident's most recent consent decision.
// found is false when the resident has no consent record yet.
func (r *ConsentRepository) latest(residentPseudonymID string) (consented bool, found bool, err error) {
	err = r.db.QueryRow(`
		SELECT consented
		FROM consents
		WHERE resident_pseudonym_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, residentPseudonymID).Scan(&consented)
	if err == sql.ErrNoRows {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return consented, true, nil
}

// IsConsented reports whether the resident's most recent consent decision is yes.
// A resident with no consent record is not consented.
func (r *ConsentRepository) IsConsented(residentPseudonymID string) (bool, error) {
	consented, _, err := r.latest(residentPseudonymID)
	return consented, err
}

// Record stores a consent decision. If the latest decision is already the same,
// nothing is inserted, so a retried request does not add duplicate rows.
func (r *ConsentRepository) Record(residentPseudonymID, operatorID, languageShown string, consented bool) error {
	current, found, err := r.latest(residentPseudonymID)
	if err != nil {
		return err
	}
	if found && current == consented {
		return nil
	}

	var language interface{}
	if languageShown != "" {
		language = languageShown
	}

	_, err = r.db.Exec(`
		INSERT INTO consents (consent_id, resident_pseudonym_id, consented, language_shown, operator_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		uuid.New().String(),
		residentPseudonymID,
		consented,
		language,
		operatorID,
		time.Now().UTC(),
	)
	return err
}
