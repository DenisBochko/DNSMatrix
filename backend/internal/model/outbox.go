package model

import (
	"time"

	"github.com/google/uuid"
)

type OutboxMessage struct {
	ID        uuid.UUID  `db:"id"`
	Topic     string     `db:"topic"`
	Payload   []byte     `db:"payload"`
	CreatedAt time.Time  `db:"created_at"`
	Sent      bool       `db:"sent"`
	SentAt    *time.Time `db:"sent_at"`
}
