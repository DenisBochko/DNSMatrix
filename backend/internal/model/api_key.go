package model

import (
	"github.com/google/uuid"
	"time"
)

// APIKey
// @Description Модель API ключа пользователя.
type APIKey struct {
	ID        uuid.UUID  `example:"7b2aab2e-4d1f-45b5-90c5-4d5d4db5ef11" json:"id"`                      // ID ключа
	UserID    uuid.UUID  `example:"1a2b3c4d-5678-90ab-cdef-1234567890ab" json:"user_id"`                 // ID пользователя
	Name      string     `example:"mobile_app" json:"name"`                                              // Название ключа (для понимания)
	KeyHash   []byte     `json:"-"`                                                                      // bcrypt-хэш ключа (не возвращается)
	CreatedAt time.Time  `example:"2025-10-25T13:40:00Z" format:"date-time" json:"created_at"`           // Дата создания
	ExpiresAt *time.Time `example:"2025-11-25T13:40:00Z" format:"date-time" json:"expires_at,omitempty"` // Дата истечения (если задана)
	Revoked   bool       `example:"false" json:"revoked"`                                                // Отозван ли ключ
} // @Name APIKey

// APIKeyCreateRequest
// @Description Запрос на создание API ключа.
type APIKeyCreateRequest struct {
	Name     string `binding:"required" example:"partner_api" json:"name"` // Имя ключа
	TTLHours int64  `example:"720" json:"ttl_hours,omitempty"`             // Время жизни в часах (0 — бессрочно)
} // @Name APIKeyCreateRequest

// APIKeyCreateResponse
// @Description Ответ при создании API ключа.
type APIKeyCreateResponse struct {
	APIKey string `example:"KJHsT9W-2oP3sA1Q-LzM8fD4eC" json:"api_key"` // Секретный API ключ (показывается один раз)
} // @Name APIKeyCreateResponse

// APIKeyListResponse
// @Description Ответ со списком ключей пользователя.
type APIKeyListResponse struct {
	Keys []APIKey `json:"keys"` // Список всех активных ключей
} // @Name APIKeyListResponse

// APIKeyRevokeRequest
// @Description Запрос на отзыв API ключа.
type APIKeyRevokeRequest struct {
	ID uuid.UUID `binding:"required" example:"7b2aab2e-4d1f-45b5-90c5-4d5d4db5ef11" json:"id"` // ID ключа
} // @Name APIKeyRevokeRequest
