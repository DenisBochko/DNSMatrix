package apperrors

import (
	"errors"
)

var (
	ErrShutdown = errors.New("shutdown error")

	ErrTestDataDoesNotExist = errors.New("test data does not exist")

	ErrUserAlreadyExists        = errors.New("user already exists")
	ErrUserDoesNotExist         = errors.New("user does not exist")
	ErrTokenDoesNotExist        = errors.New("token does not exist")
	ErrInvalidVerificationCode  = errors.New("invalid verification code")
	ErrInvalidVerificationToken = errors.New("token has expired")
	ErrUserIsNotConfirmed       = errors.New("user isn't confirmed")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrRefreshTokenExpired      = errors.New("refresh token expired")

	ErrContextValueDoesNotExist = errors.New("context value does not exist")
	ErrContextValueInvalidType  = errors.New("invalid context value type")

	ErrArticleDoesNotExist = errors.New("article does not exist")

	ErrFAQAlreadyExists = errors.New("faq already exists")
	ErrFAQNotFound      = errors.New("faq does not exist")
)
