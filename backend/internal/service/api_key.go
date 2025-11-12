package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"hackathon-back/internal/model"
)

type APIKeyRepository interface {
	Insert(ctx context.Context, key *model.APIKey) error
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]model.APIKey, error)
	GetAllActive(ctx context.Context) ([]model.APIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type APIKeyService struct {
	repo APIKeyRepository
}

func NewAPIKeyService(repo APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

// Generate — создаёт новый API ключ
func (s *APIKeyService) Generate(ctx context.Context, userID uuid.UUID, name string, ttl time.Duration) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	apiKey := base64.URLEncoding.EncodeToString(raw)

	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expiresAt = &t
	}

	key := &model.APIKey{
		UserID:    userID,
		Name:      name,
		KeyHash:   hash,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.Insert(ctx, key); err != nil {
		return "", err
	}

	return apiKey, nil
}

func (s *APIKeyService) GetUserKeys(ctx context.Context, userID uuid.UUID) ([]model.APIKey, error) {
	return s.repo.GetAllByUser(ctx, userID)
}

func (s *APIKeyService) Revoke(ctx context.Context, id uuid.UUID) error {
	return s.repo.Revoke(ctx, id)
}
