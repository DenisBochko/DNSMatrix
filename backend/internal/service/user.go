package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"hackathon-back/internal/model"
	"hackathon-back/internal/repository"
	"hackathon-back/pkg/mailer"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	Pool() *pgxpool.Pool

	InsertUser(ctx context.Context, ext repository.RepoExtension, user *model.User) (*model.User, error)
	InsertTestUser(ctx context.Context, ext repository.RepoExtension, user *model.User) (*model.User, error)
	SelectUserByID(ctx context.Context, ext repository.RepoExtension, id uuid.UUID) (*model.User, error)
	SelectUserByEmail(ctx context.Context, ext repository.RepoExtension, email string) (*model.User, error)
	Delete(ctx context.Context, ext repository.RepoExtension, id uuid.UUID) error
	Block(ctx context.Context, ext repository.RepoExtension, id uuid.UUID) error

	InsertPasswordResetToken(ctx context.Context, ext repository.RepoExtension, userID uuid.UUID, token []byte, expiresAt time.Time) error
	SelectUserByResetToken(ctx context.Context, ext repository.RepoExtension, token []byte) (*model.User, error)
	DeletePasswordResetToken(ctx context.Context, ext repository.RepoExtension, token []byte) error
	UpdateUserPassword(ctx context.Context, ext repository.RepoExtension, userID uuid.UUID, hashedPassword []byte) error
}

type UserService struct {
	userRepo UserRepository
	mailer   mailer.Mailer
}

func NewUserService(userRepo UserRepository, mlr mailer.Mailer) *UserService {
	return &UserService{
		userRepo: userRepo,
		mailer:   mlr,
	}
}

func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.userRepo.SelectUserByID(ctx, nil, id)
	if err != nil {
		return nil, fmt.Errorf("failed to select user: %w", err)
	}

	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if err := s.userRepo.Delete(ctx, nil, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (s *UserService) BlockUser(ctx context.Context, id uuid.UUID) error {
	if err := s.userRepo.Block(ctx, nil, id); err != nil {
		return fmt.Errorf("failed to block user: %w", err)
	}

	return nil
}

func (s *UserService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.userRepo.SelectUserByEmail(ctx, nil, email)
	if err != nil {
		return err
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}

	expiresAt := time.Now().Add(15 * time.Minute)

	if err := s.userRepo.InsertPasswordResetToken(ctx, nil, user.ID, tokenBytes, expiresAt); err != nil {
		return err
	}

	tokenStr := base64.URLEncoding.EncodeToString(tokenBytes)
	resetURL := fmt.Sprintf("https://frontend.example.com/reset-password?token=%s", tokenStr)

	if err := s.mailer.SendHTML(user.Email, "Password Reset", "Click here to reset your password", resetURL); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *UserService) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
	tokenBytes, err := base64.URLEncoding.DecodeString(tokenStr)
	if err != nil {
		return err
	}

	user, err := s.userRepo.SelectUserByResetToken(ctx, nil, tokenBytes)
	if err != nil {
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdateUserPassword(ctx, nil, user.ID, hashed); err != nil {
		return err
	}

	return s.userRepo.DeletePasswordResetToken(ctx, nil, tokenBytes)
}

func (s *UserService) DeleteSelf(ctx context.Context, userID uuid.UUID) error {
	return s.userRepo.Delete(ctx, nil, userID)
}
