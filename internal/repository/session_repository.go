package repository

import (
	"database/sql"

	"github.com/lib/pq"

	"contactless-fingerprint-backend/internal/model"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create inserts a new session record and returns it with generated session_id.
// Returns *ErrForeignKeyViolation if any referenced ID (operator, device, centre, resident) does not exist.
func (r *SessionRepository) Create(req model.CreateSessionRequest) (*model.Session, error) {
	session := &model.Session{}

	query := `
		INSERT INTO sessions (operator_id, device_id, centre_id, resident_pseudonym_id, status)
		VALUES ($1, $2, $3, $4, 'ACTIVE')
		RETURNING session_id, operator_id, device_id, centre_id, resident_pseudonym_id, status, started_at
	`
	err := r.db.QueryRow(query,
		req.OperatorID,
		req.DeviceID,
		req.CentreID,
		req.ResidentPseudonymID,
	).Scan(
		&session.SessionID,
		&session.OperatorID,
		&session.DeviceID,
		&session.CentreID,
		&session.ResidentPseudonymID,
		&session.Status,
		&session.StartedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return nil, &ErrForeignKeyViolation{Field: parseFKField(pqErr.Constraint)}
		}
		return nil, err
	}

	return session, nil
}

// Close marks a session as COMPLETED. Returns ErrNotFound if the session does not exist,
// ErrSessionAlreadyClosed if it is already in COMPLETED or CLOSED status.
func (r *SessionRepository) Close(sessionID string, closeReason string) error {
	session, err := r.GetByID(sessionID)
	if err != nil {
		return err
	}
	if session.Status == "COMPLETED" || session.Status == "CLOSED" {
		return ErrSessionAlreadyClosed
	}

	_, err = r.db.Exec(`
		UPDATE sessions
		SET status = 'COMPLETED', closed_at = NOW(), close_reason = $1
		WHERE session_id = $2
	`, closeReason, sessionID)
	return err
}

// GetByID fetches a session by its ID. Returns ErrNotFound if no row exists.
func (r *SessionRepository) GetByID(sessionID string) (*model.Session, error) {
	session := &model.Session{}

	err := r.db.QueryRow(`
		SELECT session_id, operator_id, device_id, centre_id,
		       resident_pseudonym_id, status, started_at
		FROM sessions
		WHERE session_id = $1
	`, sessionID).Scan(
		&session.SessionID,
		&session.OperatorID,
		&session.DeviceID,
		&session.CentreID,
		&session.ResidentPseudonymID,
		&session.Status,
		&session.StartedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return session, nil
}
