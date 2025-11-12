package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hackathon-back/internal/api/http/handler"
	"hackathon-back/internal/model"
)

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	roleSet := make(map[string]struct{}, len(allowedRoles))

	for _, role := range allowedRoles {
		roleSet[role] = struct{}{}
	}

	return func(c *gin.Context) {
		roleVal, exists := c.Get(model.UserRoleKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, handler.ResponseWithMessage{
				Status:  handler.StatusNotPermitted,
				Message: "no data about the users roles",
			})

			return
		}

		role, ok := roleVal.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, handler.ResponseWithMessage{
				Status:  handler.StatusNotPermitted,
				Message: "invalid role format",
			})
		}

		if _, ok := roleSet[role]; ok {
			c.Next()

			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, handler.ResponseWithMessage{
			Status:  handler.StatusForbidden,
			Message: "insufficient roles privileges",
		})
	}
}
