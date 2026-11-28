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

package federation

import (
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
)

const subscriberChanSize = 128

// EventHub buffers sanitized native global-board events in memory and fans them out to SSE listeners.
type EventHub struct {
	epoch      string
	bufferSize int

	mu     sync.RWMutex
	seq    uint64
	buf    []WireEvent
	nextID uint64
	subs   map[uint64]*hubSubscriber
}

type hubSubscriber struct {
	ch     chan WireEvent
	closed bool
}

var _ bbs.EventSink = (*EventHub)(nil)

// NewEventHub constructs a bounded ring-buffer EventHub with a fresh startup epoch.
func NewEventHub(bufferSize int) *EventHub {
	if bufferSize <= 0 {
		bufferSize = DefaultEventBufferSize
	}
	return &EventHub{
		epoch:      uuid.NewString(),
		bufferSize: bufferSize,
		buf:        make([]WireEvent, 0, min(bufferSize, 256)),
		subs:       make(map[uint64]*hubSubscriber),
	}
}

// Epoch returns the unique server startup epoch identifying sequence numbers on this instance run.
func (h *EventHub) Epoch() string {
	return h.epoch
}

// LatestSeq returns the most recently assigned event sequence number.
func (h *EventHub) LatestSeq() uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.seq
}

// Publish records ev in the ring buffer and dispatches it non-blockingly to connected SSE listeners.
// Slow listeners whose buffers overflow have their channels closed so they reconnect and resync cleanly.
func (h *EventHub) Publish(ev bbs.Event) {
	now := ev.OccurredAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	h.mu.Lock()
	h.seq++
	we := WireEvent{
		Seq:          h.seq,
		Epoch:        h.epoch,
		Type:         ev.Type,
		BoardID:      ev.BoardID,
		ThreadID:     ev.ThreadID,
		PostID:       ev.PostID,
		BackwardRefs: slices.Clone(ev.BackwardRefs),
		HitMaxRes:    ev.HitMaxRes,
		OriginPeer:   ev.OriginPeer,
		OccurredAt:   now,
	}
	if ev.Board != nil {
		b := *ev.Board
		we.Board = &b
	}
	if ev.Thread != nil {
		t := *ev.Thread
		we.Thread = &t
	}
	if ev.Post != nil {
		p := *ev.Post
		we.Post = &p
	}

	if len(h.buf) >= h.bufferSize {
		copy(h.buf, h.buf[1:])
		h.buf[len(h.buf)-1] = we
	} else {
		h.buf = append(h.buf, we)
	}

	for id, sub := range h.subs {
		if sub.closed {
			continue
		}
		select {
		case sub.ch <- we:
		default:
			sub.closed = true
			close(sub.ch)
			delete(h.subs, id)
		}
	}
	h.mu.Unlock()
}

// Subscribe returns buffered events strictly after afterSeq, a live stream channel,
// and resyncRequired=true if clientEpoch mismatches or the ring buffer no longer covers afterSeq.
func (h *EventHub) Subscribe(afterSeq uint64, clientEpoch string) (
	epoch string,
	latestSeq uint64,
	replay []WireEvent,
	live <-chan WireEvent,
	resyncRequired bool,
	unsubscribe func(),
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	epoch = h.epoch
	latestSeq = h.seq

	if clientEpoch != "" && clientEpoch != h.epoch {
		resyncRequired = true
	} else if afterSeq > 0 {
		if afterSeq > h.seq {
			resyncRequired = true
		} else if len(h.buf) > 0 {
			oldest := h.buf[0].Seq
			if afterSeq+1 < oldest {
				resyncRequired = true
			}
		} else if h.seq > 0 {
			resyncRequired = true
		}
	}

	if !resyncRequired && len(h.buf) > 0 {
		for _, we := range h.buf {
			if we.Seq > afterSeq {
				replay = append(replay, we)
			}
		}
	}

	h.nextID++
	id := h.nextID
	sub := &hubSubscriber{ch: make(chan WireEvent, subscriberChanSize)}
	h.subs[id] = sub

	unsubscribe = func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if cur, ok := h.subs[id]; ok {
			delete(h.subs, id)
			if !cur.closed {
				cur.closed = true
				close(cur.ch)
			}
		}
	}
	return epoch, latestSeq, replay, sub.ch, resyncRequired, unsubscribe
}
