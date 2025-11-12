package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"hackathon-back/internal/apperrors"
	"hackathon-back/internal/model"
)

type AuthService interface {
	Register(ctx context.Context, username, email, password string) (user *model.User, userToken []byte, err error)
	ResendConfirmation(ctx context.Context, email string) ([]byte, error)
	Confirmation(ctx context.Context, incCode string, incToken []byte) error
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error)
	Logout(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, err error)
	TestLogin(ctx context.Context) (accessToken, refreshToken string, err error)
}

type AuthHandler struct {
	log             *zap.Logger
	svc             AuthService
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewAuthHandler(log *zap.Logger, svc AuthService, accessTokenTTL, refreshTokenTTL time.Duration) *AuthHandler {
	return &AuthHandler{
		log:             log,
		svc:             svc,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

// Register
// @Summary Регистрация пользователя.
// @Description Принимает данные об имени, почте, пароле. Далее сохраняет пользователя в бд,
// @Description отправляет 4-х значный код на почту и возвращает структуру пользователя, токен, который нужен для подтверждения регистрации.
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body model.AuthRequest true "Данные для регистрации пользователя"
// @Success 201 {object} ResponseWithData{data=model.AuthResponse} "Success"
// @Failure 400 {object} ResponseWithMessage "Invalid JSON body"
// @Failure 409 {object} ResponseWithMessage "User already exists"
// @Failure 500 {object} ResponseWithMessage "Failed to register user"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	ctx := c.Request.Context()

	var req model.AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	user, token, err := h.svc.Register(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, ResponseWithMessage{
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

	c.JSON(http.StatusCreated, ResponseWithData{
		Status: StatusSuccess,
		Data: model.AuthResponse{
			User:  user,
			Token: token,
		},
	})
}

// ResendConfirmation
// @Summary Повторная отправка токена кода подтверждения.
// @Description Генерирует новую пару токен + 4-х значный код, код отправляет на почту,
// @Description возвращает новый токен, старый аннулируется.
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body model.ResendRequest true "Данные для повторной отправки кода подтверждения"
// @Success 200 {object} ResponseWithData{data=model.AuthToken} "Success"
// @Failure 400 {object} ResponseWithMessage "Invalid JSON body"
// @Failure 500 {object} ResponseWithMessage "Failed to resend confirmation"
// @Router /auth/resend-confirmation [post]
func (h *AuthHandler) ResendConfirmation(c *gin.Context) {
	ctx := c.Request.Context()

	var req model.ResendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	token, err := h.svc.ResendConfirmation(ctx, req.Email)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserDoesNotExist) {
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

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data: model.AuthToken{
			Token: token,
		},
	})
}

// Confirmation
// @Summary Подтверждение пользователя.
// @Description Принимает код с почты + временный токен, который вернул handler register/resend-confirmation.
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body model.ConfirmationRequest true "Данные для подтверждения пользователя"
// @Success 200 {object} ResponseWithMessage "User confirmed successfully"
// @Failure 400 {object} ResponseWithMessage "Invalid JSON body"
// @Failure 401 {object} ResponseWithMessage "Invalid verification code/Token has expired"
// @Failure 404 {object} ResponseWithMessage "Token does not exist"
// @Failure 500 {object} ResponseWithMessage "Failed to confirmation user"
// @Router /auth/confirm [post]
func (h *AuthHandler) Confirmation(c *gin.Context) {
	ctx := c.Request.Context()

	var req model.ConfirmationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	decodedToken, err := base64.StdEncoding.DecodeString(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: "Invalid token format",
		})

		return
	}

	if err := h.svc.Confirmation(ctx, req.Code, decodedToken); err != nil {
		if errors.Is(err, apperrors.ErrTokenDoesNotExist) {
			c.JSON(http.StatusNotFound, ResponseWithMessage{
				Status:  StatusErr,
				Message: err.Error(),
			})

			return
		}

		if errors.Is(err, apperrors.ErrInvalidVerificationCode) {
			c.JSON(http.StatusUnauthorized, ResponseWithMessage{
				Status:  StatusErr,
				Message: err.Error(),
			})

			return
		}

		if errors.Is(err, apperrors.ErrInvalidVerificationToken) {
			c.JSON(http.StatusUnauthorized, ResponseWithMessage{
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
		Message: "User confirmed successfully",
	})
}

// Login
// @Summary Login пользователя.
// @Description Принимает почту и пароль, выставляет access и refresh токен в cookie, они автоматически отправляются при каждом запросе к api.
// @Description Login только подтверждённых пользователей.
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body model.LoginRequest true "Данные для подтверждения входа"
// @Success 200 {object} ResponseWithData{data=model.TokenResponse} “Success”
// @Failure 400 {object} ResponseWithMessage "Invalid JSON body"
// @Failure 401 {object} ResponseWithMessage "Invalid credential/User isn't confirmed"
// @Failure 404 {object} ResponseWithMessage "User does not exist"
// @Failure 500 {object} ResponseWithMessage "Failed to login user"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	accessToken, refreshToken, err := h.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserDoesNotExist) {
			c.JSON(http.StatusNotFound, ResponseWithMessage{
				Status:  StatusErr,
				Message: err.Error(),
			})

			return
		}

		if errors.Is(err, apperrors.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, ResponseWithMessage{
				Status:  StatusErr,
				Message: err.Error(),
			})

			return
		}

		if errors.Is(err, apperrors.ErrUserIsNotConfirmed) {
			c.JSON(http.StatusUnauthorized, ResponseWithMessage{
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

	c.SetCookie("access", accessToken, int(h.accessTokenTTL.Seconds()), "/", "", true, true)
	c.SetCookie("refresh", refreshToken, int(h.refreshTokenTTL.Seconds()), "/", "", true, true)

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data: model.TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	})
}

