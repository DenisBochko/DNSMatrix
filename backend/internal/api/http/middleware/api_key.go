package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"hackathon-back/internal/model"
)

// APIKeyRepositoryInterface - интерфейс для middleware
type APIKeyRepositoryInterface interface {
	GetAllActive(ctx context.Context) ([]model.APIKey, error)
}

// APIKeyAuthMiddleware создает middleware с интерфейсом
func APIKeyAuthMiddleware(apiKeyRepo APIKeyRepositoryInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		// Получаем все активные ключи через интерфейс
		activeKeys, err := apiKeyRepo.GetAllActive(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			c.Abort()
			return
		}

		// Ищем совпадение
		var authenticatedKey *model.APIKey
		for _, key := range activeKeys {
			if err := bcrypt.CompareHashAndPassword(key.KeyHash, []byte(apiKey)); err == nil {
				authenticatedKey = &key
				break
			}
		}

		if authenticatedKey == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			c.Abort()
			return
		}

		// Устанавливаем user_id в контекст для следующих handlers
		c.Set("user_id", authenticatedKey.UserID)
		c.Set("api_key_id", authenticatedKey.ID)

		c.Next()
	}
}
