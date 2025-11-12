package model

import (
	"time"

	"github.com/google/uuid"
)

type VerificationToken struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"userID"`
	Token     []byte    `db:"token" json:"token"`
	Code      string    `db:"code" json:"code"`
	ExpiresAt time.Time `db:"expires_at" json:"expiresAt"`
}

// AuthRequest
// @Description Данные, передаваемые в json для регистрации.
type AuthRequest struct {
	Username string `binding:"required" example:"Dimka228"             json:"username"`                                 // Имя пользователя
	Email    string `binding:"required,email" example:"Dimka228@gmail.com" format:"email"      json:"email"`            // Электронная почта пользователя
	Password string `binding:"required" example:"12345678"            format:"password"  json:"password" minLength:"8"` // Пароль пользователя
} // @Name AuthRequest

// LoginRequest
// @Description Данные, для входа.
type LoginRequest struct {
	Email    string `binding:"required,email" example:"Dimka228@gmail.com" format:"email"      json:"email"`            // Электронная почта пользователя
	Password string `binding:"required" example:"12345678"            format:"password"  json:"password" minLength:"8"` // Пароль пользователя
} // @Name LoginRequest

// AuthToken
// @Description Токен для подтверждения регистрации.
type AuthToken struct {
	Token []byte `binding:"required" json:"token"` // Токен для подтверждения регистрации
} // @Name AuthToken

// AuthResponse
// @Description Данные, которые получает пользователь после регистрации.
type AuthResponse struct {
	User  *User
	Token []byte `binding:"required" json:"token"` // Токен для подтверждения регистрации
} // @Name AuthResponse

// ResendRequest
// @Description Запрос на переотправку кода подтверждения.
type ResendRequest struct {
	Email string `binding:"required,email" example:"Dimka228@gmail.com" format:"email"      json:"email"` // Электронная почта пользователя
} // @Name ResendRequest

// ConfirmationRequest
// @Description Запрос на подтверждение регистрации.
type ConfirmationRequest struct {
	Code  string `binding:"required" example:"0228" json:"code"`                        // Код, полученный с email
	Token string `binding:"required" example:"89as098ga0998=asdg=+afgk==" json:"token"` // Токен, который вернул handler register/resend-confirmation
} // @Name ConfirmationRequest

// TokenResponse
// @Description Ответ, содержащий access и refresh токены
type TokenResponse struct {
	AccessToken  string `json:"accessToken"`  // Access токен
	RefreshToken string `json:"refreshToken"` // Refresh токен
} // @Name TokenResponse

// RefreshRequest
// @Description Запрос, в котор передаёт refresh токен мобильное приложение
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"` // Refresh токен
} // @Name RefreshRequest

// ForgotPasswordRequest
// @Description Запрос на восстановление пароля.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"` // Почта, на которую придёт письмо для восстановления пароля
} // @Name ForgotPasswordRequest

// ResetPasswordRequest
// @Description Запрос на сброс пароля.
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`       // Токен, полученный по ссылке из письма
	NewPassword string `json:"newPassword" binding:"required"` // Новый пароль
} // @Name ResetPasswordRequest

// PasswordResetToken
// @Description Токен для восстановления пароля.
type PasswordResetToken struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"userID"`
	Token     []byte    `db:"token" json:"token"`
	ExpiresAt time.Time `db:"expires_at" json:"expiresAt"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
} // @Name PasswordResetToken
