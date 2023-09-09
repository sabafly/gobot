package db

import "github.com/google/uuid"

type TicketTemplate struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	About   string    `json:"about"`
	Title   string    `json:"title"`
	Content string    `json:"content"`
}
