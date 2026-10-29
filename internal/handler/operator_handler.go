package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/service"
)

const operatorNotAllowedMessage = "You are not allowed to proceed for data collection"

type OperatorHandler struct {
	operatorService *service.OperatorService
}

func NewOperatorHandler(operatorService *service.OperatorService) *OperatorHandler {
	return &OperatorHandler{operatorService: operatorService}
}

// LookupOrCreate handles operator lookup by operator_ref_id (supplied by
// Operator Mitra). An operator already in operators is returned as is; a new
// one is created only if it is in the allowed list.
//
//	200 -- operator found / created
//	400 -- malformed body
//	403 -- not on the allowed list, or marked fraud
//	500 -- unexpected error
func (h *OperatorHandler) LookupOrCreate(ctx *gin.Context) {
	var req model.OperatorLookupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	op, err := h.operatorService.Authorize(req.OperatorRefID)
	if err != nil {
		if errors.Is(err, service.ErrOperatorNotAllowed) {
			respondError(ctx, http.StatusForbidden, operatorNotAllowedMessage)
			return
		}
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
