package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/auth"
	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
)

type AuthHandler struct {
	operatorRepo     *repository.OperatorRepository
	refreshTokenRepo *repository.RefreshTokenRepository
}

func NewAuthHandler(operatorRepo *repository.OperatorRepository, refreshTokenRepo *repository.RefreshTokenRepository) *AuthHandler {
	return &AuthHandler{operatorRepo: operatorRepo, refreshTokenRepo: refreshTokenRepo}
}

// issueTokenPair is shared by IssueToken and Refresh -- both end with
// "mint a fresh access+refresh pair for this operator_id and return it."
func (h *AuthHandler) issueTokenPair(ctx *gin.Context, operatorID string) {
	accessToken, err := auth.GenerateAccessToken(operatorID)
	if err != nil {
		log.Printf("GenerateAccessToken error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		log.Printf("GenerateRefreshToken error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	expiresAt := time.Now().UTC().Add(auth.RefreshTokenTTL)
	if err := h.refreshTokenRepo.Store(operatorID, auth.HashToken(refreshToken), expiresAt); err != nil {
		log.Printf("Store refresh token error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, model.AuthTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(auth.AccessTokenTTL.Seconds()),
	})
}

// IssueToken is the "login" endpoint. Trust anchor is operator_ref_id --
// same identity resolution the /operators/lookup endpoint already does.
func (h *AuthHandler) IssueToken(ctx *gin.Context) {
	var req model.AuthTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	op, err := h.operatorRepo.FindOrCreateByRefID(req.OperatorRefID)
	if err != nil {
		log.Printf("IssueToken lookup error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	h.issueTokenPair(ctx, op.OperatorID)
}

// Refresh exchanges a still-valid refresh token for a brand new access +
// refresh pair, and immediately revokes the old refresh token (rotation).
// If the old one is presented again after this, FindValid will reject it --
// that's the signal something's wrong (token reuse/theft), not just an
// inconvenience.
func (h *AuthHandler) Refresh(ctx *gin.Context) {
	var req model.RefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	rt, err := h.refreshTokenRepo.FindValid(auth.HashToken(req.RefreshToken))
	if err == repository.ErrNotFound {
		respondError(ctx, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}
	if err != nil {
		log.Printf("Refresh lookup error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	exists, err := h.operatorRepo.ExistsByID(rt.OperatorID)
	if err != nil {
		log.Printf("Refresh operator check error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}
	if !exists {
		// Operator no longer exists (e.g. test-data reset) -- the token is dead.
		_ = h.refreshTokenRepo.Revoke(rt.TokenID)
		respondError(ctx, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	if err := h.refreshTokenRepo.Revoke(rt.TokenID); err != nil {
		log.Printf("Revoke old refresh token error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	h.issueTokenPair(ctx, rt.OperatorID)
}

// Logout revokes a refresh token. Always returns 200 whether or not the
// token was actually found/valid -- an attacker probing which refresh
// tokens exist shouldn't be able to tell the difference between "revoked"
// and "was never valid" from the response.
func (h *AuthHandler) Logout(ctx *gin.Context) {
	var req model.LogoutRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		respondError(ctx, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	rt, err := h.refreshTokenRepo.FindValid(auth.HashToken(req.RefreshToken))
	if err == nil {
		if err := h.refreshTokenRepo.Revoke(rt.TokenID); err != nil {
			log.Printf("Logout revoke error: %v", err)
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}
