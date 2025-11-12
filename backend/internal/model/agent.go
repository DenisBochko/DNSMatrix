package model

import (
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Region    string    `db:"region" json:"region"`
	ASN       int       `db:"asn" json:"asn"`
	Online    bool      `db:"online" json:"online"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}
