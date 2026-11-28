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
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
)

const (
	nameCachePositiveTTL = 10 * time.Minute
	nameCacheNegativeTTL = 2 * time.Minute
	nameCacheMaxEntries  = 4096
	nameResolveMaxREST   = 10
)

type cachedNameEntry struct {
	name      string
	found     bool
	expiresAt time.Time
}

type nameResolverCache struct {
	mu    sync.RWMutex
	users map[snowflake.ID]cachedNameEntry
}

func newNameResolverCache() *nameResolverCache {
	return &nameResolverCache{
		users: make(map[snowflake.ID]cachedNameEntry),
	}
}

func (c *nameResolverCache) get(id snowflake.ID, now time.Time) (name string, found bool, hit bool) {
	if c == nil {
		return "", false, false
	}
	c.mu.RLock()
	entry, ok := c.users[id]
	c.mu.RUnlock()
	if !ok || now.After(entry.expiresAt) {
		return "", false, false
	}
	return entry.name, entry.found, true
}

func (c *nameResolverCache) set(id snowflake.ID, name string, found bool, now time.Time) {
	if c == nil || id == 0 {
		return
	}
	ttl := nameCachePositiveTTL
	if !found {
		ttl = nameCacheNegativeTTL
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.users) >= nameCacheMaxEntries {
		for k, v := range c.users {
			if now.After(v.expiresAt) {
				delete(c.users, k)
			}
		}
		if len(c.users) >= nameCacheMaxEntries {
			for k := range c.users {
				delete(c.users, k)
				break
			}
		}
	}
	c.users[id] = cachedNameEntry{
		name:      name,
		found:     found,
		expiresAt: now.Add(ttl),
	}
}

func formatDiscordUserName(displayName, username string) string {
	displayName = strings.TrimSpace(displayName)
	username = strings.TrimSpace(username)
	switch {
	case displayName != "" && username != "" && displayName != username:
		return fmt.Sprintf("%s (@%s)", displayName, username)
	case username != "":
		return "@" + username
	default:
		return displayName
	}
}

