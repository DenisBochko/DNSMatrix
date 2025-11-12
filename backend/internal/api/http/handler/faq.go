// handler/faq_handler.go
package handler

import (
	"context"
	"errors"
	"hackathon-back/internal/apperrors"
	"hackathon-back/internal/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FAQService interface {
	Create(ctx context.Context, req *model.FAQCreateRequest, createdBy uuid.UUID) (*model.FAQ, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.FAQ, error)
	Update(ctx context.Context, id uuid.UUID, req *model.FAQUpdateRequest) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params model.FAQQueryParams) (*model.FAQListResponse, error)
	GetByCategory(ctx context.Context, category string) ([]model.FAQ, error)
	GetCategories(ctx context.Context) ([]string, error)
	GetCategoriesWithFAQs(ctx context.Context) ([]model.FAQCategoryResponse, error)
}

type FAQHandler struct {
	BaseHandler
	svc FAQService
}

func NewFAQHandler(service FAQService) *FAQHandler {
	return &FAQHandler{
		svc: service,
	}
}

// CreateFAQ
// @Summary Создать новый FAQ
// @Description Создает новый часто задаваемый вопрос
// @Tags FAQ
// @Security AccessToken
// @Security RefreshToken
// @Accept json
// @Produce json
// @Param input body model.FAQCreateRequest true "Данные для создания FAQ"
// @Success 201 {object} ResponseWithData{data=model.FAQ} "FAQ успешно создан"
// @Failure 400 {object} ResponseWithMessage "Некорректные данные"
// @Failure 401 {object} ResponseWithMessage "Не авторизован"
// @Failure 500 {object} ResponseWithMessage "Ошибка при создании FAQ"
// @Router /faq [post]
func (h *FAQHandler) CreateFAQ(c *gin.Context) {
	ctx := c.Request.Context()

	userID, err := h.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ResponseWithMessage{
			Status:  StatusNotPermitted,
			Message: "User not authorized",
		})
		return
	}

	var req model.FAQCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	faq, err := h.svc.Create(ctx, &req, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, apperrors.ErrFAQAlreadyExists) {
			status = http.StatusConflict
		}

		c.JSON(status, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, ResponseWithData{
		Status: StatusSuccess,
		Data:   faq,
	})
}

