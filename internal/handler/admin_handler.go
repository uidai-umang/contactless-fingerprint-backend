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

type AdminHandler struct {
	operatorService *service.OperatorService
}

func NewAdminHandler(operatorService *service.OperatorService) *AdminHandler {
	return &AdminHandler{operatorService: operatorService}
}

// AddAllowedOperator adds an operator_ref_id to the allowed list.
// Idempotent.
//
//	201 -- added
//	200 -- already on the list, nothing changed
//	400 -- malformed body / missing operator_ref_id
//	500 -- unexpected error
func (h *AdminHandler) AddAllowedOperator(ctx *gin.Context) {
	var req model.AllowedOperatorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	refID := strings.TrimSpace(req.OperatorRefID)
	if refID == "" {
		respondError(ctx, http.StatusBadRequest, "operator_ref_id must not be blank")
		return
	}

	created, err := h.operatorService.AddAllowed(refID)
	if err != nil {
		log.Printf("AddAllowedOperator error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	status := http.StatusOK
	message := "Operator already in allowed list"
	if created {
		status = http.StatusCreated
		message = "Operator added to allowed list"
	}
	ctx.JSON(status, gin.H{"message": message, "operator_ref_id": refID})
}

// SetOperatorFraud marks (is_fraud=true) or clears (is_fraud=false) the fraud
// flag on an existing operator.
//
//	200 -- updated
//	400 -- malformed body / missing fields
//	404 -- operator does not exist
//	500 -- unexpected error
func (h *AdminHandler) SetOperatorFraud(ctx *gin.Context) {
	var req model.OperatorFraudRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.operatorService.SetFraud(req.OperatorRefID, *req.IsFraud); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			respondError(ctx, http.StatusNotFound, "Operator not found")
			return
		}
		log.Printf("SetOperatorFraud error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"operator_ref_id": req.OperatorRefID, "is_fraud": *req.IsFraud})
}
