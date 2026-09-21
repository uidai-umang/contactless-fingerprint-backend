package repository

import (
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"contactless-fingerprint-backend/internal/model"
)

type QuotaRepository struct {
	db *sql.DB
}

func NewQuotaRepository(db *sql.DB) *QuotaRepository {
	return &QuotaRepository{db: db}
}

func (r *QuotaRepository) GetTargets() ([]model.QuotaTarget, error) {
	rows, err := r.db.Query("SELECT dimension, `key`, target_count FROM quota_targets")
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
		SELECT dimension, `+"`key`"+`, target_count
		FROM quota_targets
		WHERE dimension = ? AND `+"`key`"+` = ?
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
	// MySQL has no FILTER (WHERE ...) clause -- CASE WHEN inside COUNT is the equivalent
	err = r.db.QueryRow(`
		SELECT COUNT(*), COUNT(CASE WHEN operator_id = ? THEN 1 END)
		FROM quota_overrides
		WHERE dimension = ? AND `+"`key`"+` = ?
	`, operatorID, dimension, key).Scan(&total, &byOperator)
	return total, byOperator, err
}

func (r *QuotaRepository) InsertOverride(req model.LogOverrideRequest) (*model.QuotaOverride, error) {
	override := &model.QuotaOverride{
		OverrideID:          uuid.New().String(),
		ResidentPseudonymID: req.ResidentPseudonymID,
		OperatorID:          req.OperatorID,
		Dimension:           req.Dimension,
		Key:                 req.Key,
		CreatedAt:           time.Now().UTC(),
	}

	query := "INSERT INTO quota_overrides (override_id, resident_pseudonym_id, operator_id, dimension, `key`, created_at) VALUES (?, ?, ?, ?, ?, ?)"

	_, err := r.db.Exec(query,
		override.OverrideID,
		override.ResidentPseudonymID,
		override.OperatorID,
		override.Dimension,
		override.Key,
		override.CreatedAt,
	)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1452 {
			return nil, &ErrForeignKeyViolation{Field: parseFKField(mysqlErr.Message)}
		}
		return nil, err
	}

	return override, nil
}
