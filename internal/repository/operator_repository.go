package repository

import (
	"contactless-fingerprint-backend/internal/model"
	"database/sql"

	"github.com/google/uuid"
)

type OperatorRepository struct {
	db *sql.DB
}

func NewOperatorRepository(db *sql.DB) *OperatorRepository {
	return &OperatorRepository{db: db}
}

// operator_ref_id is NULL for the dev test operator, hence the COALESCE.
const operatorColumns = `operator_id, COALESCE(operator_ref_id, ''), status, is_fraud, date_created, date_modified`

func scanOperator(row *sql.Row) (*model.Operator, error) {
	op := &model.Operator{}
	err := row.Scan(&op.OperatorID, &op.OperatorRefID, &op.Status, &op.IsFraud, &op.DateCreated, &op.DateModified)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return op, nil
}

// SeedTestOperator inserts the reserved test operator if it doesn't already
// exist. INSERT IGNORE makes this idempotent -- safe to call again after a
// DB reset; does nothing (no error) if the row is already there.
func (r *OperatorRepository) AddTestOperator(operatorID string) (created bool, err error) {
	res, err := r.db.Exec(
		`INSERT IGNORE INTO operators (operator_id, status) VALUES (?, 'ACTIVE')`,
		operatorID,
	)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// FindByRefID returns the operator with this operator_ref_id, or ErrNotFound.
func (r *OperatorRepository) FindByRefID(operatorRefID string) (*model.Operator, error) {
	return scanOperator(r.db.QueryRow(
		`SELECT `+operatorColumns+` FROM operators WHERE operator_ref_id = ?`,
		operatorRefID,
	))
}

// FindByID returns the operator with this operator_id, or ErrNotFound.
func (r *OperatorRepository) FindByID(operatorID string) (*model.Operator, error) {
	return scanOperator(r.db.QueryRow(
		`SELECT `+operatorColumns+` FROM operators WHERE operator_id = ?`,
		operatorID,
	))
}

// Create inserts a new ACTIVE, non-fraud operator for this operator_ref_id.
// date_created / date_modified are filled by the DB defaults.
func (r *OperatorRepository) Create(operatorRefID string) (*model.Operator, error) {
	_, err := r.db.Exec(`
		INSERT INTO operators (operator_id, operator_ref_id, status)
		VALUES (?, ?, 'ACTIVE')
	`, uuid.New().String(), operatorRefID)
	if err != nil {
		return nil, err
	}

	return r.FindByRefID(operatorRefID)
}

// ExistsByID reports whether an operator row with this id exists.
func (r *OperatorRepository) ExistsByID(operatorID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM operators WHERE operator_id = ?)`,
		operatorID,
	).Scan(&exists)
	return exists, err
}

// IsAllowed reports whether operator_ref_id is in the allowed_operators list.
func (r *OperatorRepository) IsAllowed(operatorRefID string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM allowed_operators WHERE operator_ref_id = ?)`,
		operatorRefID,
	).Scan(&allowed)
	return allowed, err
}

// AddAllowed adds operator_ref_id to the allowed list. Idempotent: created is
// false if it was already there.
func (r *OperatorRepository) AddAllowed(operatorRefID string) (created bool, err error) {
	res, err := r.db.Exec(
		`INSERT IGNORE INTO allowed_operators (operator_ref_id) VALUES (?)`,
		operatorRefID,
	)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// SetFraud sets the fraud flag on the operator with this operator_ref_id.
// date_modified is bumped by the column's ON UPDATE clause when the value
// actually changes. Returns ErrNotFound if no such operator exists.
func (r *OperatorRepository) SetFraud(operatorRefID string, isFraud bool) error {
	if _, err := r.FindByRefID(operatorRefID); err != nil {
		return err
	}
	_, err := r.db.Exec(
		`UPDATE operators SET is_fraud = ? WHERE operator_ref_id = ?`,
		isFraud, operatorRefID,
	)
	return err
}