// GetFAQ
// @Summary Получить FAQ по ID
// @Description Возвращает часто задаваемый вопрос по его ID
// @Tags FAQ
// @Produce json
// @Param id path string true "FAQ UUID"
// @Success 200 {object} ResponseWithData{data=model.FAQ} "Данные FAQ"
// @Failure 400 {object} ResponseWithMessage "Неверный параметр пути"
// @Failure 404 {object} ResponseWithMessage "FAQ не найден"
// @Failure 500 {object} ResponseWithMessage "Ошибка при получении FAQ"
// @Router /faq/{id} [get]
func (h *FAQHandler) GetFAQ(c *gin.Context) {
	ctx := c.Request.Context()

	var uri model.FAQIDPathParam
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	faqID, err := uuid.Parse(uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: "Invalid FAQ ID format",
		})
		return
	}

	faq, err := h.svc.GetByID(ctx, faqID)
	if err != nil {
		if errors.Is(err, apperrors.ErrFAQNotFound) {
			c.JSON(http.StatusNotFound, ResponseWithMessage{
				Status:  StatusErr,
				Message: "FAQ not found",
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
		Data:   faq,
	})
}

// UpdateFAQ
// @Summary Обновить FAQ
// @Description Обновляет данные часто задаваемого вопроса
// @Tags FAQ
// @Security AccessToken
// @Security RefreshToken
// @Accept json
// @Produce json
// @Param id path string true "FAQ UUID"
// @Param input body model.FAQUpdateRequest true "Данные для обновления"
// @Success 200 {object} ResponseWithMessage "FAQ успешно обновлён"
// @Failure 400 {object} ResponseWithMessage "Некорректные данные"
// @Failure 401 {object} ResponseWithMessage "Не авторизован"
// @Failure 404 {object} ResponseWithMessage "FAQ не найден"
// @Failure 500 {object} ResponseWithMessage "Ошибка при обновлении FAQ"
// @Router /faq/{id} [patch]
func (h *FAQHandler) UpdateFAQ(c *gin.Context) {
	ctx := c.Request.Context()

	var uri model.FAQIDPathParam
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	faqID, err := uuid.Parse(uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: "Invalid FAQ ID format",
		})
		return
	}

	var req model.FAQUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	if err := h.svc.Update(ctx, faqID, &req); err != nil {
		if errors.Is(err, apperrors.ErrFAQNotFound) {
			c.JSON(http.StatusNotFound, ResponseWithMessage{
				Status:  StatusErr,
				Message: "FAQ not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseWithMessage{
		Status:  StatusSuccess,
		Message: "FAQ updated successfully",
	})
}

// DeleteFAQ
// @Summary Удалить FAQ
// @Description Удаляет часто задаваемый вопрос
// @Tags FAQ
// @Security AccessToken
// @Security RefreshToken
// @Produce json
// @Param id path string true "FAQ UUID"
// @Success 200 {object} ResponseWithMessage "FAQ успешно удалён"
// @Failure 400 {object} ResponseWithMessage "Неверный параметр пути"
// @Failure 401 {object} ResponseWithMessage "Не авторизован"
// @Failure 404 {object} ResponseWithMessage "FAQ не найден"
// @Failure 500 {object} ResponseWithMessage "Ошибка при удалении FAQ"
// @Router /faq/{id} [delete]
func (h *FAQHandler) DeleteFAQ(c *gin.Context) {
	ctx := c.Request.Context()

	var uri model.FAQIDPathParam
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	faqID, err := uuid.Parse(uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: "Invalid FAQ ID format",
		})
		return
	}

	if err := h.svc.Delete(ctx, faqID); err != nil {
		if errors.Is(err, apperrors.ErrFAQNotFound) {
			c.JSON(http.StatusNotFound, ResponseWithMessage{
				Status:  StatusErr,
				Message: "FAQ not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseWithMessage{
		Status:  StatusSuccess,
		Message: "FAQ deleted successfully",
	})
}

// ListFAQs
// @Summary Получить список FAQ
// @Description Возвращает список часто задаваемых вопросов с пагинацией и фильтрацией
// @Tags FAQ
// @Produce json
// @Param category query string false "Фильтр по категории"
// @Param is_active query bool false "Фильтр по активности"
// @Param limit query int false "Лимит (по умолчанию 50, максимум 100)" default(50)
// @Param offset query int false "Смещение" default(0)
// @Success 200 {object} ResponseWithData{data=model.FAQListResponse} "Список FAQ"
// @Failure 400 {object} ResponseWithMessage "Некорректные параметры запроса"
// @Failure 500 {object} ResponseWithMessage "Ошибка при получении списка FAQ"
// @Router /faq [get]
func (h *FAQHandler) ListFAQs(c *gin.Context) {
	ctx := c.Request.Context()

	var params model.FAQQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	// Парсим boolean параметр
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ResponseWithMessage{
				Status:  StatusErr,
				Message: "Invalid is_active parameter",
			})
			return
		}
		params.IsActive = &isActive
	}

	result, err := h.svc.List(ctx, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data:   result,
	})
}

// GetFAQsByCategory
// @Summary Получить FAQ по категории
// @Description Возвращает активные часто задаваемые вопросы по указанной категории
// @Tags FAQ
// @Produce json
// @Param category path string true "Категория FAQ"
// @Success 200 {object} ResponseWithData{data=[]model.FAQ} "Список FAQ категории"
// @Failure 400 {object} ResponseWithMessage "Неверный параметр пути"
// @Failure 500 {object} ResponseWithMessage "Ошибка при получении FAQ"
// @Router /faq/category/{category} [get]
func (h *FAQHandler) GetFAQsByCategory(c *gin.Context) {
	ctx := c.Request.Context()

	category := c.Param("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: "Category parameter is required",
		})
		return
	}

	faqs, err := h.svc.GetByCategory(ctx, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data:   faqs,
	})
}

// GetCategories
// @Summary Получить список категорий
// @Description Возвращает список всех категорий FAQ
// @Tags FAQ
// @Produce json
// @Success 200 {object} ResponseWithData{data=[]string} "Список категорий"
// @Failure 500 {object} ResponseWithMessage "Ошибка при получении категорий"
// @Router /faq/categories [get]
func (h *FAQHandler) GetCategories(c *gin.Context) {
	ctx := c.Request.Context()

	categories, err := h.svc.GetCategories(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data:   categories,
	})
}

// GetCategoriesWithFAQs
// @Summary Получить FAQ по категориям
// @Description Возвращает все активные FAQ сгруппированные по категориям
// @Tags FAQ
// @Produce json
// @Success 200 {object} ResponseWithData{data=[]model.FAQCategoryResponse} "FAQ по категориям"
// @Failure 500 {object} ResponseWithMessage "Ошибка при получении данных"
// @Router /faq/grouped [get]
func (h *FAQHandler) GetCategoriesWithFAQs(c *gin.Context) {
	ctx := c.Request.Context()

	categoriesWithFAQs, err := h.svc.GetCategoriesWithFAQs(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data:   categoriesWithFAQs,
	})
}
