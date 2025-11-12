// nolint: staticcheck // Ignore imports.
package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"hackathon-back/internal/app"
	"hackathon-back/internal/config"
	"hackathon-back/internal/docs"
	_ "hackathon-back/internal/docs" // DO NOT REMOVE MFK
	"hackathon-back/pkg/logger"
)

// @title Hackathon API
// @version 0.1.0
// @description Флоу авторизации: сначала пользователь регистрируется, в ответе получает модельку своего user и токен.
// @description Токен + код с почты он отправляет на ручку confirm, если что-то идёт не так (токен истёк, не пришёл код на почту), то запрос нужно отправить на ручку resend-confirmation.
// @description Далее уже можно авторизоваться, login/refresh/test-login выставляет в cookie access и refresh токены, фронтенду, ничего с ними делать не нужно, они сами по себе живут в браузере и отправляются при каждом запросе.
// @description Специально для мобильного приложения при login/refresh/test-login токены дублируются в теле ответа.
// @description При запросе к защищённым ручкам API мобильному приложению необходимо выставить заголовок Authorization: Bearer *access_token*.
// @description При refresh мобильное приложение передаёт refresh токен в теле запроса.
// @host localhost:8080
// @BasePath /api/
func main() {
	ctx := context.Background()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.MustLoadConfig()
	config.MustPrintConfig(cfg)

	docs.SwaggerInfo.Title = cfg.ServiceName
	docs.SwaggerInfo.Version = cfg.Version
	docs.SwaggerInfo.BasePath = cfg.BasePath
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", cfg.HTTPServer.Port)

	loggerCfg := &logger.Config{
		Level:      cfg.Level,
		FormatJSON: cfg.FormatJSON,
		Rotation: logger.Rotation{
			File:       cfg.Rotation.File,
			MaxSize:    cfg.Rotation.MaxSize,
			MaxBackups: cfg.Rotation.MaxBackups,
			MaxAge:     cfg.Rotation.MaxAge,
		},
	}

	log := logger.MustSetupLogger(loggerCfg)

	errors := make(chan error)

	application := app.MustNew(cfg, log)

	defer func() {
		close(errors)

		if err := application.Shutdown(); err != nil {
			log.Error("Failed to shutdown application", zap.Error(err))
		}

		if err := log.Sync(); err != nil {
			log.Warn("Failed to sync logger", zap.Error(err))
		}

		log.Info("Application has shutdown")
	}()

	go func() { errors <- application.Run(ctx) }()

	select {
	case err := <-errors:
		if err != nil {
			log.Error("Server error, shutting down...", zap.Error(err))
		}
	case <-ctx.Done():
		log.Info("Received stop signal, shutting down...")
	}
}
