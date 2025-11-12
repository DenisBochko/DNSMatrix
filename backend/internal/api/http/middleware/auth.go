package middleware

import (
	"crypto/ecdsa"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"hackathon-back/internal/api/http/handler"
	"hackathon-back/internal/model"
	"hackathon-back/pkg/jwt"
)

func JWTAuth(publicKey *ecdsa.PublicKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		if cookie, err := c.Cookie("access"); err == nil {
			tokenStr = cookie
		}

		if tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, handler.ResponseWithMessage{
				Status:  handler.StatusNotPermitted,
				Message: "Missing access token",
			})

			return
		}

		claims, err := jwt.ValidateToken(tokenStr, publicKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, handler.ResponseWithMessage{
				Status:  handler.StatusNotPermitted,
				Message: "invalid or expired token",
			})
			return
		}

		c.Set(model.UserUIDKey, claims[model.UserUIDKey])
		c.Set(model.UserEmailKey, claims[model.UserEmailKey])
		c.Set(model.UserNameKey, claims[model.UserNameKey])
		c.Set(model.UserConfirmedKey, claims[model.UserConfirmedKey])
		c.Set(model.UserRoleKey, claims[model.UserRoleKey])

		c.Next()
	}
}
