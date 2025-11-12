package inbox

import (
	"context"
	"encoding/json"
	"fmt"
	"hackathon-back/internal/model"
	"hackathon-back/internal/repository"
	"hackathon-back/pkg/kafka"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const messagePipeBuffer = 1000

type InboxRepository interface {
	InsertMessage(ctx context.Context, ext repository.RepoExtension, message model.InboxMessage) error
	UpdateAsProcessed(ctx context.Context, ext repository.RepoExtension, messageID uuid.UUID) error
	SelectUnprocessedBatch(ctx context.Context, ext repository.RepoExtension, batchSize int) ([]model.InboxMessage, error)
}

type RequestRepository interface {
	Pool() *pgxpool.Pool

	InsertRequest(ctx context.Context, ext repository.RepoExtension, request *model.Request) error
	InsertAssignment(ctx context.Context, ext repository.RepoExtension, assignment *model.Assignment) error
	InsertCheckResult(ctx context.Context, ext repository.RepoExtension, checkResult *model.CheckResult) error
}

type Config struct {
	Name        string
	WorkerCount int
	BatchSize   int
	Topic       string
}

type Subscriber struct {
	l           *zap.Logger
	cfg         Config
	consumer    kafka.ConsumerGroupRunner
	inboxRepo   InboxRepository
	requestRepo RequestRepository
}

func NewSubscriber(l *zap.Logger, cfg Config, consumer kafka.ConsumerGroupRunner, inboxRepo InboxRepository, requestRepo RequestRepository) *Subscriber {
	return &Subscriber{
		l:           l,
		cfg:         cfg,
		consumer:    consumer,
		inboxRepo:   inboxRepo,
		requestRepo: requestRepo,
	}
}

func (s *Subscriber) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		s.consumer.Run()
	}()

	messagePipe := make(chan *kafka.MessageWithMarkFunc, messagePipeBuffer)

	for i := 0; i < s.cfg.WorkerCount; i++ {
		go s.worker(ctx, i, messagePipe)
	}

	for {
		select {
		case <-ctx.Done():
			s.l.Info("Context canceled, stopping inbox")

			close(messagePipe)

			return
		case msg, ok := <-s.consumer.Messages():
			if !ok {
				s.l.Info("Consumer messages channel closed")

				close(messagePipe)

				return
			}

			messagePipe <- msg
		}
	}
}

func (s *Subscriber) worker(ctx context.Context, id int, messagePipe <-chan *kafka.MessageWithMarkFunc) {
	s.l.Info("Inbox Worker started", zap.Int("worker_id", id))

	for {
		select {
		case <-ctx.Done():
			s.l.Info("Worker stopping", zap.Int("worker_id", id))

			return
		case msg, ok := <-messagePipe:
			if !ok {
				s.l.Info("Message channel closed", zap.Int("worker_id", id))

				return
			}

			messageID, err := uuid.FromBytes(msg.Message.Key)
			if err != nil {
				s.l.Error("Error parsing message id", zap.Int("worker_id", id), zap.Error(err))

				continue
			}

			if err := s.process(ctx, msg); err != nil {
				s.l.Error("Error processing message", zap.Int("worker_id", id), zap.Error(err))
			}

			s.l.Info("Message received",
				zap.String("message_id", messageID.String()),
			)

			msg.Mark()
		}
	}
}

func (s *Subscriber) process(ctx context.Context, message *kafka.MessageWithMarkFunc) (err error) {
	messageID, err := uuid.FromBytes(message.Message.Key)
	if err != nil {
		return fmt.Errorf("failed to parse message id: %w", err)
	}

	messageInbox := model.InboxMessage{
		ID:      messageID,
		Topic:   s.cfg.Topic,
		Payload: message.Message.Value,
	}

	var checkResultFromAgent model.CheckResultFromAgent
	if err := json.Unmarshal(message.Message.Value, &checkResultFromAgent); err != nil {
		return fmt.Errorf("failed to unmarshal checkResult: %w", err)
	}

	checkResult := &model.CheckResult{
		ID:           uuid.New(),
		AssignmentId: checkResultFromAgent.TaskID,
		Type:         checkResultFromAgent.Type,
		Status:       "DONE",
		StartedAt:    checkResultFromAgent.StartedAt,
		FinishedAt:   time.Now(),
		Payload:      message.Message.Value,
	}

	if checkResultFromAgent.OK {
		checkResult.Status = "DONE"
	} else {
		checkResult.Status = "FAILED"
	}

	tx, err := s.requestRepo.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("error begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			if rErr := tx.Rollback(ctx); rErr != nil {
				err = fmt.Errorf("%w, failed to rollback transaction: %w", err, rErr)
			}
		}
	}()

	if err := s.inboxRepo.InsertMessage(ctx, tx, messageInbox); err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	if err := s.requestRepo.InsertCheckResult(ctx, tx, checkResult); err != nil {
		return fmt.Errorf("failed to insert checkResult: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
