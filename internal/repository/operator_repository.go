package repository

import "database/sql"

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
