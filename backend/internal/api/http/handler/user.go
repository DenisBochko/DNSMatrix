package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"hackathon-back/internal/apperrors"
	"hackathon-back/internal/model"
)

type UserService interface {
	GetUser(ctx context.Context, id uuid.UUID) (*model.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	BlockUser(ctx context.Context, id uuid.UUID) error

	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	DeleteSelf(ctx context.Context, userID uuid.UUID) error
}

type UserHandler struct {
	BaseHandler
	svc UserService
}

func NewUserHandler(service UserService) *UserHandler {
	return &UserHandler{
		svc: service,
	}
}

// DeleteUser
// @Summary Удалить пользователя по ID
// @Description Удаляет пользователя по ID. Доступно для пользователей с ролью manager и выше.
// @Tags User
// @Security AccessToken
// @Security RefreshToken
// @Produce json
// @Param user_id path string true "User UUID"
// @Success 200 {object} ResponseWithMessage "Пользователь успешно удалён"
// @Failure 400 {object} ResponseWithMessage "Неверный параметр пути"
// @Failure 401 {object} ResponseWithMessage "Не авторизован"
// @Failure 403 {object} ResponseWithMessage "Недостаточно прав"
// @Failure 404 {object} ResponseWithMessage "Пользователь не найден"
// @Failure 500 {object} ResponseWithMessage "Ошибка при удалении пользователя"
// @Router /user/{user_id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	ctx := c.Request.Context()

	var uri model.UserIDPathParam
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	userUID, err := uuid.Parse(uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	if err := h.svc.DeleteUser(ctx, userUID); err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: "Failed to delete user",
		})

		return
	}

	c.JSON(http.StatusOK, ResponseWithMessage{
		Status:  StatusSuccess,
		Message: "User deleted successfully",
	})
}

// BlockUser
// @Summary Заблокировать пользователя по ID
// @Description Блокирует пользователя по ID. Доступно для пользователей с ролью manager и выше.
// @Tags User
// @Security AccessToken
// @Security RefreshToken
// @Produce json
// @Param user_id path string true "User UUID"
// @Success 200 {object} ResponseWithMessage "Пользователь успешно заблокирован"
// @Failure 400 {object} ResponseWithMessage "Неверный параметр пути"
// @Failure 401 {object} ResponseWithMessage "Не авторизован"
// @Failure 403 {object} ResponseWithMessage "Недостаточно прав"
// @Failure 404 {object} ResponseWithMessage "Пользователь не найден"
// @Failure 500 {object} ResponseWithMessage "Ошибка при блокировке пользователя"
// @Router /user/block/{user_id} [post]
func (h *UserHandler) BlockUser(c *gin.Context) {
	ctx := c.Request.Context()

	var uri model.UserIDPathParam
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	userUID, err := uuid.Parse(uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	if err := h.svc.BlockUser(ctx, userUID); err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, ResponseWithMessage{
		Status:  StatusSuccess,
		Message: "User blocked successfully",
	})
}

// GetUser
// @Summary Получить пользователя по ID
// @Description Получает информацию о пользователе по его ID. Доступно без авторизации.
// @Tags User
// @Produce json
// @Param user_id path string true "User UUID"
// @Success 200 {object} ResponseWithData{data=model.User} "Данные пользователя"
// @Failure 400 {object} ResponseWithMessage "Неверный параметр пути"
// @Failure 404 {object} ResponseWithMessage "Пользователь не найден"
// @Failure 500 {object} ResponseWithMessage "Ошибка при получении пользователя"
// @Router /user/{user_id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	ctx := c.Request.Context()

	var uri model.UserIDPathParam
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	userUID, err := uuid.Parse(uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	user, err := h.svc.GetUser(ctx, userUID)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserDoesNotExist) {
			c.JSON(http.StatusNotFound, ResponseWithMessage{
				Status:  StatusErr,
				Message: err.Error(),
			})

			return
		}

		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data:   user,
	})
}

