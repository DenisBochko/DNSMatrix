package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"hackathon-back/internal/apperrors"
	"hackathon-back/internal/model"
)

type HealthService interface {
	IsOK() (bool, error)
	GetTestData(ctx context.Context) (*model.TestTable, error)
}

type HealthHandler struct {
	BaseHandler

	log *zap.Logger
	svc HealthService
}

func NewHealthHandler(log *zap.Logger, svc HealthService) *HealthHandler {
	return &HealthHandler{
		BaseHandler: BaseHandler{},
		log:         log,
		svc:         svc,
	}
}

// Ping
// @Summary Проверка здоровья сервиса.
// @Description Возвращает “pong”.
// @Tags Health
// @Produce json
// @Success 200 {object} ResponseWithMessage "Success"
// @Router /health/ping [get]
func (h *HealthHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, ResponseWithMessage{
		Status:  StatusSuccess,
		Message: "pong",
	})
}

// ProtectedPing
// @Summary Проверка здоровья сервиса, используя JWT.
// @Description Возвращает “pong” + uuid пользователя из JWT, который её вызвал.
// @Tags Health
// @Produce json
// @Security AccessToken
// @Security RefreshToken
// @Success 200 {object} ResponseWithMessage "Success + id"
// @Failure 401 {object} ResponseWithMessage "Invalid or missing token"
// @Failure 403 {object} ResponseWithMessage "Invalid user data format"
// @Failure 401 {object} ResponseWithMessage "Invalid or missing JWT token"
// @Router /health/protected/ping [get]
func (h *HealthHandler) ProtectedPing(c *gin.Context) {
	_ = c.Request.Context()

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

	c.JSON(http.StatusOK, ResponseWithMessage{
		Status:  StatusSuccess,
		Message: fmt.Sprintf("pong + %s", userID.String()),
	})
}

// Health
// На эту ручку не обращай внимание, дорогой читатель, это просто я сделал по фану,
// Когда только собриал +- архитектуру, оставил на память.
func (h *HealthHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()

	_, err := h.svc.IsOK()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	data, err := h.svc.GetTestData(ctx)
	if err != nil {
		if errors.Is(err, apperrors.ErrTestDataDoesNotExist) {
			c.JSON(http.StatusNotFound, ResponseWithMessage{
				Status:  StatusErr,
				Message: err.Error(),
			})

			return
		}

		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data:   data,
	})
}
