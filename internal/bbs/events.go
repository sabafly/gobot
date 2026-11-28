/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package bbs

import (
	"time"

	"github.com/google/uuid"
)

// EventType identifies a global-board state transition propagated to federation subscribers.
type EventType string

const (
	EventBoardUpdated   EventType = "board.updated"
	EventBoardDeleted   EventType = "board.deleted"
	EventThreadCreated  EventType = "thread.created"
	EventPostCreated    EventType = "post.created"
	EventPostAboned     EventType = "post.aboned"
	EventThreadArchived EventType = "thread.archived"
)

// Event represents a sanitized global-board domain event suitable for cross-instance fanout.
type Event struct {
	Type         EventType
	BoardID      uuid.UUID
	ThreadID     uuid.UUID
	PostID       uuid.UUID
	Board        *Board
	Thread       *Thread
	Post         *Post
	BackwardRefs []uint32
	HitMaxRes    bool
	OriginPeer   string
	OccurredAt   time.Time
}

// EventSink receives native global-board domain events from Service.
type EventSink interface {
	Publish(ev Event)
}
