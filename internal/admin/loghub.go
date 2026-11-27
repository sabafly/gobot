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

package admin

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	defaultLogBufferSize = 2000
	maxLogListLimit      = 2000
	defaultLogListLimit  = 200
)

// LogHub stores recent slog records in a fixed-capacity ring buffer and broadcasts new records to subscribers.
type LogHub struct {
	mu              sync.RWMutex
	cap             int
	entries         []LogEntry
	nextSeq         uint64
	subs            map[uint64]chan LogEntry
	nextSub         uint64
	captureMode     string
	captureLevel    slog.Level
	captureExplicit bool
	downstream      slog.Handler
}

// NewLogHub constructs a LogHub with capacity entries retained in memory.
func NewLogHub(capacity int) *LogHub {
	if capacity <= 0 {
		capacity = defaultLogBufferSize
	}
	return &LogHub{
		cap:         capacity,
		entries:     make([]LogEntry, 0, min(capacity, 256)),
		nextSeq:     1,
		subs:        make(map[uint64]chan LogEntry),
		captureMode: "auto",
	}
}

// ParseLogCaptureLevel validates a runtime or config LogHub capture level ("auto", "debug", "info", "warn", "error").
func ParseLogCaptureLevel(raw string) (mode string, level slog.Level, explicit bool, err error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "auto":
		return "auto", 0, false, nil
	case "debug":
		return "debug", slog.LevelDebug, true, nil
	case "info":
		return "info", slog.LevelInfo, true, nil
	case "warn", "warning":
		return "warn", slog.LevelWarn, true, nil
	case "error", "err":
		return "error", slog.LevelError, true, nil
	default:
		return "", 0, false, fmt.Errorf("invalid log level %q (expected auto, debug, info, warn, or error)", raw)
	}
}

// SetDownstreamHandler records the primary slog handler used to evaluate effective level in "auto" mode.
func (h *LogHub) SetDownstreamHandler(down slog.Handler) {
	if h == nil || down == nil {
		return
	}
	h.mu.Lock()
	h.downstream = down
	h.mu.Unlock()
}

// SetCaptureLevel configures the minimum level mirrored into this LogHub (or "auto" to follow downstream).
func (h *LogHub) SetCaptureLevel(raw string) (LogLevelResponse, error) {
	mode, lvl, explicit, err := ParseLogCaptureLevel(raw)
	if err != nil {
		return LogLevelResponse{}, err
	}
	if h == nil {
		eff := "INFO"
		if explicit {
			eff = formatSlogLevelUpper(lvl)
		}
		return LogLevelResponse{Mode: mode, EffectiveLevel: eff}, nil
	}
	h.mu.Lock()
	h.captureMode = mode
	h.captureLevel = lvl
	h.captureExplicit = explicit
	down := h.downstream
	h.mu.Unlock()

	return h.resolveCaptureLevel(context.Background(), mode, lvl, explicit, down), nil
}

