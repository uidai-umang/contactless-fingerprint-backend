package repository

import (
	"errors"
	"strings"
)

var ErrNotFound = errors.New("record not found")
var ErrDuplicateCapture = errors.New("this finger has already been captured for this resident")
var ErrCaptureModeMismatch = errors.New("resident is already enrolled in a different capture mode")

type ErrForeignKeyViolation struct {
	Field string
}

func (e *ErrForeignKeyViolation) Error() string {
	return "referenced " + e.Field + " does not exist"
}

func parseFKField(message string) string {
	switch {
	case strings.Contains(message, "operator_id"):
		return "operator_id"
	case strings.Contains(message, "device_id"):
		return "device_id"
	case strings.Contains(message, "resident_pseudonym_id"):
		return "resident_pseudonym_id"
	default:
		return "related record"
	}
}
