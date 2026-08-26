package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
	"contactless-fingerprint-backend/internal/service"
)

type QuotaHandler struct {
	quotaService *service.QuotaService
}

func NewQuotaHandler(quotaService *service.QuotaService) *QuotaHandler {
	return &QuotaHandler{quotaService: quotaService}
}

var validDimensions = map[string]bool{
	"GENDER":    true,
	"AGE_GROUP": true,
}

var validAgeGroups = map[string]bool{
	"5-17":  true,
	"18-40": true,
	"41-60": true,
	"60+":   true,
}

// Check reports quota status for a proposed gender+age_group pair, plus
// alternative brackets that still have open slots.
//
//	400 — gender or age_group missing/invalid
//	500 — unexpected DB error
func (h *QuotaHandler) Check(ctx *gin.Context) {
	gender := strings.ToUpper(strings.TrimSpace(ctx.Query("gender")))
	if !validGenders[gender] {
		respondErrorWithData(ctx, http.StatusBadRequest,
			"Invalid gender value",
			gin.H{"allowed_values": []string{"MALE", "FEMALE", "OTHER"}},
		)
		return
	}

	ageGroup := strings.ToUpper(strings.TrimSpace(ctx.Query("age_group")))
	if !validAgeGroups[ageGroup] {
		respondErrorWithData(ctx, http.StatusBadRequest,
			"Invalid age_group value",
			gin.H{"allowed_values": []string{"5-17", "18-40", "41-60", "60+"}},
		)
		return
	}

	result, err := h.quotaService.CheckQuota(gender, ageGroup)
	if err != nil {
		log.Printf("Quota check error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// LogOverride records that an operator proceeded with a capture despite the
// resident's demographic bracket being full/near-full.
//
//	400 — missing/malformed required fields, or invalid dimension
//	404 — session_id, resident_pseudonym_id, or operator_id does not exist
//	500 — unexpected DB error
func (h *QuotaHandler) LogOverride(ctx *gin.Context) {
	var req model.LogOverrideRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	req.Dimension = strings.ToUpper(strings.TrimSpace(req.Dimension))
	if !validDimensions[req.Dimension] {
		respondErrorWithData(ctx, http.StatusBadRequest,
			"Invalid dimension value",
			gin.H{"allowed_values": []string{"GENDER", "AGE_GROUP"}},
		)
		return
	}

	override, err := h.quotaService.LogOverride(req)
	if err != nil {
		var fkErr *repository.ErrForeignKeyViolation
		if errors.As(err, &fkErr) {
			respondError(ctx, http.StatusNotFound, "Referenced "+fkErr.Field+" does not exist")
			return
		}
		log.Printf("Quota override error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusCreated, override)
}
