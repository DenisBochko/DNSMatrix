package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"hackathon-back/internal/model"
)

type OutboxRepository struct {
	db *pgxpool.Pool
}

func NewOutboxRepository(db *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{
		db: db,
	}
}

func (r *OutboxRepository) InsertMessage(ctx context.Context, ext RepoExtension, message model.OutboxMessage) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
        INSERT INTO messages.outbox_messages (id, topic, payload)
		VALUES ($1, $2, $3)
        ON CONFLICT DO NOTHING;
    `

	_, err := ext.Exec(ctx, query, message.ID, message.Topic, message.Payload)
	if err != nil {
		return err
	}

	return nil
}

func (r *OutboxRepository) UpdateAsSent(ctx context.Context, ext RepoExtension, messageID uuid.UUID) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
        UPDATE messages.outbox_messages
        SET sent = true, sent_at = NOW()
        WHERE id = $1;
    `

	_, err := ext.Exec(ctx, query, messageID)
	if err != nil {
		return err
	}

	return nil
}

func (r *OutboxRepository) SelectUnsentBatch(ctx context.Context, ext RepoExtension, batchSize int) ([]model.OutboxMessage, error) {
	if ext == nil {
		ext = r.db
	}

	var messages []model.OutboxMessage

	const query = `
        SELECT id, topic, payload, created_at, sent, sent_at
        FROM messages.outbox_messages
        WHERE sent = false
        ORDER BY created_at
        LIMIT $1;
    `

	rows, err := ext.Query(ctx, query, batchSize)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var message model.OutboxMessage
		if err := rows.Scan(
			&message.ID,
			&message.Topic,
			&message.Payload,
			&message.CreatedAt,
			&message.Sent,
			&message.SentAt,
		); err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	return messages, nil
}
