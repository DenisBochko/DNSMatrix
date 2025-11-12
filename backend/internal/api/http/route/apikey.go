package route

import (
	"github.com/gin-gonic/gin"
)

type APIKeyHandler interface {
	Create(c *gin.Context)
	List(c *gin.Context)
	Revoke(c *gin.Context)
}
