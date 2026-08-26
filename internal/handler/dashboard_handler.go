package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/service"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
	quotaService     *service.QuotaService
}

func NewDashboardHandler(dashboardService *service.DashboardService, quotaService *service.QuotaService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService, quotaService: quotaService}
}

// Overview returns an operator's overall capture progress.
//
//	400 — operator_id missing
//	500 — unexpected DB error
func (h *DashboardHandler) Overview(ctx *gin.Context) {
	operatorID := ctx.Query("operator_id")
	if operatorID == "" {
		respondError(ctx, http.StatusBadRequest, "operator_id is required")
		return
	}

	overview, err := h.dashboardService.GetOverview(operatorID)
	if err != nil {
		log.Printf("Dashboard overview error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, overview)
}

// Diversity returns quota status lines for gender and age_group brackets.
//
//	400 — operator_id missing
//	500 — unexpected DB error
func (h *DashboardHandler) Diversity(ctx *gin.Context) {
	operatorID := ctx.Query("operator_id")
	if operatorID == "" {
		respondError(ctx, http.StatusBadRequest, "operator_id is required")
		return
	}

	diversity, err := h.quotaService.BuildDiversityLines(operatorID)
	if err != nil {
		log.Printf("Dashboard diversity error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, diversity)
}

// Fingers returns an operator's finger-level capture stats.
//
//	400 — operator_id missing
//	500 — unexpected DB error
func (h *DashboardHandler) Fingers(ctx *gin.Context) {
	operatorID := ctx.Query("operator_id")
	if operatorID == "" {
		respondError(ctx, http.StatusBadRequest, "operator_id is required")
		return
	}

	fingers, err := h.dashboardService.GetFingerStats(operatorID)
	if err != nil {
		log.Printf("Dashboard fingers error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, fingers)
}

// Alerts returns every demographic bracket currently near or at capacity.
//
//	500 — unexpected DB error
func (h *DashboardHandler) Alerts(ctx *gin.Context) {
	alerts, err := h.quotaService.BuildAlerts()
	if err != nil {
		log.Printf("Dashboard alerts error: %v", err)
		respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
		return
	}

	ctx.JSON(http.StatusOK, alerts)
}
