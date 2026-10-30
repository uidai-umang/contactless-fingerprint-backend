package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// RequireAdminKey guards admin routes behind the ADMIN_API_KEY env var,
// sent as the X-Admin-Key header. If ADMIN_API_KEY is not set, admin routes
// stay disabled rather than open.
func RequireAdminKey() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		expected := os.Getenv("ADMIN_API_KEY")
		if expected == "" {
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"message": "Admin API is not configured"})
			return
		}

		provided := ctx.GetHeader("X-Admin-Key")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid admin key"})
			return
		}

		ctx.Next()
	}
}
