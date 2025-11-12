package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hackathon-back/internal/apperrors"
	"hackathon-back/internal/model"
)

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{
		db: db,
	}
}

func (r *AuthRepository) Pool() *pgxpool.Pool {
	return r.db
}

func (r *AuthRepository) UpdateUserAsConfirmed(ctx context.Context, ext RepoExtension, userID uuid.UUID) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
		UPDATE sso.users 
		SET confirmed = true,
			updated_at = NOW()
		WHERE id = $1 
		  AND deleted = false 
		  AND blocked = false;
	`

	_, err := ext.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	return nil
}

func (r *AuthRepository) InsertVerificationToken(ctx context.Context, ext RepoExtension, verificationToken *model.VerificationToken) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
		INSERT INTO sso.verification_tokens (id, user_id, token, code, expires_at)
		VALUES ($1, $2, $3, $4, $5);
	`

	_, err := ext.Exec(ctx, query,
		verificationToken.ID,
		verificationToken.UserID,
		verificationToken.Token,
		verificationToken.Code,
		verificationToken.ExpiresAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *AuthRepository) SelectVerificationToken(ctx context.Context, ext RepoExtension, token []byte) (*model.VerificationToken, error) {
	if ext == nil {
		ext = r.db
	}

	const query = `
		SELECT id, user_id, token, code, expires_at
		FROM sso.verification_tokens
		WHERE token = $1;
	`

	var verificationToken model.VerificationToken

	if err := ext.QueryRow(ctx, query, token).Scan(
		&verificationToken.ID,
		&verificationToken.UserID,
		&verificationToken.Token,
		&verificationToken.Code,
		&verificationToken.ExpiresAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrTokenDoesNotExist
		}

		return nil, err
	}

	return &verificationToken, nil
}

func (r *AuthRepository) DeleteVerificationTokenByUserID(ctx context.Context, ext RepoExtension, userID uuid.UUID) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
		DELETE FROM sso.verification_tokens 
		WHERE user_id = $1;
	`

	res, err := ext.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return apperrors.ErrTokenDoesNotExist
	}

	return nil
}
