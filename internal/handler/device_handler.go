package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
	"contactless-fingerprint-backend/internal/service"
)

type DeviceHandler struct {
	deviceService *service.DeviceService
}

func NewDeviceHandler(deviceService *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceService: deviceService}
}

// Register handles device (+ camera spec) registration.
// Idempotent: calling this again with the same android_id returns the
// existing device record rather than erroring.
//
//	201 — new device registered (or existing one returned)
//	400 — missing required fields
//	404 — referenced operator_id does not exist
//	500 — unexpected error
func (h *DeviceHandler) Register(ctx *gin.Context) {
	var req model.DeviceRegistrationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.AndroidID == "" {
		respondError(ctx, http.StatusBadRequest, "android_id is required")
		return
	}
	if req.OperatorID == "" {
		respondError(ctx, http.StatusBadRequest, "operator_id is required")
		return
	}
	if req.CameraFingerprintHash == "" {
		respondError(ctx, http.StatusBadRequest, "camera_fingerprint_hash is required")
		return
	}

	device, err := h.deviceService.Register(req)
	if err != nil {
		var fkErr *repository.ErrForeignKeyViolation
		if errors.As(err, &fkErr) {
			respondError(ctx, http.StatusNotFound, "Referenced "+fkErr.Field+" does not exist")
			return
		}
		log.Printf("Device registration error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusCreated, device)
}