// CaptureLevel returns the active capture mode and effective minimum log level for this LogHub.
func (h *LogHub) CaptureLevel(ctx context.Context) LogLevelResponse {
	if h == nil {
		return LogLevelResponse{Mode: "auto", EffectiveLevel: "INFO"}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	h.mu.RLock()
	mode := h.captureMode
	lvl := h.captureLevel
	explicit := h.captureExplicit
	down := h.downstream
	h.mu.RUnlock()
	if mode == "" {
		mode = "auto"
	}
	return h.resolveCaptureLevel(ctx, mode, lvl, explicit, down)
}

func (h *LogHub) resolveCaptureLevel(ctx context.Context, mode string, lvl slog.Level, explicit bool, down slog.Handler) LogLevelResponse {
	if explicit {
		return LogLevelResponse{
			Mode:           mode,
			EffectiveLevel: formatSlogLevelUpper(lvl),
		}
	}
	eff := "INFO"
	if down != nil {
		switch {
		case down.Enabled(ctx, slog.LevelDebug):
			eff = "DEBUG"
		case down.Enabled(ctx, slog.LevelInfo):
			eff = "INFO"
		case down.Enabled(ctx, slog.LevelWarn):
			eff = "WARN"
		case down.Enabled(ctx, slog.LevelError):
			eff = "ERROR"
		default:
			eff = "OFF"
		}
	}
	return LogLevelResponse{
		Mode:           mode,
		EffectiveLevel: eff,
	}
}

func formatSlogLevelUpper(lvl slog.Level) string {
	switch {
	case lvl <= slog.LevelDebug:
		return "DEBUG"
	case lvl <= slog.LevelInfo:
		return "INFO"
	case lvl <= slog.LevelWarn:
		return "WARN"
	default:
		return "ERROR"
	}
}

// ShouldCapture reports whether a record at level should be stored in this LogHub.
func (h *LogHub) ShouldCapture(ctx context.Context, downstream slog.Handler, level slog.Level) bool {
	if h == nil {
		return false
	}
	h.mu.RLock()
	explicit := h.captureExplicit
	capLevel := h.captureLevel
	fallbackDown := h.downstream
	h.mu.RUnlock()

	if explicit {
		return level >= capLevel
	}
	down := downstream
	if down == nil {
		down = fallbackDown
	}
	if down == nil {
		return level >= slog.LevelInfo
	}
	return down.Enabled(ctx, level)
}

// SetCapacity resizes the ring buffer while preserving the most recent entries up to capacity.
func (h *LogHub) SetCapacity(capacity int) {
	if h == nil || capacity <= 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cap = capacity
	if len(h.entries) > capacity {
		h.entries = append([]LogEntry(nil), h.entries[len(h.entries)-capacity:]...)
	}
}

// Stats returns the current buffered entry count and configured capacity.
func (h *LogHub) Stats() (count int, capacity int) {
	if h == nil {
		return 0, 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.entries), h.cap
}

// Append records entry into the ring buffer and delivers it non-blockingly to active subscribers.
func (h *LogHub) Append(entry LogEntry) LogEntry {
	if h == nil {
		return entry
	}
	h.mu.Lock()
	if entry.Time.IsZero() {
		entry.Time = time.Now().UTC()
	}
	entry.Seq = h.nextSeq
	h.nextSeq++
	if len(h.entries) >= h.cap {
		copy(h.entries, h.entries[1:])
		h.entries[len(h.entries)-1] = entry
	} else {
		h.entries = append(h.entries, entry)
	}
	for _, ch := range h.subs {
		select {
		case ch <- entry:
		default:
			// Never block slog emitters because of slow stream readers.
		}
	}
	h.mu.Unlock()
	return entry
}

// List returns buffered entries matching filter together with the next sequence number.
func (h *LogHub) List(filter LogFilter) ([]LogEntry, uint64) {
	if h == nil {
		return nil, 1
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultLogListLimit
	}
	if limit > maxLogListLimit {
		limit = maxLogListLimit
	}
	minLevel, hasMinLevel := parseSlogLevel(filter.MinLevel)
	q := strings.ToLower(strings.TrimSpace(filter.Query))

	h.mu.RLock()
	defer h.mu.RUnlock()

	var matched []LogEntry
	for _, e := range h.entries {
		if e.Seq <= filter.AfterSeq {
			continue
		}
		if hasMinLevel {
			el, ok := parseSlogLevel(e.Level)
			if !ok || el < minLevel {
				continue
			}
		}
		if q != "" && !MatchLogQuery(e, q) {
			continue
		}
		matched = append(matched, e)
	}
	nextSeq := h.nextSeq
	if len(matched) > limit {
		if filter.AfterSeq > 0 {
			matched = matched[:limit]
			nextSeq = matched[len(matched)-1].Seq + 1
		} else {
			matched = matched[len(matched)-limit:]
		}
	}
	return matched, nextSeq
}

// MatchLogEntry reports whether entry satisfies minLevel and query filters.
func MatchLogEntry(entry LogEntry, minLevelStr string, query string) bool {
	if lvl, ok := parseSlogLevel(minLevelStr); ok {
		el, okEntry := parseSlogLevel(entry.Level)
		if !okEntry || el < lvl {
			return false
		}
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	return MatchLogQuery(entry, q)
}

func MatchLogQuery(entry LogEntry, lowerQuery string) bool {
	if strings.Contains(strings.ToLower(entry.Message), lowerQuery) ||
		strings.Contains(strings.ToLower(entry.Source), lowerQuery) {
		return true
	}
	for k, v := range entry.Attrs {
		if strings.Contains(strings.ToLower(k), lowerQuery) ||
			strings.Contains(strings.ToLower(fmt.Sprint(v)), lowerQuery) {
			return true
		}
	}
	return false
}

// Subscribe registers a live log stream listener and returns the channel plus an unsubscribe func.
func (h *LogHub) Subscribe(bufSize int) (<-chan LogEntry, func()) {
	if bufSize <= 0 {
		bufSize = 128
	}
	ch := make(chan LogEntry, bufSize)
	h.mu.Lock()
	id := h.nextSub
	h.nextSub++
	h.subs[id] = ch
	h.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			if cur, ok := h.subs[id]; ok {
				delete(h.subs, id)
				close(cur)
			}
			h.mu.Unlock()
		})
	}
}

