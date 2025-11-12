package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		latency := time.Since(startTime)
		statusCode := c.Writer.Status()

		log.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("uri", c.Request.URL.RequestURI()),
			zap.Int("code", statusCode),
			zap.String("status", http.StatusText(statusCode)),
			zap.String("latency", fmt.Sprintf("%d µs", latency)),
		)
	}
}
