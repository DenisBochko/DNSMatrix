package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleUser    = "user"
)

// User
// @Description Модель пользователя, хз что ещё сказать можно по этому поводу.
type User struct {
	ID             uuid.UUID `binding:"required,uuid" db:"id"                   example:"b4b03119-1290-44bc-b599-6a5e91d6611f"                    json:"id"`        // ID пользователя
	Username       string    `db:"username"             example:"Dimka228"             json:"username"`                                                             // Имя пользователя
	Email          string    `binding:"required,email" db:"email"                example:"Dimka228@gmail.com"   json:"email"`                                       // Электронная почта пользователя
	HashedPassword []byte    `db:"password"             json:"-"                       swaggerignore:"true"`                                                        // Хэш пароля
	Confirmed      bool      `binding:"required" db:"confirmed"            example:"true"                 json:"confirmed"`                                         // Подтверждён ли пользователь
	Deleted        bool      `binding:"required" db:"deleted"              example:"true"                 json:"deleted"`                                           // Удалён ли пользователь
	Blocked        bool      `binding:"required" db:"blocked"              example:"false"                json:"blocked"`                                           // Заблокирован ли пользователь
	Role           string    `binding:"required" db:"role"                 example:"user"                 json:"role"`                                              // Роль пользователя
	CreatedAt      time.Time `binding:"required" db:"created_at"           example:"2006-01-02T15:04:05Z" format:"date-time" json:"createdAt" swaggertype:"string"` // Timestamp создания аккаунта
	UpdatedAt      time.Time `binding:"required" db:"updated_at"           example:"2006-01-02T15:04:05Z" format:"date-time" json:"updatedAt" swaggertype:"string"` // Timestamp последнего обновления аккаунта
} // @Name User

type UserIDPathParam struct {
	ID string `uri:"user_id" binding:"required,uuid" example:"b4b03119-1290-44bc-b599-6a5e91d6611f"`
}
