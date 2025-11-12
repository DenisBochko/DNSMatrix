package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"hackathon-back/internal/apperrors"
	"hackathon-back/internal/model"
)

type ArticleService interface {
	CreateArticle(ctx context.Context, req *model.ArticleCreateRequest) (*model.Article, error)
	GetArticle(ctx context.Context, id string) (*model.Article, error)
	DeleteArticle(ctx context.Context, id string) error
	UpdateArticle(ctx context.Context, id string, upd model.ArticleUpdate) error
	SearchArticles(ctx context.Context, query string) ([]model.SearchResult, error)
}

type ArticleHandler struct {
	svc ArticleService
}

func NewArticleHandler(svc ArticleService) *ArticleHandler {
	return &ArticleHandler{
		svc: svc,
	}
}

// CreateArticle
// @Summary Создать статью.
// @Description Создать статью.
// @Tags Articles
// @Security AccessToken
// @Security RefreshToken
// @Accept json
// @Produce json
// @Param article body model.ArticleCreateRequest true "Данные для создания статьи"
// @Success 201 {object} ResponseWithData{data=model.Article} "Success"
// @Failure 400 {object} ResponseWithMessage "Invalid body param"
// @Failure 500 {object} ResponseWithMessage "Failed to get article"
// @Router /article [post]
func (h *ArticleHandler) CreateArticle(c *gin.Context) {
	ctx := c.Request.Context()

	var req model.ArticleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	art, err := h.svc.CreateArticle(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusInternalError,
			Message: err.Error()},
		)

		return
	}

	c.JSON(http.StatusCreated, ResponseWithData{
		Status: StatusSuccess,
		Data:   art,
	})
}

// GetArticle
// @Summary Получить статью по ID.
// @Description Получить статью по ID.
// @Tags Articles
// @Security AccessToken
// @Security RefreshToken
// @Produce json
// @Param article_id path string true "Article UUID"
// @Success 200 {object} ResponseWithData{data=model.Article} "Success"
// @Failure 400 {object} ResponseWithMessage "Invalid path param"
// @Failure 404 {object} ResponseWithMessage "Article not found"
// @Failure 500 {object} ResponseWithMessage "Failed to get article"
// @Router /article/{article_id} [get]
func (h *ArticleHandler) GetArticle(c *gin.Context) {
	ctx := c.Request.Context()

	var uri model.ArticleIDPathParam
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	art, err := h.svc.GetArticle(ctx, uri.ID)
	if err != nil {
		if errors.Is(err, apperrors.ErrArticleDoesNotExist) {
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
		Data:   art,
	})
}

// UpdateArticle
// @Summary Частично обновить статью.
// @Description Частично обновить статью. Некоторые поля могут быть пустыми, обновляются только те, которые не пустые.
// @Tags Articles
// @Security AccessToken
// @Security RefreshToken
// @Accept json
// @Produce json
// @Param article_id path string true "Article UUID"
// @Param updateArticle body model.ArticleCreateRequest true "Поля для обновления статьи"
// @Success 200 {object} ResponseWithMessage "Success"
// @Failure 400 {object} ResponseWithMessage "Invalid path param"
// @Failure 404 {object} ResponseWithMessage "Article not found"
// @Failure 500 {object} ResponseWithMessage "Failed to update article"
// @Router /article/{article_id} [patch]
func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	ctx := c.Request.Context()

	var uri model.ArticleIDPathParam
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	var req model.ArticleUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	if err := h.svc.UpdateArticle(ctx, uri.ID, req); err != nil {
		if errors.Is(err, apperrors.ErrArticleDoesNotExist) {
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

	c.JSON(http.StatusOK, ResponseWithMessage{
		Status:  StatusSuccess,
		Message: "Updated successfully",
	})
}

// DeleteArticle
// @Summary Удалить статью.
// @Description Удалить статью по ID.
// @Tags Articles
// @Security AccessToken
// @Security RefreshToken
// @Produce json
// @Param article_id path string true "Article UUID"
// @Success 200 {object} ResponseWithMessage "Success"
// @Failure 400 {object} ResponseWithMessage "Invalid path param"
// @Failure 404 {object} ResponseWithMessage "Article not found"
// @Failure 500 {object} ResponseWithMessage "Failed to delete article"
// @Router /article/{article_id} [delete]
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	ctx := c.Request.Context()

	var uri model.ArticleIDPathParam
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	if err := h.svc.DeleteArticle(ctx, uri.ID); err != nil {
		if errors.Is(err, apperrors.ErrArticleDoesNotExist) {
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

	c.JSON(http.StatusOK, ResponseWithMessage{
		Status:  StatusSuccess,
		Message: "Deleted successfully",
	})
}

// SearchArticles
// @Summary Поиск статей по содержанию.
// @Description Полнотекстовый поиск по статьям.
// @Tags Articles
// @Security AccessToken
// @Security RefreshToken
// @Produce json
// @Param q query string true "Строка поиска"
// @Success 200 {object} ResponseWithData{data=[]model.SearchResult} "Success"
// @Failure 400 {object} ResponseWithMessage "Invalid query param"
// @Failure 500 {object} ResponseWithMessage "Failed to get articles"
// @Router /article/search [get]
func (h *ArticleHandler) SearchArticles(c *gin.Context) {
	ctx := c.Request.Context()

	var qp model.ArticleQueryParams
	if err := c.ShouldBindQuery(&qp); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	res, err := h.svc.SearchArticles(ctx, qp.Q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data:   res,
	})
}
