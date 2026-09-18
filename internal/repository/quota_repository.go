package repository

import (
	"database/sql"

	"github.com/lib/pq"

	"contactless-fingerprint-backend/internal/model"
)

type QuotaRepository struct {
	db *sql.DB
}

func NewQuotaRepository(db *sql.DB) *QuotaRepository {
	return &QuotaRepository{db: db}
}

func (r *QuotaRepository) GetTargets() ([]model.QuotaTarget, error) {
	rows, err := r.db.Query(`SELECT dimension, key, target_count FROM quota_targets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []model.QuotaTarget
	for rows.Next() {
		t := model.QuotaTarget{}
		if err := rows.Scan(&t.Dimension, &t.Key, &t.TargetCount); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}

	return targets, nil
}

func (r *QuotaRepository) GetTarget(dimension, key string) (*model.QuotaTarget, error) {
	t := &model.QuotaTarget{}

	err := r.db.QueryRow(`
		SELECT dimension, key, target_count
		FROM quota_targets
		WHERE dimension = $1 AND key = $2
	`, dimension, key).Scan(&t.Dimension, &t.Key, &t.TargetCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (r *QuotaRepository) GetNationalCapturedByGender() (map[string]int, error) {
	rows, err := r.db.Query(`
		SELECT r.gender, COUNT(DISTINCT c.resident_pseudonym_id)
		FROM captures c
		JOIN residents r ON r.resident_pseudonym_id = c.resident_pseudonym_id
		WHERE c.upload_status = 'UPLOADED'
		GROUP BY r.gender
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		counts[key] = count
	}

	return counts, nil
}

func (r *QuotaRepository) GetNationalCapturedByAgeGroup() (map[string]int, error) {
	rows, err := r.db.Query(`
		SELECT r.age_group, COUNT(DISTINCT c.resident_pseudonym_id)
		FROM captures c
		JOIN residents r ON r.resident_pseudonym_id = c.resident_pseudonym_id
		WHERE c.upload_status = 'UPLOADED'
		GROUP BY r.age_group
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		counts[key] = count
	}

	return counts, nil
}

func (r *QuotaRepository) CountOverrides(dimension, key, operatorID string) (total int, byOperator int, err error) {
	err = r.db.QueryRow(`
		SELECT COUNT(*), COUNT(*) FILTER (WHERE operator_id = $3)
		FROM quota_overrides
		WHERE dimension = $1 AND key = $2
	`, dimension, key, operatorID).Scan(&total, &byOperator)
	return total, byOperator, err
}

func (r *QuotaRepository) InsertOverride(req model.LogOverrideRequest) (*model.QuotaOverride, error) {
	override := &model.QuotaOverride{}

	query := `
		INSERT INTO quota_overrides (resident_pseudonym_id, operator_id, dimension, key)
		VALUES ($1, $2, $3, $4)
		RETURNING override_id, resident_pseudonym_id, operator_id, dimension, key, created_at
	`

	err := r.db.QueryRow(query,
		req.ResidentPseudonymID,
		req.OperatorID,
		req.Dimension,
		req.Key,
	).Scan(
		&override.OverrideID,
		&override.ResidentPseudonymID,
		&override.OperatorID,
		&override.Dimension,
		&override.Key,
		&override.CreatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return nil, &ErrForeignKeyViolation{Field: parseFKField(pqErr.Constraint)}
		}
		return nil, err
	}

	return override, nil
}
