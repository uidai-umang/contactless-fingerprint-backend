package repository

import (
	"errors"
	"strings"
)

var ErrNotFound = errors.New("record not found")
var ErrDuplicateCapture = errors.New("this finger has already been captured for this session")
var ErrSessionAlreadyClosed = errors.New("session is already closed")

// ErrForeignKeyViolation is returned when an INSERT references a non-existent foreign key.
type ErrForeignKeyViolation struct {
	Field string
}

func (e *ErrForeignKeyViolation) Error() string {
	return "referenced " + e.Field + " does not exist"
}

// parseFKField maps a PostgreSQL constraint name to a human-readable field name.
func parseFKField(constraint string) string {
	switch {
	case strings.Contains(constraint, "operator_id"):
		return "operator_id"
	case strings.Contains(constraint, "device_id"):
		return "device_id"
	case strings.Contains(constraint, "centre_id"):
		return "centre_id"
	case strings.Contains(constraint, "resident_pseudonym_id"):
		return "resident_pseudonym_id"
	case strings.Contains(constraint, "session_id"):
		return "session_id"
	default:
		return "related record"
	}
}
