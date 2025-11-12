package route

import (
	"github.com/gin-gonic/gin"
)

type HealthHandler interface {
	Ping(c *gin.Context)
	ProtectedPing(c *gin.Context)
	Health(c *gin.Context)
}

func RegisterHealth(g *gin.RouterGroup, h HealthHandler, jwtAuthMiddleware gin.HandlerFunc) {
	g.GET("", h.Health)
	g.GET("/ping", h.Ping)

	protected := g.Group("/protected", jwtAuthMiddleware)
	protected.GET("/ping", h.ProtectedPing)
}
