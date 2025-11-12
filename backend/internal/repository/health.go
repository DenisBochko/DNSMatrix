package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hackathon-back/internal/apperrors"
	"hackathon-back/internal/model"
)

type HealthRepository struct {
	db *pgxpool.Pool
}

func NewHealthRepository(db *pgxpool.Pool) *HealthRepository {
	return &HealthRepository{
		db: db,
	}
}

func (r *HealthRepository) IsOK() (bool, error) {
	return true, nil
}

func (r *HealthRepository) SelectData(ctx context.Context, ext RepoExtension) (*model.TestTable, error) {
	if ext == nil {
		ext = r.db
	}

	const query = `
		SELECT id, data FROM test.init;
	`

	var testTable model.TestTable

	if err := ext.QueryRow(ctx, query).Scan(
		&testTable.ID,
		&testTable.Data,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrTestDataDoesNotExist
		}
	}

	return &testTable, nil
}
