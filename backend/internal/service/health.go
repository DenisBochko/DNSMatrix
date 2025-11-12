package service

import (
	"context"

	"go.uber.org/zap"

	"hackathon-back/internal/model"
	"hackathon-back/internal/repository"
)

type HealthRepository interface {
	IsOK() (bool, error)
	SelectData(ctx context.Context, ext repository.RepoExtension) (*model.TestTable, error)
}

type HealthService struct {
	log        *zap.Logger
	healthRepo HealthRepository
}

func NewHealthService(log *zap.Logger, healthRepo HealthRepository) *HealthService {
	return &HealthService{
		log:        log,
		healthRepo: healthRepo,
	}
}

func (s *HealthService) IsOK() (bool, error) {
	s.log.Debug("HealthService.IsOK()")

	ok, err := s.healthRepo.IsOK()
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (s *HealthService) GetTestData(ctx context.Context) (*model.TestTable, error) {
	data, err := s.healthRepo.SelectData(ctx, nil)
	if err != nil {
		return nil, err
	}

	return data, nil
}
