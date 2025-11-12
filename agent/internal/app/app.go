package app

import (
	"context"
	"fmt"
	"hackathon-agent/internal/config"
	"hackathon-agent/internal/service"
	"hackathon-agent/pkg/kafka"

	"go.uber.org/zap"
)

type App struct {
	Cfg     *config.Config
	Log     *zap.Logger
	EBus    *EBus
	Service *service.Service
}

type EBus struct {
	Consumer kafka.ConsumerGroupRunner
	Producer kafka.Producer
}

func New(cfg *config.Config, log *zap.Logger) (*App, error) {
	eBus, err := initEBus(cfg, log)
	if err != nil {
		return nil, err
	}

	svc := initService(cfg, log, eBus)

	return &App{
		Cfg:     cfg,
		Log:     log,
		EBus:    eBus,
		Service: svc,
	}, nil
}

func MustNew(cfg *config.Config, log *zap.Logger) *App {
	app, err := New(cfg, log)
	if err != nil {
		panic(err)
	}

	return app
}

func (a *App) Run(ctx context.Context) error {
	if err := a.Service.Run(ctx); err != nil {
		return fmt.Errorf("failed to run service: %w", err)
	}

	return nil
}

func (a *App) Shutdown() error {
	if err := a.Service.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}

func initEBus(cfg *config.Config, log *zap.Logger) (*EBus, error) {
	consumerGroup, err := kafka.NewConsumerGroupRunner(
		cfg.Subscriber.Brokers,
		cfg.Subscriber.GroupID,
		[]string{cfg.Subscriber.Topic},
		cfg.Subscriber.BufferSize,
		kafka.WithBalancerConsumer(kafka.RoundrobinBalanceStrategy),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	go func() {
		startAndRunningStr := <-consumerGroup.Info()

		log.Info(startAndRunningStr)
	}()

	producer, err := kafka.NewProducer(
		cfg.Publisher.Brokers,
		kafka.WithBalancer(kafka.RoundRobin),
		kafka.WithRequiredAcks(kafka.RequireAll),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init kafka producer: %w", err)
	}

	return &EBus{
		Consumer: consumerGroup,
		Producer: producer,
	}, nil
}

func initService(cfg *config.Config, log *zap.Logger, eBus *EBus) *service.Service {
	svc := service.NewService(log, eBus.Consumer, eBus.Producer, cfg.Publisher.Topic)
	return svc
}
