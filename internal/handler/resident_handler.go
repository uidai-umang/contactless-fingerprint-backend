package handler

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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

// LookupOrCreate handles resident lookup by aadhaar_hash.
// Creates a new resident record if not found.
// Returns resident ID and capture progress.
func (h *ResidentHandler) LookupOrCreate(ctx *gin.Context) {
	var req model.ResidentLookupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ageGroup, err := normalizeAgeGroup(req.AgeGroup)
	if err != nil {
		respondError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	req.AgeGroup = ageGroup

	req.Gender = strings.ToUpper(strings.TrimSpace(req.Gender))
	if !validGenders[req.Gender] {
		respondErrorWithData(ctx, http.StatusBadRequest,
			"Invalid gender value",
			gin.H{"allowed_values": []string{"MALE", "FEMALE", "OTHER"}},
		)
		return
	}

	response, err := h.residentService.FindOrCreateResident(req)
	if err != nil {
		log.Printf("LookupOrCreate service error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Reset wipes all data for a resident.
// Only allowed for the reserved test Aadhaar hash to prevent misuse in production.
func (h *ResidentHandler) Reset(ctx *gin.Context) {
	var req model.DevResetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	reservedTestHash := os.Getenv("TEST_AADHAAR_HASH")
	if req.AadhaarHash != reservedTestHash {
		respondError(ctx, http.StatusForbidden, "Reset only allowed for reserved test resident")
		return
	}

	if err := h.residentService.Reset(req.AadhaarHash); err != nil {
		log.Printf("Reset service error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Test resident data reset successfully"})
}

// normalizeAgeGroup converts a raw age number (e.g. "25") into the
// bracket format required by the DB CHECK constraint (e.g. "18-40").
func normalizeAgeGroup(rawAge string) (string, error) {
	age, err := strconv.Atoi(strings.TrimSpace(rawAge))
	if err != nil {
		return "", errInvalidAge
	}
	if age < 0 || age > 130 {
		return "", errInvalidAge
	}

	switch {
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

var errInvalidAge = &validationError{"age must be a valid number between 0 and 130"}

type validationError struct {
	msg string
}

func (e *validationError) Error() string {
	return e.msg
}