func parseSlogLevel(raw string) (slog.Level, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "ALL":
		return 0, false
	case "DEBUG":
		return slog.LevelDebug, true
	case "INFO":
		return slog.LevelInfo, true
	case "WARN", "WARNING":
		return slog.LevelWarn, true
	case "ERROR", "ERR":
		return slog.LevelError, true
	default:
		return 0, false
	}
}

// TeeHandler wraps a downstream slog.Handler and copies structured log entries into a LogHub.
type TeeHandler struct {
	downstream slog.Handler
	hub        *LogHub
	attrs      []slog.Attr
	groups     []string
}

// NewTeeHandler returns an slog.Handler that forwards to downstream and mirrors records into hub.
func NewTeeHandler(downstream slog.Handler, hub *LogHub) slog.Handler {
	if hub != nil && downstream != nil {
		hub.SetDownstreamHandler(downstream)
	}
	return &TeeHandler{
		downstream: downstream,
		hub:        hub,
	}
}

func (t *TeeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	downOK := t.downstream != nil && t.downstream.Enabled(ctx, level)
	hubOK := t.hub != nil && t.hub.ShouldCapture(ctx, t.downstream, level)
	return downOK || hubOK
}

func (t *TeeHandler) Handle(ctx context.Context, r slog.Record) error {
	var err error
	if t.downstream != nil && t.downstream.Enabled(ctx, r.Level) {
		err = t.downstream.Handle(ctx, r)
	}
	if t.hub != nil && t.hub.ShouldCapture(ctx, t.downstream, r.Level) {
		attrs := make(map[string]any, len(t.attrs)+r.NumAttrs())
		prefix := strings.Join(t.groups, ".")
		for _, a := range t.attrs {
			collectAttr(attrs, prefix, a)
		}
		r.Attrs(func(a slog.Attr) bool {
			collectAttr(attrs, prefix, a)
			return true
		})
		if len(attrs) == 0 {
			attrs = nil
		}
		var src string
		if r.PC != 0 {
			fs := runtime.CallersFrames([]uintptr{r.PC})
			f, _ := fs.Next()
			if f.File != "" {
				src = fmt.Sprintf("%s:%d", filepath.Base(f.File), f.Line)
			}
		}
		t.hub.Append(LogEntry{
			Time:    r.Time.UTC(),
			Level:   r.Level.String(),
			Message: r.Message,
			Source:  src,
			Attrs:   attrs,
		})
	}
	return err
}

func (t *TeeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var down slog.Handler
	if t.downstream != nil {
		down = t.downstream.WithAttrs(attrs)
	}
	copiedAttrs := append([]slog.Attr(nil), t.attrs...)
	if len(t.groups) > 0 {
		prefix := strings.Join(t.groups, ".")
		for _, a := range attrs {
			if a.Key != "" {
				a.Key = prefix + "." + a.Key
			}
			copiedAttrs = append(copiedAttrs, a)
		}
	} else {
		copiedAttrs = append(copiedAttrs, attrs...)
	}
	return &TeeHandler{
		downstream: down,
		hub:        t.hub,
		attrs:      copiedAttrs,
		groups:     append([]string(nil), t.groups...),
	}
}

func (t *TeeHandler) WithGroup(name string) slog.Handler {
	var down slog.Handler
	if t.downstream != nil {
		down = t.downstream.WithGroup(name)
	}
	groups := append([]string(nil), t.groups...)
	if name != "" {
		groups = append(groups, name)
	}
	return &TeeHandler{
		downstream: down,
		hub:        t.hub,
		attrs:      append([]slog.Attr(nil), t.attrs...),
		groups:     groups,
	}
}

func collectAttr(out map[string]any, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	key := a.Key
	if prefix != "" && !strings.HasPrefix(key, prefix+".") {
		if key != "" {
			key = prefix + "." + key
		} else {
			key = prefix
		}
	}
	if a.Value.Kind() == slog.KindGroup {
		for _, sub := range a.Value.Group() {
			collectAttr(out, key, sub)
		}
		return
	}
	if key == "" {
		return
	}
	if isSensitiveAttrKey(key) {
		out[key] = "[REDACTED]"
		return
	}
	switch a.Value.Kind() {
	case slog.KindAny:
		if err, ok := a.Value.Any().(error); ok {
			out[key] = err.Error()
		} else {
			out[key] = fmt.Sprint(a.Value.Any())
		}
	default:
		out[key] = a.Value.Any()
	}
}

func isSensitiveAttrKey(key string) bool {
	lk := strings.ToLower(key)
	for _, needle := range []string{"token", "secret", "password", "passwd", "dsn", "authorization"} {
		if strings.Contains(lk, needle) {
			return true
		}
	}
	return false
}
