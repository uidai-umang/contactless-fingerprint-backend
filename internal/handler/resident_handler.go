package handler

import (
	"net/http"
	"os"

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

// LookupOrCreate handles resident lookup by aadhaar_hash.
// Creates a new resident record if not found.
// Returns resident ID and capture progress.
func (h *ResidentHandler) LookupOrCreate(ctx *gin.Context) {
	var req model.ResidentLookupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.residentService.FindOrCreateResident(req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Reset wipes all data for a resident.
// Only allowed for the reserved test Aadhaar hash to prevent misuse in production.
func (h *ResidentHandler) Reset(ctx *gin.Context) {
	var req model.DevResetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Only allow reset for the reserved test Aadhaar hash
	// Android app must hash "555522222222" before sending
	reservedTestHash := os.Getenv("TEST_AADHAAR_HASH")
	if req.AadhaarHash != reservedTestHash {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "Reset only allowed for reserved test resident",
		})
		return
	}

	if err := h.residentService.Reset(req.AadhaarHash); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Test resident data reset successfully"})
}