// Logout
// @Summary Logout пользователя.
// @Description Аннигиляция access и refresh токена.
// @Description Для web-клиентов токен автоматически берётся из cookies, затем access и refresh токены сбрасываются.
// @Description Для мобильного клиента .
// @Tags Auth
// @Accept json
// @Produce json
// @Param token body model.RefreshRequest true "Refresh токен (Нужно только при передаче токена из мобильного проложения!)"
// @Success 200 {object} ResponseWithMessage "Logged out"
// @Failure 400 {object} ResponseWithMessage "Invalid JSON body"
// @Failure 500 {object} ResponseWithMessage "Failed to logout"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx := c.Request.Context()

	var refreshToken string

	if cookie, err := c.Cookie("refresh"); err == nil {
		refreshToken = cookie
	}

	if refreshToken == "" {
		var req model.RefreshRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ResponseWithMessage{
				Status:  StatusErr,
				Message: err.Error(),
			})

			return
		}

		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		h.clearCookies(c)

		c.JSON(http.StatusOK, ResponseWithMessage{
			Status:  StatusSuccess,
			Message: "Logged out",
		})

		return
	}

	if err := h.svc.Logout(ctx, refreshToken); err != nil {
		h.log.Error("Failed to delete refresh token from redis",
			zap.Error(err),
		)
	}

	h.clearCookies(c)

	c.JSON(http.StatusOK, ResponseWithMessage{
		Status:  StatusSuccess,
		Message: "Logged out",
	})
}

// Refresh
// @Summary Refresh jwt токенов.
// @Description Получает refresh токен из cookies (Для мобильного приложения нужно передать токен в теле запроса), проверяет его, если он не истёк, то выставляет новые access и refresh токены.
// @Tags Auth
// @Accept json
// @Produce json
// @Param token body model.RefreshRequest true "Refresh токен (Нужно только при передаче токена из мобильного проложения!)"
// @Success 200 {object} ResponseWithData{data=model.TokenResponse} “Success”
// @Failure 400 {object} ResponseWithMessage "Invalid JSON body"
// @Failure 401 {object} ResponseWithMessage "Refresh token expired"
// @Failure 404 {object} ResponseWithMessage "User does not exist"
// @Failure 500 {object} ResponseWithMessage "Failed to refresh user"
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	ctx := c.Request.Context()

	var refreshToken string

	if cookie, err := c.Cookie("refresh"); err == nil {
		refreshToken = cookie
	}

	if refreshToken == "" {
		var req model.RefreshRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ResponseWithMessage{
				Status:  StatusErr,
				Message: err.Error(),
			})

			return
		}

		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ResponseWithMessage{
			Status:  StatusNotPermitted,
			Message: "Missing refresh token",
		})

		return
	}

	accessToken, refreshToken, err := h.svc.Refresh(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, apperrors.ErrRefreshTokenExpired) {
			c.JSON(http.StatusUnauthorized, ResponseWithMessage{
				Status:  StatusNotPermitted,
				Message: err.Error(),
			})

			return
		}

		if errors.Is(err, apperrors.ErrUserDoesNotExist) {
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

	c.SetCookie("access", accessToken, int(h.accessTokenTTL.Seconds()), "/", "", true, true)
	c.SetCookie("refresh", refreshToken, int(h.refreshTokenTTL.Seconds()), "/", "", true, true)

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data: model.TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	})
}

// TestLogin
// @Summary Тестовый единоразовый вход.
// @Description Создаёт пользователя с рандомными данными, выставляет access токен в cookie.
// @Description	Учти, читатель, пользователь здесь не сохраняется в бд, т.е. refresh токен не выставляется и в теле возвращается пустая строка.
// @Description	Из этого можно сделать вывод, что этот логин будет действителен примерно 20 минут.
// @Tags Auth
// @Produce json
// @Success 200 {object} ResponseWithData{data=model.TokenResponse} “Success”
// @Failure 500 {object} ResponseWithMessage "Failed to login user"
// @Router /auth/test-login [post]
func (h *AuthHandler) TestLogin(c *gin.Context) {
	ctx := c.Request.Context()

	accessToken, refreshToken, err := h.svc.TestLogin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})

		return
	}

	c.SetCookie("access", accessToken, int(h.accessTokenTTL.Seconds()), "/", "", true, true)

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data: model.TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	})
}

func (h *AuthHandler) clearCookies(c *gin.Context) {
	c.SetCookie("access", "", -1, "/", "", true, true)
	c.SetCookie("refresh", "", -1, "/", "", true, true)
}
