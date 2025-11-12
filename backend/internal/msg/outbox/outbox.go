package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"hackathon-back/internal/model"
	"hackathon-back/internal/repository"
	"hackathon-back/pkg/kafka"
)

const BathSizeMultiply = 5

type Repository interface {
	UpdateAsSent(ctx context.Context, ext repository.RepoExtension, messageID uuid.UUID) error
	SelectUnsentBatch(ctx context.Context, ext repository.RepoExtension, batchSize int) ([]model.OutboxMessage, error)
}

type Config struct {
	Name         string
	WorkerCount  int
	PollInterval time.Duration
	BatchSize    int
}

type Publisher struct {
	l          *zap.Logger
	cfg        Config
	producer   kafka.Producer
	outboxRepo Repository
}

func NewPublisher(l *zap.Logger, cfg Config, producer kafka.Producer, outboxRepo Repository) *Publisher {
	return &Publisher{
		l:          l,
		cfg:        cfg,
		producer:   producer,
		outboxRepo: outboxRepo,
	}
}

func (p *Publisher) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	messagePipe := make(chan model.OutboxMessage, p.cfg.BatchSize*BathSizeMultiply)

	for i := 0; i < p.cfg.WorkerCount; i++ {
		go p.worker(ctx, i, messagePipe)
	}

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.l.Info("Outbox publisher stopped")
			close(messagePipe)

			return
		case <-ticker.C:
			messages, err := p.outboxRepo.SelectUnsentBatch(ctx, nil, p.cfg.BatchSize)
			if err != nil {
				p.l.Error("Failed to select unsent messages", zap.Error(err))
				continue
			}

			for _, msg := range messages {
				messagePipe <- msg
			}
		}
	}
}

func (p *Publisher) worker(ctx context.Context, id int, messagePipe <-chan model.OutboxMessage) {
	p.l.Info("OutBox Worker started", zap.Int("id", id))

	for {
		select {
		case <-ctx.Done():
			p.l.Info("Worker stopping", zap.Int("id", id))

			return
		case msg, ok := <-messagePipe:
			if !ok {
				p.l.Info("Message channel closed", zap.Int("id", id))

				return
			}

			partition, offset, err := p.sendAndMark(ctx, msg)
			if err != nil {
				p.l.Error("Failed to send message", zap.Error(err), zap.String("message_id", msg.ID.String()))
			}

			p.l.Info("Message sent",
				zap.String("message_id", msg.ID.String()),
				zap.Int32("partition", partition),
				zap.Int64("offset", offset),
			)
		}
	}
}

func (p *Publisher) sendAndMark(ctx context.Context, message model.OutboxMessage) (partition int32, offset int64, err error) {
	messageID, err := message.ID.MarshalBinary()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal message id: %w", err)
	}

	partition, offset, err = p.producer.PushMessage(ctx, messageID, message.Payload, message.Topic)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to push message: %w", err)
	}

	if err := p.outboxRepo.UpdateAsSent(ctx, nil, message.ID); err != nil {
		return 0, 0, fmt.Errorf("failed to update as sent: %w", err)
	}

	return partition, offset, nil
}
