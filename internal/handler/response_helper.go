package handler

import (
	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/model"
)

func respondError(ctx *gin.Context, statusCode int, message string) {
	ctx.JSON(statusCode, model.NewErrorResponse(message))
}

func respondErrorWithData(ctx *gin.Context, statusCode int, message string, data interface{}) {
	ctx.JSON(statusCode, model.NewErrorResponseWithData(message, data))
}
