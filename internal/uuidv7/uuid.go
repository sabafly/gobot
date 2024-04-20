package uuidv7

import (
	"github.com/google/uuid"
)

// New returns a new uuid.UUID v7
func New() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}
