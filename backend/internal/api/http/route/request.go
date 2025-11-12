package route

import (
	"github.com/gin-gonic/gin"
)

type RequestHandler interface {
	CreateRequest(c *gin.Context)
	GetResults(c *gin.Context)
	StreamResults(c *gin.Context)
}

func RegisterRequestRoutes(g *gin.RouterGroup, handler RequestHandler) {
	g.POST("/task", handler.CreateRequest)
	g.GET("/:request_id", handler.GetResults)
	g.GET("/ws/check/:request_id", handler.StreamResults)
}
