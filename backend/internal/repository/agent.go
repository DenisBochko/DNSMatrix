package repository

import (
	"context"
	"hackathon-back/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentRepository struct {
	db *pgxpool.Pool
}

func NewAgentRepository(db *pgxpool.Pool) *AgentRepository {
	return &AgentRepository{
		db: db,
	}
}

func (r *AgentRepository) Pool() *pgxpool.Pool {
	return r.db
}

func (r *AgentRepository) SelectAgents(ctx context.Context, ext RepoExtension) ([]*model.Agent, error) {
	if ext == nil {
		ext = r.db
	}

	var agents []*model.Agent

	const query = `
		SELECT id, region, asn, online, updated_at 
		FROM domain.agents;
	`

	rows, err := ext.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var agent model.Agent
		if err := rows.Scan(
			&agent.ID,
			&agent.Region,
			&agent.ASN,
			&agent.Online,
			&agent.UpdatedAt,
		); err != nil {
			return nil, err
		}

		agents = append(agents, &agent)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return agents, nil
}

func (r *AgentRepository) SelectAgentByRegion(ctx context.Context, ext RepoExtension, region string) (*model.Agent, error) {
	if ext == nil {
		ext = r.db
	}

	var agent model.Agent

	const query = `
		SELECT id, region, asn, online, updated_at
		FROM domain.agents
		WHERE region = $1;
	`

	if err := ext.QueryRow(ctx, query, region).Scan(
		&agent.ID,
		&agent.Region,
		&agent.ASN,
		&agent.Online,
		&agent.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &agent, nil
}
