package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
)

type OperatorHandler struct {
	operatorRepo *repository.OperatorRepository
}

func NewOperatorHandler(operatorRepo *repository.OperatorRepository) *OperatorHandler {
	return &OperatorHandler{operatorRepo: operatorRepo}
}

// LookupOrCreate handles operator lookup by operator_ref_id (supplied by
// Operator Mitra). Creates a new operator record the first time a given
// ref_id is seen.
func (h *OperatorHandler) LookupOrCreate(ctx *gin.Context) {
	var req model.OperatorLookupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	op, err := h.operatorRepo.FindOrCreateByRefID(req.OperatorRefID)
	if err != nil {
		log.Printf("LookupOrCreate service error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, model.OperatorLookupResponse{
		OperatorID:    op.OperatorID,
		OperatorRefID: op.OperatorRefID,
		Status:        op.Status,
	})
}
