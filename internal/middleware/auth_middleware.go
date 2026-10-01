package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/auth"
)

// RequireAuth guards a route group behind a valid access token. On success
// it stores the verified operator_id in the request context under the key
// "operator_id" -- downstream handlers read it from there instead of
// trusting whatever operator_id the client put in the request body/query,
// which is what makes this actually enforce anything (a bad actor can put
// any operator_id they want in a JSON body; they can't forge a valid
// signature without the secret).
func RequireAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		if header == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Missing Authorization header"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Authorization header must be: Bearer <token>"})
			return
		}

		claims, err := auth.ValidateAccessToken(parts[1])
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid or expired access token"})
			return
		}

		// Makes operator_id available to every handler further down the
		// chain via ctx.MustGet("operator_id") / ctx.Get("operator_id").
		ctx.Set("operator_id", claims.OperatorID)

		// Without this, the request would just stop here -- Next() is what
		// hands control to the next middleware/handler in the chain.
		// AbortWithStatusJSON above implicitly skips this by writing the
		// response and returning early, so it never reaches Next().
		ctx.Next()
	}

}

func OperatorID(ctx *gin.Context) string {
	return ctx.MustGet("operator_id").(string)
}
