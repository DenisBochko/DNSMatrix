package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"hackathon-back/internal/model"
)

type RequestRepository struct {
	db *pgxpool.Pool
}

func NewRequestRepository(db *pgxpool.Pool) *RequestRepository {
	return &RequestRepository{
		db: db,
	}
}

func (r *RequestRepository) Pool() *pgxpool.Pool {
	return r.db
}

func (r *RequestRepository) InsertRequest(ctx context.Context, ext RepoExtension, request *model.Request) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
		INSERT INTO domain.requests (id, 
		                             target, 
		                             timeout_seconds,
		                             broadcast,
		                             client_ip,
		                             user_agent,
		                             client_asn,
		                             client_cc,
		                             client_region,
		                             checks_types,
		                             request_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING status, created_at, updated_at;
	`

	err := ext.QueryRow(ctx, query,
		request.ID,
		request.Target,
		request.TimeoutSeconds,
		request.Broadcast,
		request.ClientIP,
		request.UserAgent,
		request.ClientASN,
		request.ClientCC,
		request.ClientRegion,
		request.ChecksTypes,
		request.RequestJSON,
	).Scan(&request.Status, &request.CreatedAt, &request.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *RequestRepository) InsertAssignment(ctx context.Context, ext RepoExtension, assignment *model.Assignment) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
		INSERT INTO domain.assignments (
		                                id,
		                                request_id,
		                                agent_id,
		                                agent_region,
		                                outbox_id
		)
		VALUES ($1, $2, $3, $4, $5);
	`

	_, err := ext.Exec(ctx, query,
		assignment.ID,
		assignment.RequestID,
		assignment.AgentID,
		assignment.AgentRegion,
		assignment.OutboxID,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *RequestRepository) InsertCheckResult(ctx context.Context, ext RepoExtension, checkResult *model.CheckResult) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
		INSERT INTO domain.check_results (
		                                  id,
		                                  assignment_id,
		                                  type,
		                                  status,
		                                  started_at,
		                                  finished_at,
		                                  payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7);
	`

	_, err := ext.Exec(ctx, query,
		checkResult.ID,
		checkResult.AssignmentId,
		checkResult.Type,
		checkResult.Status,
		checkResult.StartedAt,
		checkResult.FinishedAt,
		checkResult.Payload,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *RequestRepository) SelectResultsByRequestID(ctx context.Context, ext RepoExtension, requestID uuid.UUID) ([]model.CheckResultResponse, error) {
	if ext == nil {
		ext = r.db
	}

	var result []model.CheckResultResponse

	const query = `
		SELECT a.request_id, a.agent_id, a.agent_region, c.type, c.status, c.started_at, c.finished_at, c.payload
		FROM domain.assignments a
		JOIN domain.check_results c ON a.request_id = c.assignment_id
		WHERE a.request_id = $1
		ORDER BY a.agent_id;
	`

	rows, err := ext.Query(ctx, query, requestID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var r model.CheckResultResponse

		if err := rows.Scan(
			&r.RequestID,
			&r.AgentID,
			&r.AgentRegion,
			&r.Type,
			&r.Status,
			&r.StartedAt,
			&r.FinishedAt,
			&r.Payload,
		); err != nil {
			return nil, err
		}

		result = append(result, r)
	}

	return result, nil
}
