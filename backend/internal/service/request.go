package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"hackathon-back/internal/model"
	"hackathon-back/internal/repository"
	"hackathon-back/pkg/geoip"
)

var baseProduceTopic = "hosts-check"

type RequestRepository interface {
	Pool() *pgxpool.Pool

	SelectResultsByRequestID(ctx context.Context, ext repository.RepoExtension, requestID uuid.UUID) ([]model.CheckResultResponse, error)
	InsertRequest(ctx context.Context, ext repository.RepoExtension, request *model.Request) error
	InsertAssignment(ctx context.Context, ext repository.RepoExtension, assignment *model.Assignment) error
	InsertCheckResult(ctx context.Context, ext repository.RepoExtension, checkResult *model.CheckResult) error
}

type OutboxRepository interface {
	InsertMessage(ctx context.Context, ext repository.RepoExtension, message model.OutboxMessage) error
	UpdateAsSent(ctx context.Context, ext repository.RepoExtension, messageID uuid.UUID) error
	SelectUnsentBatch(ctx context.Context, ext repository.RepoExtension, batchSize int) ([]model.OutboxMessage, error)
}

type AgentRepository interface {
	SelectAgents(ctx context.Context, ext repository.RepoExtension) ([]*model.Agent, error)
	SelectAgentByRegion(ctx context.Context, ext repository.RepoExtension, region string) (*model.Agent, error)
}

type GeoIPDB interface {
	Lookup(ip net.IP) geoip.GeoInfo
}

type RequestService struct {
	log         *zap.Logger
	requestRepo RequestRepository
	outboxRepo  OutboxRepository
	agentRepo   AgentRepository
	geo         GeoIPDB
}

func NewRequestService(log *zap.Logger, requestRepo RequestRepository, outboxRepo OutboxRepository, agentRepo AgentRepository, geo GeoIPDB) *RequestService {
	return &RequestService{
		log:         log,
		requestRepo: requestRepo,
		outboxRepo:  outboxRepo,
		agentRepo:   agentRepo,
		geo:         geo,
	}
}

func (s *RequestService) CreateRequest(ctx context.Context, req model.TaskMessageRequest, ip net.IP, ua string) (request *model.Request, error error) {
	gi := s.geo.Lookup(ip)

	id := uuid.New()

	taskMessage := &model.TaskMessage{
		ID:             id,
		Target:         req.Target,
		TimeoutSeconds: req.TimeoutSeconds,
		ClientContext: model.ClientContext{
			IP:  ip.String(),
			ASN: gi.ASN,
			Geo: model.Geo{
				Region:    gi.Region,
				Continent: gi.Continent,
			},
			UserAgent: ua,
		},
		Checks:   make([]model.CheckRequest, 0, len(req.Checks)),
		Metadata: map[string]string{"origin": "api", "region": gi.Region},
	}

	checkTypes := make([]string, 0, len(req.Checks))

	for _, check := range req.Checks {
		taskMessage.Checks = append(taskMessage.Checks, model.CheckRequest{
			Type:   check.Type,
			Params: check.Params,
		})

		checkTypes = append(checkTypes, check.Type)
	}

	payload, err := json.Marshal(taskMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task message: %w", err)
	}

	request = &model.Request{
		ID:             id,
		Target:         req.Target,
		TimeoutSeconds: req.TimeoutSeconds,
		Broadcast:      req.Broadcast,
		ClientIP:       ip.String(),
		UserAgent:      ua,
		ClientASN:      gi.ASN,
		ClientCC:       gi.CC,
		ClientRegion:   gi.Region,
		// Status
		ChecksTypes: checkTypes,
		RequestJSON: payload,
		// CreatedAt
		// UpdatedAt
	}

	if req.Broadcast {
		tx, err := s.requestRepo.Pool().Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}

		defer func() {
			if rErr := tx.Rollback(ctx); rErr != nil {
				err = fmt.Errorf("%w, failed to roll back transaction: %w", err, rErr)
			}
		}()

		if err := s.requestRepo.InsertRequest(ctx, tx, request); err != nil {
			return nil, fmt.Errorf("failed to insert request: %w", err)
		}

		allAgents, err := s.agentRepo.SelectAgents(ctx, tx)
		if err != nil {
			return nil, fmt.Errorf("failed to select agents: %w", err)
		}

		for _, agent := range allAgents {
			topic := fmt.Sprintf("%s-%s", baseProduceTopic, agent.Region)
			outboxID := uuid.New()

			outboxMessage := model.OutboxMessage{
				ID:      outboxID,
				Topic:   topic,
				Payload: payload,
			}

			assignment := &model.Assignment{
				ID:          uuid.New(),
				RequestID:   id,
				AgentID:     agent.ID,
				AgentRegion: agent.Region,
				OutboxID:    outboxID,
			}

			if err := s.outboxRepo.InsertMessage(ctx, tx, outboxMessage); err != nil {
				return nil, fmt.Errorf("failed to insert outbox message: %w", err)
			}

			if err := s.requestRepo.InsertAssignment(ctx, tx, assignment); err != nil {
				return nil, fmt.Errorf("failed to insert assignment: %w", err)
			}
		}

		if err = tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("error committing transaction: %w", err)
		}
	} else {
		tx, err := s.requestRepo.Pool().Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}

		defer func() {
			if rErr := tx.Rollback(ctx); rErr != nil {
				err = fmt.Errorf("%w, failed to roll back transaction: %w", err, rErr)
			}
		}()

		if err := s.requestRepo.InsertRequest(ctx, tx, request); err != nil {
			return nil, fmt.Errorf("failed to insert request: %w", err)
		}

		agent, err := s.agentRepo.SelectAgentByRegion(ctx, tx, gi.Region)
		if err != nil {
			return nil, fmt.Errorf("failed to select agent: %w", err)
		}

		topic := fmt.Sprintf("%s-%s", baseProduceTopic, gi.Region)
		outboxID := uuid.New()

		outboxMessage := model.OutboxMessage{
			ID:      outboxID,
			Topic:   topic,
			Payload: payload,
		}

		assignment := &model.Assignment{
			ID:          uuid.New(),
			RequestID:   id,
			AgentID:     agent.ID,
			AgentRegion: agent.Region,
			OutboxID:    outboxID,
		}

		if err := s.outboxRepo.InsertMessage(ctx, tx, outboxMessage); err != nil {
			return nil, fmt.Errorf("failed to insert outbox message: %w", err)
		}

		if err := s.requestRepo.InsertAssignment(ctx, tx, assignment); err != nil {
			return nil, fmt.Errorf("failed to insert assignment: %w", err)
		}

		if err = tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("error committing transaction: %w", err)
		}
	}

	return request, nil
}

func (s *RequestService) GetResultsByRequestID(ctx context.Context, requestID uuid.UUID) ([]model.CheckResultResponse, error) {
	results, err := s.requestRepo.SelectResultsByRequestID(ctx, nil, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to select results: %w", err)
	}

	return results, nil
}
