package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/middleware"
	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
	"contactless-fingerprint-backend/internal/service"
)

type ConsentHandler struct {
	consentService *service.ConsentService
}

func NewConsentHandler(consentService *service.ConsentService) *ConsentHandler {
	return &ConsentHandler{consentService: consentService}
}

// Record saves the consent the resident gave (or declined) on the consent screen.
// Idempotent: repeating the same decision does not add a new record.
//
//	400 -- malformed body, or resident_pseudonym_id / consented missing
//	404 -- resident does not exist
//	500 -- unexpected error
func (h *ConsentHandler) Record(ctx *gin.Context) {
	var req model.ConsentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	response, err := h.consentService.Record(req, middleware.OperatorID(ctx))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			respondError(ctx, http.StatusNotFound, "Resident not found")
			return
		}
		log.Printf("Consent record error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, response)
}
