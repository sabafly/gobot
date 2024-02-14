package uuid

import (
	"github.com/google/uuid"
)

// New returns a new UUID v7
func New() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}
