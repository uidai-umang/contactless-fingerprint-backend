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

type SessionHandler struct {
	sessionService *service.SessionService
}

func NewSessionHandler(sessionService *service.SessionService) *SessionHandler {
	return &SessionHandler{sessionService: sessionService}
}

// Create starts a new capture session for a resident.
//
//	400 — missing/malformed required fields
//	404 — operator_id, device_id, centre_id, or resident_pseudonym_id does not exist
//	500 — unexpected DB error
func (h *SessionHandler) Create(ctx *gin.Context) {
	var req model.CreateSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	session, err := h.sessionService.Create(req)
	if err != nil {
		var fkErr *repository.ErrForeignKeyViolation
		if errors.As(err, &fkErr) {
			respondError(ctx, http.StatusNotFound, "Referenced "+fkErr.Field+" does not exist")
			return
		}
		log.Printf("Session create error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusCreated, session)
}

// Close marks a session as completed.
//
//	400 — session_id missing or malformed
//	404 — session_id does not exist
//	409 — session is already closed/completed
//	500 — unexpected DB error
func (h *SessionHandler) Close(ctx *gin.Context) {
	var req model.CloseSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.sessionService.Close(req.SessionID, req.CloseReason); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			respondError(ctx, http.StatusNotFound, "Session not found")
			return
		}
		if errors.Is(err, repository.ErrSessionAlreadyClosed) {
			respondError(ctx, http.StatusConflict, "Session is already closed")
			return
		}
		log.Printf("Session close error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Session closed successfully"})
}
