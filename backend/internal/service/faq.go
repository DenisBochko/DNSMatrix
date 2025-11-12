// service/faq_service.go
package service

import (
	"context"
	"fmt"
	"hackathon-back/internal/model"
	"hackathon-back/internal/repository"
	"strings"

	"github.com/google/uuid"
)

type FAQRepository interface {
	Create(ctx context.Context, ext repository.RepoExtension, faq *model.FAQ) error
	GetByID(ctx context.Context, ext repository.RepoExtension, id uuid.UUID) (*model.FAQ, error)
	Update(ctx context.Context, ext repository.RepoExtension, id uuid.UUID, updateData *model.FAQUpdateRequest) error
	Delete(ctx context.Context, ext repository.RepoExtension, id uuid.UUID) error
	List(ctx context.Context, ext repository.RepoExtension, params model.FAQQueryParams) ([]model.FAQ, int, error)
	GetByCategory(ctx context.Context, ext repository.RepoExtension, category string) ([]model.FAQ, error)
	GetCategories(ctx context.Context, ext repository.RepoExtension) ([]string, error)
}

type FAQService struct {
	repo FAQRepository
}

func NewFAQService(repo FAQRepository) *FAQService {
	return &FAQService{
		repo: repo,
	}
}

// Create создает новый FAQ
func (s *FAQService) Create(ctx context.Context, req *model.FAQCreateRequest, createdBy uuid.UUID) (*model.FAQ, error) {
	// Валидация
	if strings.TrimSpace(req.Question) == "" {
		return nil, fmt.Errorf("question cannot be empty")
	}
	if strings.TrimSpace(req.Answer) == "" {
		return nil, fmt.Errorf("answer cannot be empty")
	}
	if strings.TrimSpace(req.Category) == "" {
		return nil, fmt.Errorf("category cannot be empty")
	}

	faq := &model.FAQ{
		ID:        uuid.New(),
		Question:  strings.TrimSpace(req.Question),
		Answer:    strings.TrimSpace(req.Answer),
		Category:  strings.TrimSpace(req.Category),
		Order:     req.Order,
		IsActive:  true,
		CreatedBy: createdBy,
	}

	if err := s.repo.Create(ctx, nil, faq); err != nil {
		return nil, fmt.Errorf("failed to create FAQ: %w", err)
	}

	return faq, nil
}

// GetByID возвращает FAQ по ID
func (s *FAQService) GetByID(ctx context.Context, id uuid.UUID) (*model.FAQ, error) {
	faq, err := s.repo.GetByID(ctx, nil, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get FAQ: %w", err)
	}
	return faq, nil
}

// Update обновляет FAQ
func (s *FAQService) Update(ctx context.Context, id uuid.UUID, req *model.FAQUpdateRequest) error {
	// Проверяем существование FAQ
	_, err := s.repo.GetByID(ctx, nil, id)
	if err != nil {
		return fmt.Errorf("FAQ not found: %w", err)
	}

	// Валидация обновляемых полей
	if req.Question != nil && strings.TrimSpace(*req.Question) == "" {
		return fmt.Errorf("question cannot be empty")
	}
	if req.Answer != nil && strings.TrimSpace(*req.Answer) == "" {
		return fmt.Errorf("answer cannot be empty")
	}
	if req.Category != nil && strings.TrimSpace(*req.Category) == "" {
		return fmt.Errorf("category cannot be empty")
	}

	if err := s.repo.Update(ctx, nil, id, req); err != nil {
		return fmt.Errorf("failed to update FAQ: %w", err)
	}

	return nil
}

// Delete удаляет FAQ
func (s *FAQService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, nil, id); err != nil {
		return fmt.Errorf("failed to delete FAQ: %w", err)
	}
	return nil
}

// List возвращает список FAQ с пагинацией и фильтрацией
func (s *FAQService) List(ctx context.Context, params model.FAQQueryParams) (*model.FAQListResponse, error) {
	// Устанавливаем значения по умолчанию
	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	faqs, total, err := s.repo.List(ctx, nil, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list FAQs: %w", err)
	}

	return &model.FAQListResponse{
		FAQs:  faqs,
		Total: total,
	}, nil
}

// GetByCategory возвращает активные FAQ по категории
func (s *FAQService) GetByCategory(ctx context.Context, category string) ([]model.FAQ, error) {
	if strings.TrimSpace(category) == "" {
		return nil, fmt.Errorf("category cannot be empty")
	}

	faqs, err := s.repo.GetByCategory(ctx, nil, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get FAQs by category: %w", err)
	}

	return faqs, nil
}

// GetCategories возвращает список всех категорий
func (s *FAQService) GetCategories(ctx context.Context) ([]string, error) {
	categories, err := s.repo.GetCategories(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	return categories, nil
}

// GetCategoriesWithFAQs возвращает FAQ сгруппированные по категориям
func (s *FAQService) GetCategoriesWithFAQs(ctx context.Context) ([]model.FAQCategoryResponse, error) {
	categories, err := s.repo.GetCategories(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	var result []model.FAQCategoryResponse
	for _, category := range categories {
		faqs, err := s.repo.GetByCategory(ctx, nil, category)
		if err != nil {
			continue // Пропускаем категории с ошибками
		}

		if len(faqs) > 0 {
			result = append(result, model.FAQCategoryResponse{
				Category: category,
				FAQs:     faqs,
			})
		}
	}

	return result, nil
}
