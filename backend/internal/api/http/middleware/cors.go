package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"hackathon-back/internal/config"
)

func CORS(cfg config.CORS) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	corsConfig := cors.Config{
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		ExposeHeaders:    cfg.ExposeHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
		AllowWebSockets:  cfg.AllowWebSockets,
		AllowFiles:       cfg.AllowFiles,
	}

	if cfg.AllowAllOrigins {
		corsConfig.AllowOriginFunc = func(origin string) bool {
			return true
		}
	} else {
		corsConfig.AllowOrigins = cfg.AllowOrigins
	}

	return cors.New(corsConfig)
}
