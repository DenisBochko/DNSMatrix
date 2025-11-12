package model

import (
	"time"

	"github.com/google/uuid"
)

// ArticleCreateRequest
// @Description Данные, передаваемые для создания статьи.
type ArticleCreateRequest struct {
	TitleRU   string `binding:"required" json:"title_ru,omitempty"`   // Заголовок статьи на русском языке
	TitleEN   string `binding:"required" json:"title_en,omitempty"`   // Заголовок статьи на английском языке
	ContentRU string `binding:"required" json:"content_ru,omitempty"` // Содержимое статьи на русском
	ContentEN string `binding:"required" json:"content_en,omitempty"` // Содержимое статьи на английском
} // @Name ArticleCreateRequest

// Article
// @Description модель статьи.
type Article struct {
	ID        uuid.UUID `binding:"required" example:"b4b03119-1290-44bc-b599-6a5e91d6611f" json:"id"`                                                                               // ID статьи (UUID)
	TitleRU   string    `binding:"required" example:"Пример"                               json:"title_ru,omitempty"`                                                               // Заголовок статьи на русском языке
	TitleEN   string    `binding:"required" example:"Example"                              json:"title_en,omitempty"`                                                               // Заголовок статьи на английском языке
	ContentRU string    `binding:"required" example:"Пример"                               json:"content_ru,omitempty"`                                                             // Содержимое статьи на русском
	ContentEN string    `binding:"required" example:"Example"                              json:"content_en,omitempty"`                                                             // Содержимое статьи на английском
	CreatedAt time.Time `binding:"required" db:"created_at"                                example:"2006-01-02T15:04:05Z" format:"date-time" json:"createdAt" swaggertype:"string"` // Timestamp создания аккаунта
	UpdatedAt time.Time `binding:"required" db:"updated_at"                                example:"2006-01-02T15:04:05Z" format:"date-time" json:"updatedAt" swaggertype:"string"` // Timestamp последнего обновления аккаунта
} // @Name Article

type SearchResult struct {
	Article   Article             `json:"article"`
	Highlight map[string][]string `json:"highlight,omitempty"`
}

// ArticleUpdate
// @Description Модель обновления данных статьи.
// @Description Некоторые поля могут быть пустыми, обновляются только те, которые не пустые.
type ArticleUpdate struct {
	TitleRU   *string `json:"title_ru,omitempty"   example:"Пример" ` // Новый заголовок статьи на русском языке
	TitleEN   *string `json:"title_en,omitempty"   example:"Example"` // Новый заголовок статьи на английском языке
	ContentRU *string `json:"content_ru,omitempty" example:"Пример" ` // Новое содержимое статьи на русском
	ContentEN *string `json:"content_en,omitempty" example:"Example"` // Новое содержимое статьи на английском
} // @Name ArticleUpdate

// SearchParams Параметры при поисковом запросе статьи
type SearchParams struct {
	Q    string
	From int
	Size int
	Sort string // example: "created_at:desc"
}

type ArticleIDPathParam struct {
	ID string `uri:"article_id" binding:"required,uuid" example:"b4b03119-1290-44bc-b599-6a5e91d6611f"`
}

type ArticleQueryParams struct {
	Q string `binding:"required" form:"q"`
}
