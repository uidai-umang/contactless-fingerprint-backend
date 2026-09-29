package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/repository"
)

type DevHandler struct {
	operatorRepo *repository.OperatorRepository
}

func NewDevHandler(operatorRepo *repository.OperatorRepository) *DevHandler {
	return &DevHandler{operatorRepo: operatorRepo}
}

const testOperatorID = "00000000-0000-0000-0000-000000000001"

// SeedTestOperator creates the reserved dummy test operator used for
// local/Postman testing. Idempotent -- call it any time after clearing data.
//
//	201 -- created
//	200 -- already existed, nothing changed
//	500 -- unexpected DB error
func (h *DevHandler) SeedTestOperator(ctx *gin.Context) {
	created, err := h.operatorRepo.SeedTestOperator(
		testOperatorID,
		"Test Operator",
		"test.operator@uidai.gov.in",
		"9999999999",
	)
	if err != nil {
		log.Printf("SeedTestOperator error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	status := http.StatusOK
	message := "Test operator already exists"
	if created {
		status = http.StatusCreated
		message = "Test operator created"
	}
	ctx.JSON(status, gin.H{"message": message, "operator_id": testOperatorID})
}
