package handler

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/service"
)

type ResidentHandler struct {
	residentService *service.ResidentService
}

func NewResidentHandler(residentService *service.ResidentService) *ResidentHandler {
	return &ResidentHandler{residentService: residentService}
}

var validGenders = map[string]bool{
	"MALE":   true,
	"FEMALE": true,
	"OTHER":  true,
}

// LookupOrCreate registers a resident from the details Operator Mitra provides
// (ref id, date of birth, gender) and returns their id and capture progress.
// Idempotent: the same resident_ref_id always returns the same resident, and
// the details stored on first sight win.
//
//	400 -- malformed body, unparseable date_of_birth, or invalid gender
//	422 -- date_of_birth in the future, or resident younger than 5
//	500 -- unexpected error
func (h *ResidentHandler) LookupOrCreate(ctx *gin.Context) {
	var req model.ResidentLookupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	dob, err := parseDateOfBirth(req.DateOfBirth)
	if err != nil {
		respondError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	ageGroup, err := ageGroupForAge(ageInYears(dob, time.Now().UTC()))
	if err != nil {
		respondError(ctx, http.StatusUnprocessableEntity, err.Error())
		return
	}

	gender, ok := normalizeGender(req.Gender)
	if !ok {
		respondErrorWithData(ctx, http.StatusBadRequest,
			"Invalid gender value",
			gin.H{"allowed_values": []string{"MALE", "FEMALE", "OTHER"}},
		)
		return
	}

	response, err := h.residentService.FindOrCreateResident(req.ResidentRefID, dob, gender, ageGroup)
	if err != nil {
		log.Printf("LookupOrCreate service error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Reset wipes a resident.
// Only allowed for the reserved test ref id (TEST_RESIDENT_REF_ID) to prevent
// misuse in production.
func (h *ResidentHandler) Reset(ctx *gin.Context) {
	var req model.DevResetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	reservedRefID := os.Getenv("TEST_RESIDENT_REF_ID")
	if reservedRefID == "" || req.ResidentRefID != reservedRefID {
		respondError(ctx, http.StatusForbidden, "Reset only allowed for reserved test resident")
		return
	}

	if err := h.residentService.Reset(req.ResidentRefID); err != nil {
		log.Printf("Reset service error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Test resident data reset successfully"})
}

// dobLayouts are the date_of_birth formats we accept. The year-only layout is
// there because Aadhaar records can carry just a year of birth.
// Confirm against a real Mitra response and trim/reorder if needed.
var dobLayouts = []string{
	"2006-01-02",
	"02-01-2006",
	"02/01/2006",
	time.RFC3339,
	"2006",
}

// parseDateOfBirth parses raw into a date (midnight UTC). A year-only value
// becomes 1 January of that year.
func parseDateOfBirth(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range dobLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}
	return time.Time{}, errors.New("date_of_birth must be YYYY-MM-DD, DD-MM-YYYY, DD/MM/YYYY or YYYY")
}

// ageInYears returns the completed years between dob and now.
func ageInYears(dob, now time.Time) int {
	age := now.Year() - dob.Year()
	if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
		age--
	}
	return age
}

// ageGroupForAge maps an age to the bracket enforced by the DB CHECK
// constraint. Fixed at enrollment time and stored, so a resident never moves
// between quota buckets as they get older.
func ageGroupForAge(age int) (string, error) {
	switch {
	case age < 0:
		return "", errors.New("date_of_birth cannot be in the future")
	case age < 5:
		return "", errors.New("resident must be at least 5 years old")
	case age > 130:
		return "", errors.New("date_of_birth is not a valid date")
	case age <= 17:
		return "5-17", nil
	case age <= 40:
		return "18-40", nil
	case age <= 60:
		return "41-60", nil
	default:
		return "60+", nil
	}
}

// normalizeGender maps Mitra / Aadhaar gender values (M, F, T or the full
// words, any case) to MALE, FEMALE or OTHER.
func normalizeGender(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "M", "MALE":
		return "MALE", true
	case "F", "FEMALE":
		return "FEMALE", true
	case "T", "O", "TRANSGENDER", "OTHER":
		return "OTHER", true
	default:
		return "", false
	}
}
