package repository

import (
	"context"
	"hackathon-back/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InboxRepository struct {
	db *pgxpool.Pool
}

func NewInboxRepository(db *pgxpool.Pool) *InboxRepository {
	return &InboxRepository{db: db}
}

func (r *InboxRepository) InsertMessage(ctx context.Context, ext RepoExtension, message model.InboxMessage) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
        INSERT INTO messages.inbox_messages (id, topic, payload)
		VALUES ($1, $2, $3)
        ON CONFLICT DO NOTHING;
    `

	_, err := ext.Exec(ctx, query, message.ID, message.Topic, message.Payload)
	if err != nil {
		return err
	}

	return nil
}

func (r *InboxRepository) UpdateAsProcessed(ctx context.Context, ext RepoExtension, messageID uuid.UUID) error {
	if ext == nil {
		ext = r.db
	}

	const query = `
        UPDATE messages.inbox_messages
        SET processed = true, processed_at = NOW()
        WHERE id = $1;
    `

	_, err := ext.Exec(ctx, query, messageID)
	if err != nil {
		return err
	}

	return nil
}

func (r *InboxRepository) SelectUnprocessedBatch(ctx context.Context, ext RepoExtension, batchSize int) ([]model.InboxMessage, error) {
	if ext == nil {
		ext = r.db
	}

	var messages []model.InboxMessage

	const query = `
        SELECT id, topic, payload, created_at, processed, processed_at
        FROM messages.inbox_messages
        WHERE processed = false
        ORDER BY created_at
        LIMIT $1;
    `

	rows, err := ext.Query(ctx, query, batchSize)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var message model.InboxMessage
		if err := rows.Scan(
			&message.ID,
			&message.Topic,
			&message.Payload,
			&message.CreatedAt,
			&message.Processed,
			&message.ProcessedAt,
		); err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	return messages, nil
}
