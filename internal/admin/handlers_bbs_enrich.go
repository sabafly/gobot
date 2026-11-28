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

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
)

func (s *Server) enrichBBSCategories(ctx context.Context, cats []bbs.Category) []BBSCategoryDTO {
	guildIDs := make([]snowflake.ID, 0, len(cats))
	for _, c := range cats {
		if c.GuildID != nil {
			guildIDs = append(guildIDs, *c.GuildID)
		}
	}
	guildNames := s.resolveGuildNames(ctx, guildIDs)
	out := make([]BBSCategoryDTO, 0, len(cats))
	for _, c := range cats {
		dto := toBBSCategoryDTO(c)
		if c.GuildID != nil {
			dto.GuildName = guildNames[*c.GuildID]
		}
		out = append(out, dto)
	}
	return out
}

func (s *Server) enrichBBSBoards(ctx context.Context, boards []bbs.Board) []BBSBoardDTO {
	catIDs := make([]uuid.UUID, 0, len(boards))
	guildIDs := make([]snowflake.ID, 0, len(boards))
	for _, b := range boards {
		catIDs = append(catIDs, b.CategoryID)
		if b.GuildID != nil {
			guildIDs = append(guildIDs, *b.GuildID)
		}
	}
	catNames, _, _ := s.resolveBBSNames(ctx, catIDs, nil, nil)
	guildNames := s.resolveGuildNames(ctx, guildIDs)

	out := make([]BBSBoardDTO, 0, len(boards))
	for _, b := range boards {
		dto := toBBSBoardDTO(b)
		dto.CategoryName = catNames[b.CategoryID]
		if b.GuildID != nil {
			dto.GuildName = guildNames[*b.GuildID]
		}
		out = append(out, dto)
	}
	return out
}

func (s *Server) enrichBBSBoardMounts(ctx context.Context, mounts []bbs.BoardMount) []BBSBoardMountDTO {
	guildIDs := make([]snowflake.ID, 0, len(mounts))
	channelIDs := make([]snowflake.ID, 0, len(mounts))
	for _, m := range mounts {
		guildIDs = append(guildIDs, m.GuildID)
		channelIDs = append(channelIDs, m.ChannelID)
	}
	guildNames := s.resolveGuildNames(ctx, guildIDs)
	channelNames := s.resolveChannelNames(channelIDs)

	out := make([]BBSBoardMountDTO, 0, len(mounts))
	for _, m := range mounts {
		dto := toBBSBoardMountDTO(m)
		dto.GuildName = guildNames[m.GuildID]
		dto.ChannelName = channelNames[m.ChannelID]
		out = append(out, dto)
	}
	return out
}

func (s *Server) enrichBBSReports(ctx context.Context, reps []bbs.Report) []BBSReportDTO {
	guildIDs := make([]snowflake.ID, 0, len(reps))
	userIDs := make([]snowflake.ID, 0, len(reps)*2)
	boardIDs := make([]uuid.UUID, 0, len(reps))
	threadIDs := make([]uuid.UUID, 0, len(reps))
	for _, r := range reps {
		if r.GuildID != 0 {
			guildIDs = append(guildIDs, r.GuildID)
		}
		if r.ReporterUserID != 0 {
			userIDs = append(userIDs, r.ReporterUserID)
		}
		if r.HandledBy != nil && *r.HandledBy != 0 {
			userIDs = append(userIDs, *r.HandledBy)
		}
		boardIDs = append(boardIDs, r.BoardID)
		threadIDs = append(threadIDs, r.ThreadID)
	}
	guildNames := s.resolveGuildNames(ctx, guildIDs)
	userNames := s.resolveUserNames(ctx, userIDs, guildIDs, false)
	_, boardNames, threadTitles := s.resolveBBSNames(ctx, nil, boardIDs, threadIDs)

	out := make([]BBSReportDTO, 0, len(reps))
	for _, r := range reps {
		dto := toBBSReportDTO(r)
		dto.GuildName = guildNames[r.GuildID]
		dto.BoardName = boardNames[r.BoardID]
		dto.ThreadTitle = threadTitles[r.ThreadID]
		dto.ReporterName = userNames[r.ReporterUserID]
		if r.HandledBy != nil {
			dto.HandledByName = userNames[*r.HandledBy]
		}
		out = append(out, dto)
	}
	return out
}

func (s *Server) enrichBBSBans(ctx context.Context, bans []bbs.Ban, includeTarget bool) []BBSBanDTO {
	guildIDs := make([]snowflake.ID, 0, len(bans))
	userIDs := make([]snowflake.ID, 0, len(bans)*2)
	for _, b := range bans {
		if b.GuildID != nil && *b.GuildID != 0 {
			guildIDs = append(guildIDs, *b.GuildID)
		}
		if b.CreatedBy != 0 {
			userIDs = append(userIDs, b.CreatedBy)
		}
		if includeTarget && b.TargetUserID != nil && *b.TargetUserID != 0 {
			userIDs = append(userIDs, *b.TargetUserID)
		}
	}
	guildNames := s.resolveGuildNames(ctx, guildIDs)
	userNames := s.resolveUserNames(ctx, userIDs, guildIDs, false)

	out := make([]BBSBanDTO, 0, len(bans))
	for _, b := range bans {
		dto := toBBSBanDTO(b)
		if b.GuildID != nil {
			dto.GuildName = guildNames[*b.GuildID]
		}
		dto.CreatedByName = userNames[b.CreatedBy]
		if includeTarget && b.TargetUserID != nil {
			dto.TargetUserName = userNames[*b.TargetUserID]
		} else {
			dto.TargetUserID = nil
			dto.TargetUserName = ""
		}
		out = append(out, dto)
	}
	return out
}

func (s *Server) enrichBBSCaps(ctx context.Context, defs []bbs.CapDefinition) []BBSCapDTO {
	var allUIDs []snowflake.ID
	for _, d := range defs {
		allUIDs = append(allUIDs, d.Users...)
	}
	userNames := s.resolveUserNames(ctx, allUIDs, nil, true)
	out := make([]BBSCapDTO, 0, len(defs))
	for _, d := range defs {
		dto := toBBSCapDTO(d)
		if len(d.Users) > 0 {
			m := make(map[string]string, len(d.Users))
			for _, u := range d.Users {
				if name := userNames[u]; name != "" {
					m[u.String()] = name
				}
			}
			if len(m) > 0 {
				dto.UserNames = m
			}
		}
		out = append(out, dto)
	}
	return out
}
