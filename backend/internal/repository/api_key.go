package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"hackathon-back/internal/model"
)

type APIKeyRepository struct {
	db *pgxpool.Pool
}

func NewAPIKeyRepository(db *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Insert — сохраняет новый ключ в базу
func (r *APIKeyRepository) Insert(ctx context.Context, key *model.APIKey) error {
	const q = `
		INSERT INTO sso.api_keys (user_id, key_hash, name, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at;
	`
	return r.db.QueryRow(ctx, q, key.UserID, key.KeyHash, key.Name, key.ExpiresAt).
		Scan(&key.ID, &key.CreatedAt)
}

// GetAllByUser — возвращает все активные ключи пользователя
func (r *APIKeyRepository) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]model.APIKey, error) {
	const q = `
		SELECT id, user_id, name, key_hash, created_at, expires_at, revoked
		FROM sso.api_keys
		WHERE user_id = $1 AND revoked = FALSE;
	`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.CreatedAt, &k.ExpiresAt, &k.Revoked); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// GetAllActive — все действующие ключи (для middleware)
func (r *APIKeyRepository) GetAllActive(ctx context.Context) ([]model.APIKey, error) {
	const q = `
		SELECT id, user_id, name, key_hash, created_at, expires_at, revoked
		FROM sso.api_keys
		WHERE revoked = FALSE AND (expires_at IS NULL OR expires_at > NOW());
	`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.CreatedAt, &k.ExpiresAt, &k.Revoked); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// Revoke — отзывает ключ
func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE sso.api_keys SET revoked = TRUE WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}