func dedupSnowflakes(ids []snowflake.ID) []snowflake.ID {
	seen := make(map[snowflake.ID]struct{}, len(ids))
	out := make([]snowflake.ID, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func dedupUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *Server) resolveGuildNames(ctx context.Context, ids []snowflake.ID) map[snowflake.ID]string {
	unique := dedupSnowflakes(ids)
	out := make(map[snowflake.ID]string, len(unique))
	if len(unique) == 0 {
		return out
	}

	var missing []snowflake.ID
	for _, id := range unique {
		if s.client != nil && s.client.Caches != nil {
			if g, ok := s.client.Caches.Guild(id); ok && strings.TrimSpace(g.Name) != "" {
				out[id] = strings.TrimSpace(g.Name)
				continue
			}
		}
		missing = append(missing, id)
	}

	if len(missing) > 0 && s.db != nil && s.db.DB != nil {
		gdb := s.db.DB.WithContext(ctx)
		if gdb.Migrator().HasTable(&models.Guild{}) {
			var rows []struct {
				ID   snowflake.ID
				Name string
			}
			if err := gdb.Model(&models.Guild{}).Select("id", "name").Where("id IN ?", missing).Find(&rows).Error; err == nil {
				for _, r := range rows {
					if name := strings.TrimSpace(r.Name); name != "" {
						out[r.ID] = name
					}
				}
			}
		}
	}
	return out
}

func (s *Server) resolveChannelNames(ids []snowflake.ID) map[snowflake.ID]string {
	unique := dedupSnowflakes(ids)
	out := make(map[snowflake.ID]string, len(unique))
	if len(unique) == 0 || s.client == nil || s.client.Caches == nil {
		return out
	}
	for _, id := range unique {
		if ch, ok := s.client.Caches.Channel(id); ok {
			if name := strings.TrimSpace(ch.Name()); name != "" {
				out[id] = name
				continue
			}
		}
		if th, ok := s.client.Caches.GuildThread(id); ok {
			if name := strings.TrimSpace(th.Name()); name != "" {
				out[id] = name
			}
		}
	}
	return out
}

func (s *Server) resolveUserNames(ctx context.Context, ids []snowflake.ID, guildHints []snowflake.ID, allowREST bool) map[snowflake.ID]string {
	unique := dedupSnowflakes(ids)
	out := make(map[snowflake.ID]string, len(unique))
	if len(unique) == 0 {
		return out
	}
	hints := dedupSnowflakes(guildHints)
	now := time.Now()
	negCached := make(map[snowflake.ID]bool)

	var missing []snowflake.ID
	for _, uid := range unique {
		if name, found, hit := s.nameCache.get(uid, now); hit {
			if found && name != "" {
				out[uid] = name
			} else {
				negCached[uid] = true
			}
			continue
		}
		if s.client != nil && s.client.Caches != nil {
			if self, ok := s.client.Caches.SelfUser(); ok && self.ID == uid {
				if formatted := formatDiscordUserName(self.EffectiveName(), self.Username); formatted != "" {
					out[uid] = formatted
					s.nameCache.set(uid, formatted, true, now)
					continue
				}
			}
			resolvedFromMember := false
			for _, gid := range hints {
				if mem, ok := s.client.Caches.Member(gid, uid); ok {
					if formatted := formatDiscordUserName(mem.EffectiveName(), mem.User.Username); formatted != "" {
						out[uid] = formatted
						s.nameCache.set(uid, formatted, true, now)
						resolvedFromMember = true
						break
					}
				}
			}
			if resolvedFromMember {
				continue
			}
		}
		missing = append(missing, uid)
	}

	if len(missing) > 0 && s.db != nil && s.db.DB != nil {
		gdb := s.db.DB.WithContext(ctx)
		if gdb.Migrator().HasTable(&models.AdminSession{}) {
			var sessRows []struct {
				UserID      snowflake.ID
				Username    string
				DisplayName string
			}
			if err := gdb.Model(&models.AdminSession{}).
				Select("user_id", "username", "display_name").
				Where("user_id IN ?", missing).
				Order("last_used_at DESC").
				Find(&sessRows).Error; err == nil {
				for _, r := range sessRows {
					if _, already := out[r.UserID]; already {
						continue
					}
					if formatted := formatDiscordUserName(r.DisplayName, r.Username); formatted != "" {
						out[r.UserID] = formatted
						s.nameCache.set(r.UserID, formatted, true, now)
					}
				}
			}
		}

		stillMissing := make([]snowflake.ID, 0, len(missing))
		for _, uid := range missing {
			if _, ok := out[uid]; !ok {
				stillMissing = append(stillMissing, uid)
			}
		}
		missing = stillMissing

		if len(missing) > 0 && gdb.Migrator().HasTable(&models.User{}) {
			var userRows []struct {
				ID   snowflake.ID
				Name string
			}
			if err := gdb.Model(&models.User{}).Select("id", "name").Where("id IN ?", missing).Find(&userRows).Error; err == nil {
				for _, r := range userRows {
					if name := strings.TrimSpace(r.Name); name != "" {
						out[r.ID] = name
						s.nameCache.set(r.ID, name, true, now)
					}
				}
			}
		}
	}

	if allowREST && s.client != nil && s.client.Rest != nil {
		restCalls := 0
		for _, uid := range unique {
			if _, ok := out[uid]; ok || negCached[uid] {
				continue
			}
			if restCalls >= nameResolveMaxREST {
				break
			}
			restCalls++
			u, err := s.client.Rest.GetUser(uid)
			if err != nil || u == nil {
				s.nameCache.set(uid, "", false, now)
				continue
			}
			if formatted := formatDiscordUserName(u.EffectiveName(), u.Username); formatted != "" {
				out[uid] = formatted
				s.nameCache.set(uid, formatted, true, now)
			} else {
				s.nameCache.set(uid, "", false, now)
			}
		}
	}
	return out
}

func (s *Server) resolveBBSNames(
	ctx context.Context,
	catIDs, boardIDs, threadIDs []uuid.UUID,
) (cats map[uuid.UUID]string, boards map[uuid.UUID]string, threads map[uuid.UUID]string) {
	uCats := dedupUUIDs(catIDs)
	uBoards := dedupUUIDs(boardIDs)
	uThreads := dedupUUIDs(threadIDs)

	cats = make(map[uuid.UUID]string, len(uCats))
	boards = make(map[uuid.UUID]string, len(uBoards))
	threads = make(map[uuid.UUID]string, len(uThreads))

	if s.db == nil || s.db.DB == nil {
		return cats, boards, threads
	}
	gdb := s.db.DB.WithContext(ctx)

	if len(uCats) > 0 && gdb.Migrator().HasTable(&gormstore.BBSCategory{}) {
		var rows []struct {
			ID   uuid.UUID
			Name string
		}
		if err := gdb.Model(&gormstore.BBSCategory{}).Select("id", "name").Where("id IN ?", uCats).Find(&rows).Error; err == nil {
			for _, r := range rows {
				if name := strings.TrimSpace(r.Name); name != "" {
					cats[r.ID] = name
				}
			}
		}
	}
	if len(uBoards) > 0 && gdb.Migrator().HasTable(&gormstore.BBSBoard{}) {
		var rows []struct {
			ID   uuid.UUID
			Name string
		}
		if err := gdb.Model(&gormstore.BBSBoard{}).Select("id", "name").Where("id IN ?", uBoards).Find(&rows).Error; err == nil {
			for _, r := range rows {
				if name := strings.TrimSpace(r.Name); name != "" {
					boards[r.ID] = name
				}
			}
		}
	}
	if len(uThreads) > 0 && gdb.Migrator().HasTable(&gormstore.BBSThread{}) {
		var rows []struct {
			ID    uuid.UUID
			Title string
		}
		if err := gdb.Model(&gormstore.BBSThread{}).Select("id", "title").Where("id IN ?", uThreads).Find(&rows).Error; err == nil {
			for _, r := range rows {
				if title := strings.TrimSpace(r.Title); title != "" {
					threads[r.ID] = title
				}
			}
		}
	}

	if s.bbsSvc != nil {
		for _, id := range uBoards {
			if _, ok := boards[id]; ok {
				continue
			}
			if b, err := s.bbsSvc.GetBoard(ctx, id); err == nil && b != nil && strings.TrimSpace(b.Name) != "" {
				boards[id] = strings.TrimSpace(b.Name)
			}
		}
		for _, id := range uThreads {
			if _, ok := threads[id]; ok {
				continue
			}
			if th, err := s.bbsSvc.GetThread(ctx, id); err == nil && th != nil && strings.TrimSpace(th.Title) != "" {
				threads[id] = strings.TrimSpace(th.Title)
			}
		}
	}
	return cats, boards, threads
}
