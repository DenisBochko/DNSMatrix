package model

import (
	"time"

	"github.com/google/uuid"
)

type InboxMessage struct {
	ID          uuid.UUID  `db:"id"`
	Topic       string     `db:"topic"`
	Payload     []byte     `db:"payload"`
	CreatedAt   time.Time  `db:"created_at"`
	Processed   bool       `db:"processed"`
	ProcessedAt *time.Time `db:"processed_at"`
}
