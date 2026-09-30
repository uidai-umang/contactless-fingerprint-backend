package repository

import (
	"contactless-fingerprint-backend/internal/model"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type OperatorRepository struct {
	db *sql.DB
}

func NewOperatorRepository(db *sql.DB) *OperatorRepository {
	return &OperatorRepository{db: db}
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

func (r *OperatorRepository) FindOrCreateByRefID(operatorRefId string) (*model.Operator, error) {
	op := &model.Operator{}

	err := r.db.QueryRow(`
		SELECT operator_id, operator_ref_id, status, created_at
		FROM operators
		WHERE operator_ref_id = ?
	`, operatorRefId).Scan(&op.OperatorID, &op.OperatorRefID, &op.Status, &op.CreatedAt)

	if err == nil {
		return op, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	op.OperatorID = uuid.New().String()
	op.OperatorRefID = operatorRefId
	op.Status = "ACTIVE"
	op.CreatedAt = time.Now().UTC()

	_, err = r.db.Exec(`
		INSERT INTO operators (operator_id, operator_ref_id, status, created_at)
		VALUES (?,?, 'ACTIVE', ?)
	`, op.OperatorID, op.OperatorRefID, op.Status, op.CreatedAt)

	if err != nil {
		return nil, err
	}

	return op, nil
}
