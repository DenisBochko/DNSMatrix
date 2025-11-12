package route

import (
	"github.com/gin-gonic/gin"
)

type UserHandler interface {
	GetUser(*gin.Context)
	GetUserJWT(c *gin.Context)
	DeleteUser(c *gin.Context)
	BlockUser(c *gin.Context)

	ForgotPassword(c *gin.Context)
	ResetPassword(c *gin.Context)
	DeleteSelf(c *gin.Context)
}

func RegisterAdminUserRoutes(g *gin.RouterGroup, h UserHandler, jwtAuthMiddleware, allowManagerAndAdminMiddleware gin.HandlerFunc) {
	g.GET("/:user_id", h.GetUser)
	g.POST("/password-forgot", h.ForgotPassword)

	protected := g.Group("", jwtAuthMiddleware)
	protected.GET("", h.GetUserJWT)

	//Восстановление и сброс пароля
	protected.POST("/password-reset", h.ResetPassword)
	protected.DELETE("", h.DeleteSelf)

	adminOrManagerRequired := protected.Group("", allowManagerAndAdminMiddleware)
	adminOrManagerRequired.DELETE(":user_id", h.DeleteUser)
	adminOrManagerRequired.POST("/block/:user_id", h.BlockUser)
}
