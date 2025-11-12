package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"hackathon-back/internal/apperrors"
	"hackathon-back/internal/model"
	"hackathon-back/internal/repository"
	"hackathon-back/pkg/jwt"
	"hackathon-back/pkg/mailer"
	"hackathon-back/pkg/redis"
)

const (
	welcomeMessage             = "Добро пожаловать! Подтвердите регистрацию."
	durationOfVerificationCode = 10 * time.Minute
)

const (
	textOfWelcomeMessage = `
		<h2>Привет, {{.Name}}!</h2>
		<p>Спасибо, что зарегистрировался.</p>
		<p>Код подтверждения регистрации: {{.Code}} </p>
	`
)

type AuthRepository interface {
	Pool() *pgxpool.Pool

	UpdateUserAsConfirmed(ctx context.Context, ext repository.RepoExtension, userID uuid.UUID) error
	InsertVerificationToken(ctx context.Context, ext repository.RepoExtension, verificationToken *model.VerificationToken) error
	SelectVerificationToken(ctx context.Context, ext repository.RepoExtension, token []byte) (*model.VerificationToken, error)
	DeleteVerificationTokenByUserID(ctx context.Context, ext repository.RepoExtension, userID uuid.UUID) error
}

type AuthService struct {
	log             *zap.Logger
	publicKey       *ecdsa.PublicKey
	privateKey      *ecdsa.PrivateKey
	authRepo        AuthRepository
	userRepo        UserRepository
	mlr             mailer.Mailer
	rdb             redis.Redis
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewAuthService(
	log *zap.Logger,
	publicKey *ecdsa.PublicKey,
	privateKey *ecdsa.PrivateKey,
	authRepo AuthRepository,
	userRepo UserRepository,
	mlr mailer.Mailer,
	rdb redis.Redis,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) *AuthService {
	return &AuthService{
		log:             log,
		publicKey:       publicKey,
		privateKey:      privateKey,
		authRepo:        authRepo,
		userRepo:        userRepo,
		mlr:             mlr,
		rdb:             rdb,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, username, email, password string) (user *model.User, userToken []byte, err error) {
	// Create user.
	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, []byte{}, fmt.Errorf("failed to generate password hash: %w", err)
	}

	userID := uuid.New()

	user = &model.User{
		ID:             userID,
		Username:       username,
		Email:          email,
		HashedPassword: passHash,
	}

	// Create user confirmation token.
	verificationToken, err := generateVerificationToken(user, durationOfVerificationCode)
	if err != nil {
		return nil, []byte{}, fmt.Errorf("failed to generate verification token: %w", err)
	}

	tx, err := s.authRepo.Pool().Begin(ctx)
	if err != nil {
		return nil, []byte{}, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	user, err = s.userRepo.InsertUser(ctx, tx, user)
	if err != nil {
		return nil, []byte{}, fmt.Errorf("failed to insert user: %w", err)
	}

	if err = s.authRepo.InsertVerificationToken(ctx, tx, verificationToken); err != nil {
		return nil, []byte{}, fmt.Errorf("failed to insert verification token: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, []byte{}, fmt.Errorf("error committing transaction: %w", err)
	}

	// Send email with code
	if err := s.mlr.SendHTML(user.Email, welcomeMessage, textOfWelcomeMessage, map[string]any{"Name": user.Username, "Code": verificationToken.Code}); err != nil {
		s.log.Error("failed to send verification code", zap.Error(err))
	}

	return user, verificationToken.Token, nil
}

func (s *AuthService) ResendConfirmation(ctx context.Context, email string) ([]byte, error) {
	user, err := s.userRepo.SelectUserByEmail(ctx, nil, email)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to select user: %w", err)
	}

	tx, err := s.authRepo.Pool().Begin(ctx)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := s.authRepo.DeleteVerificationTokenByUserID(ctx, nil, user.ID); err != nil {
		return []byte{}, fmt.Errorf("failed to delete verification token: %w", err)
	}

	verificationToken, err := generateVerificationToken(user, durationOfVerificationCode)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to generate verification token: %w", err)
	}

	if err = s.authRepo.InsertVerificationToken(ctx, tx, verificationToken); err != nil {
		return []byte{}, fmt.Errorf("failed to insert verification token: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return []byte{}, fmt.Errorf("error committing transaction: %w", err)
	}

	if err := s.mlr.SendHTML(user.Email, welcomeMessage, textOfWelcomeMessage, map[string]any{"Name": user.Username, "Code": verificationToken.Code}); err != nil {
		s.log.Error("failed to send verification code", zap.Error(err))
	}

	return verificationToken.Token, nil
}

func (s *AuthService) Confirmation(ctx context.Context, incCode string, incToken []byte) error {
	token, err := s.authRepo.SelectVerificationToken(ctx, nil, incToken)
	if err != nil {
		return fmt.Errorf("failed to select verification token: %w", err)
	}

	if incCode != token.Code {
		return apperrors.ErrInvalidVerificationCode
	}

	if token.ExpiresAt.Before(time.Now().UTC()) {
		return apperrors.ErrInvalidVerificationToken
	}

	if err := s.authRepo.UpdateUserAsConfirmed(ctx, nil, token.UserID); err != nil {
		return fmt.Errorf("failed to update user as confirmed: %w", err)
	}

	return nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error) {
	user, err := s.userRepo.SelectUserByEmail(ctx, nil, email)
	if err != nil {
		return "", "", fmt.Errorf("failed to select user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password))
	if err != nil {
		return "", "", apperrors.ErrInvalidCredentials
	}

	if !user.Confirmed {
		return "", "", apperrors.ErrUserIsNotConfirmed
	}

	accessToken, err = jwt.NewToken(s.privateKey, s.accessTokenTTL,
		jwt.WithClaim(model.UserUIDKey, user.ID),
		jwt.WithClaim(model.UserEmailKey, user.Email),
		jwt.WithClaim(model.UserNameKey, user.Username),
		jwt.WithClaim(model.UserConfirmedKey, user.Confirmed),
		jwt.WithClaim(model.UserRoleKey, user.Role),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken = uuid.New().String()

	if err := s.rdb.RDB().Set(ctx, refreshToken, user.ID.String(), s.refreshTokenTTL).Err(); err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.rdb.RDB().Del(ctx, refreshToken).Err(); err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	return nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, err error) {
	userID, err := s.rdb.RDB().Get(ctx, refreshToken).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return "", "", apperrors.ErrRefreshTokenExpired
		}

		return "", "", fmt.Errorf("failed to get refresh token: %w", err)
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse refresh token: %w", err)
	}

	user, err := s.userRepo.SelectUserByID(ctx, nil, uid)
	if err != nil {
		return "", "", fmt.Errorf("failed to select user: %w", err)
	}

	newAccessToken, err = jwt.NewToken(s.privateKey, s.accessTokenTTL,
		jwt.WithClaim(model.UserUIDKey, user.ID),
		jwt.WithClaim(model.UserEmailKey, user.Email),
		jwt.WithClaim(model.UserNameKey, user.Username),
		jwt.WithClaim(model.UserConfirmedKey, user.Confirmed),
		jwt.WithClaim(model.UserRoleKey, user.Role),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	rdbPipe := s.rdb.RDB().TxPipeline()
	newRefreshToken = uuid.New().String()

	rdbPipe.Del(ctx, refreshToken)
	rdbPipe.Set(ctx, newRefreshToken, user.ID.String(), s.refreshTokenTTL)

	_, eErr := rdbPipe.Exec(ctx)
	if eErr != nil {
		return "", "", fmt.Errorf("failed to exec transaction: %w", eErr)
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *AuthService) TestLogin(ctx context.Context) (accessToken, refreshToken string, err error) {
	passHash, err := bcrypt.GenerateFromPassword([]byte(gofakeit.Password(true, true, true, true, true, 15)), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate password hash: %w", err)
	}

	user := &model.User{
		ID:             uuid.New(),
		Username:       gofakeit.Username(),
		Email:          gofakeit.Email(),
		HashedPassword: passHash,
		Confirmed:      true,
		Deleted:        false,
		Blocked:        false,
		Role:           "user",
	}

	user, err = s.userRepo.InsertTestUser(ctx, nil, user)
	if err != nil {
		return "", "", fmt.Errorf("failed to insert test user: %w", err)
	}

	accessToken, err = jwt.NewToken(s.privateKey, s.accessTokenTTL,
		jwt.WithClaim("uid", user.ID),
		jwt.WithClaim("email", user.Email),
		jwt.WithClaim("name", user.Username),
		jwt.WithClaim("confirmed", user.Confirmed),
		jwt.WithClaim("role", user.Role),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken = uuid.New().String()
	if err := s.rdb.RDB().Set(ctx, refreshToken, user.ID, s.refreshTokenTTL).Err(); err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func generateVerificationToken(user *model.User, duration time.Duration) (*model.VerificationToken, error) {
	userDataJson, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %w", err)
	}

	userToken := make([]byte, 0, 32)
	for _, h := range sha256.Sum256(userDataJson) {
		userToken = append(userToken, h)
	}

	userVerificationCode, err := generate4DigitCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification code: %w", err)
	}

	return &model.VerificationToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     userToken,
		Code:      userVerificationCode,
		ExpiresAt: time.Now().UTC().Add(duration),
	}, nil
}

func generate4DigitCode() (string, error) {
	nBig, err := rand.Int(rand.Reader, big.NewInt(10000)) // 0..9999
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%04d", nBig.Int64()), nil
}
