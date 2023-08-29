package db

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type TicketEdit struct {
	TicketID  uuid.UUID    `json:"ticket_id"`
	OwnerID   snowflake.ID `json:"owner_id"`
	CreatedAt time.Time    `json:"created_at"`
}
