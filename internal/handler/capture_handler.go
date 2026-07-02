package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/service"
)

type CaptureHandler struct {
	captureService *service.CaptureService
}

func NewCaptureHandler(captureService *service.CaptureService) *CaptureHandler {
	return &CaptureHandler{captureService: captureService}
}

// Upload handles a single fingerprint capture upload
func (h *CaptureHandler) Upload(ctx *gin.Context) {
	var req model.CaptureRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.captureService.Upload(req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

// BatchUpload handles multiple pending captures in one request.
// Android calls this when resuming a session with pending uploads.
func (h *CaptureHandler) BatchUpload(ctx *gin.Context) {
	var req model.BatchCaptureRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	responses, err := h.captureService.BatchUpload(req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, responses)
}
