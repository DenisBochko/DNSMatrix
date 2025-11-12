package route

import (
	"github.com/gin-gonic/gin"
)

type AuthHandler interface {
	Register(c *gin.Context)
	ResendConfirmation(c *gin.Context)
	Confirmation(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
	Refresh(c *gin.Context)
	TestLogin(c *gin.Context)
}

func RegisterAuth(g *gin.RouterGroup, h AuthHandler) {
	g.POST("/register", h.Register)
	g.POST("/resend-confirmation", h.ResendConfirmation)
	g.POST("/confirm", h.Confirmation)
	g.POST("/login", h.Login)
	g.POST("/logout", h.Logout)
	g.POST("/refresh", h.Refresh)
	g.POST("/test-login", h.TestLogin)
}
