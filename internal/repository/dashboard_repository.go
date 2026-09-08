package repository

import (
	"database/sql"
)

type DashboardRepository struct {
	db *sql.DB
}

func NewDashboardRepository(db *sql.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// GetOperatorTotalCaptured returns the count of distinct residents an
// operator has confirmed captures for
func (r *DashboardRepository) GetOperatorTotalCaptured(operatorID string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(DISTINCT resident_pseudonym_id)
		FROM captures
		WHERE operator_id = $1 AND upload_status = 'UPLOADED'
	`, operatorID).Scan(&count)
	return count, err
}

// GetOperatorCapturedToday returns the count of distinct residents an
// operator has confirmed captures for today
func (r *DashboardRepository) GetOperatorCapturedToday(operatorID string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(DISTINCT resident_pseudonym_id)
		FROM captures
		WHERE operator_id = $1 AND upload_status = 'UPLOADED' AND created_at::date = CURRENT_DATE
	`, operatorID).Scan(&count)
	return count, err
}

// GetOperatorByGender returns the count of distinct residents captured by an
// operator, grouped by gender
func (r *DashboardRepository) GetOperatorByGender(operatorID string) (map[string]int, error) {
	rows, err := r.db.Query(`
		SELECT r.gender, COUNT(DISTINCT c.resident_pseudonym_id)
		FROM captures c
		JOIN residents r ON r.resident_pseudonym_id = c.resident_pseudonym_id
		WHERE c.operator_id = $1 AND c.upload_status = 'UPLOADED'
		GROUP BY r.gender
	`, operatorID)
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

// GetOperatorByAgeGroup returns the count of distinct residents captured by
// an operator, grouped by age_group
func (r *DashboardRepository) GetOperatorByAgeGroup(operatorID string) (map[string]int, error) {
	rows, err := r.db.Query(`
		SELECT r.age_group, COUNT(DISTINCT c.resident_pseudonym_id)
		FROM captures c
		JOIN residents r ON r.resident_pseudonym_id = c.resident_pseudonym_id
		WHERE c.operator_id = $1 AND c.upload_status = 'UPLOADED'
		GROUP BY r.age_group
	`, operatorID)
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

// GetOperatorTotalFingers returns the count of confirmed finger captures for an operator
func (r *DashboardRepository) GetOperatorTotalFingers(operatorID string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM captures
		WHERE operator_id = $1 AND upload_status = 'UPLOADED'
	`, operatorID).Scan(&count)
	return count, err
}

// GetOperatorByFingerType returns the count of confirmed captures for an
// operator, grouped by finger_type
func (r *DashboardRepository) GetOperatorByFingerType(operatorID string) (map[string]int, error) {
	rows, err := r.db.Query(`
		SELECT finger_type, COUNT(*)
		FROM captures
		WHERE operator_id = $1 AND upload_status = 'UPLOADED'
		GROUP BY finger_type
	`, operatorID)
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

// GetOperatorResidentsByFingerCount returns, for an operator, the count of
// residents grouped by how many distinct fingers have been confirmed captured
func (r *DashboardRepository) GetOperatorResidentsByFingerCount(operatorID string) (map[int]int, error) {
	rows, err := r.db.Query(`
		SELECT finger_count, COUNT(*)
		FROM (
			SELECT resident_pseudonym_id, COUNT(DISTINCT finger_type) AS finger_count
			FROM captures
			WHERE operator_id = $1 AND upload_status = 'UPLOADED'
			GROUP BY resident_pseudonym_id
		) sub
		GROUP BY finger_count
	`, operatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[int]int)
	for rows.Next() {
		var fingerCount, residentCount int
		if err := rows.Scan(&fingerCount, &residentCount); err != nil {
			return nil, err
		}
		counts[fingerCount] = residentCount
	}

	return counts, nil
}