// GetUserJWT
// @Summary Получить данные текущего пользователя
// @Description Получает информацию о текущем авторизованном пользователе. ID берётся из JWT токена.
// @Tags User
// @Security AccessToken
// @Security RefreshToken
// @Produce json
// @Success 200 {object} ResponseWithData{data=model.User} "Данные пользователя"
// @Failure 401 {object} ResponseWithMessage "Неверный или отсутствующий токен"
// @Failure 403 {object} ResponseWithMessage "Неверный формат данных пользователя"
// @Failure 404 {object} ResponseWithMessage "Пользователь не найден"
// @Failure 500 {object} ResponseWithMessage "Ошибка при получении пользователя"
// @Router /user [get]
func (h *UserHandler) GetUserJWT(c *gin.Context) {
	ctx := c.Request.Context()

	userID, err := h.GetUserID(c)
	if err != nil {
		if errors.Is(err, apperrors.ErrContextValueDoesNotExist) {
			c.JSON(http.StatusUnauthorized, ResponseWithMessage{
				Status:  StatusNotPermitted,
				Message: "no data about the user",
			})

			return
		}

		if errors.Is(err, apperrors.ErrContextValueInvalidType) {
			c.JSON(http.StatusForbidden, ResponseWithMessage{
				Status:  StatusNotPermitted,
				Message: "invalid user data format",
			})

			return
		}
	}

	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserDoesNotExist) {
			c.JSON(http.StatusNotFound, ResponseWithMessage{
				Status:  StatusErr,
				Message: err.Error(),
			})

			return
		}

		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data:   user,
	})
}

// ForgotPassword
// @Summary Запрос сброса пароля
// @Description Отправляет письмо со ссылкой для восстановления пароля на указанный email
// @Tags User
// @Accept json
// @Produce json
// @Param input body model.ForgotPasswordRequest true "Email пользователя"
// @Success 200 {object} ResponseWithMessage "Ссылка для сброса пароля отправлена на email"
// @Failure 400 {object} ResponseWithMessage "Некорректный запрос"
// @Failure 500 {object} ResponseWithMessage "Ошибка сервера"
// @Router /user/password-forgot [post]
func (h *UserHandler) ForgotPassword(c *gin.Context) {
	ctx := c.Request.Context()

	var req model.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	if err := h.svc.RequestPasswordReset(ctx, req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseWithMessage{Status: StatusSuccess, Message: "Password reset link sent to email"})
}

// ResetPassword
// @Summary Сброс пароля
// @Description Сбрасывает пароль пользователя по токену. После успешной смены токен становится недействительным.
// @Tags User
// @Security AccessToken
// @Security RefreshToken
// @Accept json
// @Produce json
// @Param input body model.ResetPasswordRequest true "Данные для сброса пароля"
// @Success 200 {object} ResponseWithMessage "Пароль успешно изменён"
// @Failure 400 {object} ResponseWithMessage "Некорректные данные"
// @Failure 500 {object} ResponseWithMessage "Ошибка сервера"
// @Router /user/password-reset [post]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	ctx := c.Request.Context()

	var req model.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{Status: StatusErr, Message: err.Error()})
		return
	}

	if err := h.svc.ResetPassword(ctx, req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{Status: StatusInternalError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ResponseWithMessage{Status: StatusSuccess, Message: "Password reset successful"})
}

// DeleteSelf
// @Summary Удалить свой аккаунт
// @Description Удаляет аккаунт текущего авторизованного пользователя
// @Tags User
// @Security AccessToken
// @Security RefreshToken
// @Produce json
// @Success 200 {object} ResponseWithMessage "Аккаунт успешно удалён"
// @Failure 401 {object} ResponseWithMessage "Пользователь не авторизован"
// @Failure 500 {object} ResponseWithMessage "Ошибка при удалении аккаунта"
// @Router /user [delete]
func (h *UserHandler) DeleteSelf(c *gin.Context) {
	ctx := c.Request.Context()

	userID, err := h.GetUserID(c)
	if err != nil {
		if errors.Is(err, apperrors.ErrContextValueDoesNotExist) {
			c.JSON(http.StatusUnauthorized, ResponseWithMessage{
				Status:  StatusNotPermitted,
				Message: "no data about the user",
			})

			return
		}

		if errors.Is(err, apperrors.ErrContextValueInvalidType) {
			c.JSON(http.StatusForbidden, ResponseWithMessage{
				Status:  StatusNotPermitted,
				Message: "invalid user data format",
			})

			return
		}
	}

	if err := h.svc.DeleteSelf(ctx, userID); err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{Status: StatusInternalError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ResponseWithMessage{Status: StatusSuccess, Message: "User deleted"})
}
