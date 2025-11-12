package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hackathon-back/internal/model"
)

type ArticleRepository interface {
	EnsureIndex(ctx context.Context) (err error)
	Create(ctx context.Context, article *model.Article) (err error)
	Get(ctx context.Context, id string) (article *model.Article, err error)
	Delete(ctx context.Context, id string) (err error)
	Patch(ctx context.Context, id string, fields map[string]interface{}) (err error)
	Search(ctx context.Context, query string, from, size int, sort string) (results []model.SearchResult, err error)
}

type ArticleService struct {
	articleRepo ArticleRepository
}

func NewArticleService(articleRepo ArticleRepository) *ArticleService {
	return &ArticleService{
		articleRepo: articleRepo,
	}
}

func (s *ArticleService) CreateArticle(ctx context.Context, req *model.ArticleCreateRequest) (*model.Article, error) {
	now := time.Now().UTC()

	article := &model.Article{
		ID:        uuid.New(),
		TitleRU:   req.TitleRU,
		TitleEN:   req.TitleEN,
		ContentRU: req.ContentRU,
		ContentEN: req.ContentEN,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.articleRepo.Create(ctx, article); err != nil {
		return nil, fmt.Errorf("failed to create article: %w", err)
	}

	return article, nil
}

func (s *ArticleService) GetArticle(ctx context.Context, id string) (*model.Article, error) {
	article, err := s.articleRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	return article, nil
}

func (s *ArticleService) DeleteArticle(ctx context.Context, id string) error {
	if err := s.articleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete article: %w", err)
	}

	return nil
}

func (s *ArticleService) UpdateArticle(ctx context.Context, id string, upd model.ArticleUpdate) error {
	doc := make(map[string]interface{}, 4)
	if upd.TitleRU != nil {
		doc["title_ru"] = *upd.TitleRU
	}

	if upd.TitleEN != nil {
		doc["title_en"] = *upd.TitleEN
	}

	if upd.ContentRU != nil {
		doc["content_ru"] = *upd.ContentRU
	}

	if upd.ContentEN != nil {
		doc["content_en"] = *upd.ContentEN
	}

	if len(doc) == 0 {
		return nil
	}

	if err := s.articleRepo.Patch(ctx, id, doc); err != nil {
		return fmt.Errorf("failed to update article: %w", err)
	}

	return nil
}

func (s *ArticleService) SearchArticles(ctx context.Context, query string) ([]model.SearchResult, error) {
	res, err := s.articleRepo.Search(ctx, query, 0, 10, "")
	if err != nil {
		return nil, fmt.Errorf("failed to search articles: %w", err)
	}

	return res, nil
}
