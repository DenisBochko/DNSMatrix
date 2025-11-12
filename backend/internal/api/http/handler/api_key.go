package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"hackathon-back/internal/model"
)

type APIKeyService interface {
	Generate(ctx context.Context, userID uuid.UUID, name string, ttl time.Duration) (string, error)
	GetUserKeys(ctx context.Context, userID uuid.UUID) ([]model.APIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

// APIKeyHandler
// Хендлер для операций с API ключами
type APIKeyHandler struct {
	svc APIKeyService
}

func NewAPIKeyHandler(svc APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{svc: svc}
}

// Create
// @Summary Создание нового API ключа
// @Description Создаёт новый API ключ для текущего пользователя. Ключ можно использовать для аутентификации через заголовок `X-API-Key`.
// @Tags API Keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body model.APIKeyCreateRequest true "Данные для создания API ключа"
// @Success 200 {object} model.APIKeyCreateResponse "Ключ успешно создан"
// @Failure 400 {object} map[string]string "Некорректные данные запроса"
// @Failure 401 {object} map[string]string "Пользователь не авторизован"
// @Failure 500 {object} map[string]string "Ошибка на стороне сервера"
// @Example request { "name": "partner_api", "ttl_hours": 720 }
// @Example success { "api_key": "KJHsT9W-2oP3sA1Q-LzM8fD4eC" }
// @Router /auth/api-key [post]
func (h *APIKeyHandler) Create(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req model.APIKeyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	apiKey, err := h.svc.Generate(c.Request.Context(), userID.(uuid.UUID), req.Name, time.Duration(req.TTLHours)*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}

	c.JSON(http.StatusOK, model.APIKeyCreateResponse{APIKey: apiKey})
}

// List
// @Summary Получение списка всех API ключей пользователя
// @Description Возвращает все активные и неотозванные ключи текущего пользователя.
// @Tags API Keys
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.APIKeyListResponse "Список ключей"
// @Failure 401 {object} map[string]string "Пользователь не авторизован"
// @Failure 500 {object} map[string]string "Ошибка на стороне сервера"
// @Example success {
// @Example success
//
//	  "keys": [
//	    {
//	      "id": "7b2aab2e-4d1f-45b5-90c5-4d5d4db5ef11",
//	      "user_id": "1a2b3c4d-5678-90ab-cdef-1234567890ab",
//	      "name": "mobile_app",
//	      "created_at": "2025-10-25T13:40:00Z",
//	      "expires_at": null,
//	      "revoked": false
//	    }
//	  ]
//	}
//
// @Router /auth/api-key/list [get]
func (h *APIKeyHandler) List(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	keys, err := h.svc.GetUserKeys(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load keys"})
		return
	}

	c.JSON(http.StatusOK, model.APIKeyListResponse{Keys: keys})
}

// Revoke
// @Summary Отзыв (деактивация) API ключа
// @Description Делает указанный API ключ недействительным. После этого он не может быть использован для аутентификации.
// @Tags API Keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body model.APIKeyRevokeRequest true "ID ключа для отзыва"
// @Success 200 "Ключ успешно отозван"
// @Failure 400 {object} map[string]string "Некорректный формат запроса"
// @Failure 401 {object} map[string]string "Пользователь не авторизован"
// @Failure 404 {object} map[string]string "Ключ не найден"
// @Failure 500 {object} map[string]string "Ошибка на стороне сервера"
// @Example request { "id": "7b2aab2e-4d1f-45b5-90c5-4d5d4db5ef11" }
// @Router /auth/api-key/revoke [post]
func (h *APIKeyHandler) Revoke(c *gin.Context) {
	var req model.APIKeyRevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.svc.Revoke(c.Request.Context(), req.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke"})
		return
	}

	c.Status(http.StatusNoContent)
}
