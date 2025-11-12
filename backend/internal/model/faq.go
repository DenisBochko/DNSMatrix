// model/faq.go
package model

import (
	"time"

	"github.com/google/uuid"
)

// FAQ - модель часто задаваемого вопроса
type FAQ struct {
	ID        uuid.UUID `json:"id" example:"7b2aab2e-4d1f-45b5-90c5-4d5d4db5ef11"`
	Question  string    `json:"question" example:"Как восстановить пароль?"`
	Answer    string    `json:"answer" example:"Для восстановления пароля используйте форму 'Забыли пароль' на странице входа."`
	Category  string    `json:"category" example:"authentication"`
	Order     int       `json:"order" example:"1"`
	IsActive  bool      `json:"is_active" example:"true"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy uuid.UUID `json:"created_by" example:"1a2b3c4d-5678-90ab-cdef-1234567890ab"`
}

// FAQCreateRequest - запрос на создание FAQ
type FAQCreateRequest struct {
	Question string `json:"question" binding:"required" example:"Как восстановить пароль?"`
	Answer   string `json:"answer" binding:"required" example:"Для восстановления пароля используйте форму 'Забыли пароль' на странице входа."`
	Category string `json:"category" binding:"required" example:"authentication"`
	Order    int    `json:"order" example:"1"`
}

// FAQUpdateRequest - запрос на обновление FAQ
type FAQUpdateRequest struct {
	Question *string `json:"question,omitempty" example:"Как сбросить пароль?"`
	Answer   *string `json:"answer,omitempty" example:"Используйте кнопку 'Забыли пароль' на странице входа."`
	Category *string `json:"category,omitempty" example:"auth"`
	Order    *int    `json:"order,omitempty" example:"2"`
	IsActive *bool   `json:"is_active,omitempty" example:"false"`
}

// FAQListResponse - ответ со списком FAQ
type FAQListResponse struct {
	FAQs  []FAQ `json:"faqs"`
	Total int   `json:"total"`
}

// FAQCategoryResponse - ответ с FAQ по категориям
type FAQCategoryResponse struct {
	Category string `json:"category"`
	FAQs     []FAQ  `json:"faqs"`
}

// FAQQueryParams - параметры запроса для фильтрации FAQ
type FAQQueryParams struct {
	Category string `form:"category" example:"authentication"`
	IsActive *bool  `form:"is_active" example:"true"`
	Limit    int    `form:"limit" example:"10"`
	Offset   int    `form:"offset" example:"0"`
}

// FAQIDPathParam - параметр пути для ID FAQ
type FAQIDPathParam struct {
	ID string `uri:"id" binding:"required,uuid"`
}
