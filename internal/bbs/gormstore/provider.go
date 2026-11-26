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

package gormstore

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/uuidv7"
)

var _ bbs.Provider = (*Provider)(nil)
var _ bbs.Store = (*Provider)(nil)

// Provider implements both the native bbs.Provider and bbs.Store backed by GORM.
type Provider struct {
	db *gorm.DB
}

// New creates a GORM-backed BBS provider/store.
func New(db *gorm.DB) *Provider {
	return &Provider{db: db}
}

func (p *Provider) Key() string { return bbs.DefaultProviderKey }

func (p *Provider) Capabilities() bbs.ProviderCapabilities {
	return bbs.ProviderCapabilities{
		CanCreateBoard:  true,
		CanCreateThread: true,
		CanCreatePost:   true,
		CanModerate:     true,
	}
}

func (p *Provider) session(ctx context.Context) *gorm.DB {
	return p.db.WithContext(ctx)
}

func mapErr(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return bbs.ErrNotFound
	}
	return err
}

// --- CategoryProvider ---

func (p *Provider) GetCategory(ctx context.Context, id uuid.UUID) (*bbs.Category, error) {
	var row BBSCategory
	if err := p.session(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainCategory(&row), nil
}

func (p *Provider) ListCategories(ctx context.Context, filter bbs.CategoryFilter) ([]bbs.Category, error) {
	q := p.session(ctx).Model(&BBSCategory{}).Order("sort_order ASC, created_at ASC")
	if filter.Scope != nil {
		q = q.Where("scope = ?", string(*filter.Scope))
	}
	if filter.GuildID != nil {
		q = q.Where("guild_id = ?", *filter.GuildID)
	}
	var rows []BBSCategory
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.Category, len(rows))
	for i := range rows {
		out[i] = *toDomainCategory(&rows[i])
	}
	return out, nil
}

func (p *Provider) CreateCategory(ctx context.Context, input bbs.CreateCategoryInput) (*bbs.Category, error) {
	now := time.Now()
	row := BBSCategory{
		ID:                uuidv7.New(),
		Scope:             string(input.Scope),
		GuildID:           input.GuildID,
		ProviderKey:       input.ProviderKey,
		ExternalID:        input.ExternalID,
		Slug:              input.Slug,
		Name:              input.Name,
		Description:       input.Description,
		SortOrder:         input.SortOrder,
		DiscordCategoryID: input.DiscordCategoryID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := p.session(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return toDomainCategory(&row), nil
}

func (p *Provider) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		var boardIDs []uuid.UUID
		if err := tx.Model(&BBSBoard{}).Where("category_id = ?", id).Pluck("id", &boardIDs).Error; err != nil {
			return err
		}
		for _, bid := range boardIDs {
			if err := deleteBoardTx(tx, bid); err != nil && !errors.Is(err, bbs.ErrNotFound) {
				return err
			}
		}
		res := tx.Where("id = ?", id).Delete(&BBSCategory{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return bbs.ErrNotFound
		}
		return nil
	})
}

// --- BoardProvider ---

func (p *Provider) GetBoard(ctx context.Context, id uuid.UUID) (*bbs.Board, error) {
	var row BBSBoard
	if err := p.session(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainBoard(&row), nil
}

func (p *Provider) ListBoards(ctx context.Context, filter bbs.BoardFilter) ([]bbs.Board, error) {
	q := p.session(ctx).Model(&BBSBoard{}).Order("created_at ASC")
	if filter.Scope != nil {
		q = q.Where("scope = ?", string(*filter.Scope))
	}
	if filter.GuildID != nil {
		q = q.Where("guild_id = ?", *filter.GuildID)
	}
	if filter.CategoryID != nil {
		q = q.Where("category_id = ?", *filter.CategoryID)
	}
	var rows []BBSBoard
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.Board, len(rows))
	for i := range rows {
		out[i] = *toDomainBoard(&rows[i])
	}
	return out, nil
}

func (p *Provider) CreateBoard(ctx context.Context, input bbs.CreateBoardInput) (*bbs.Board, error) {
	rules := input.Rules.Normalize()
	now := time.Now()
	row := BBSBoard{
		ID:                 uuidv7.New(),
		CategoryID:         input.CategoryID,
		Scope:              string(input.Scope),
		GuildID:            input.GuildID,
		ProviderKey:        input.ProviderKey,
		ExternalID:         input.ExternalID,
		Slug:               input.Slug,
		Name:               input.Name,
		Description:        input.Description,
		DefaultName:        rules.DefaultName,
		HideID:             rules.HideID,
		SLIPLevel:          int(rules.SLIP),
		MaxRes:             rules.MaxRes,
		DatOchiIdleHours:   rules.DatOchiIdleHours,
		DatOchiLowResHours: rules.DatOchiLowResHours,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := p.session(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return toDomainBoard(&row), nil
}

func (p *Provider) UpdateBoard(ctx context.Context, id uuid.UUID, input bbs.UpdateBoardInput) (*bbs.Board, error) {
	var row BBSBoard
	err := p.session(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Where("id = ?", id)
		if tx.Name() == "mysql" {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := q.First(&row).Error; err != nil {
			return mapErr(err)
		}
		now := time.Now()
		updates := map[string]any{
			"updated_at": now,
		}
		row.UpdatedAt = now
		if input.Name != nil {
			row.Name = *input.Name
			updates["name"] = *input.Name
		}
		if input.Description != nil {
			row.Description = *input.Description
			updates["description"] = *input.Description
		}
		if input.DefaultName != nil {
			row.DefaultName = *input.DefaultName
			updates["default_name"] = *input.DefaultName
		}
		if input.HideID != nil {
			row.HideID = *input.HideID
			updates["hide_id"] = *input.HideID
		}
		if input.SLIP != nil {
			row.SLIPLevel = int(*input.SLIP)
			updates["slip_level"] = int(*input.SLIP)
		}
		if input.Locked != nil {
			row.Locked = *input.Locked
			updates["locked"] = *input.Locked
		}
		return tx.Model(&BBSBoard{}).Where("id = ?", id).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return toDomainBoard(&row), nil
}

func deleteBoardTx(tx *gorm.DB, boardID uuid.UUID) error {
	var threadIDs []uuid.UUID
	if err := tx.Model(&BBSThread{}).Where("board_id = ?", boardID).Pluck("id", &threadIDs).Error; err != nil {
		return err
	}
	if err := tx.Where("board_id = ?", boardID).Delete(&BBSReport{}).Error; err != nil {
		return err
	}
	if len(threadIDs) > 0 {
		var mirrorIDs []uuid.UUID
		if err := tx.Model(&BBSThreadMirror{}).Where("thread_id IN ?", threadIDs).Pluck("id", &mirrorIDs).Error; err != nil {
			return err
		}
		if len(mirrorIDs) > 0 {
			if err := tx.Where("thread_mirror_id IN ?", mirrorIDs).Delete(&BBSPostMirror{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("thread_id IN ?", threadIDs).Delete(&BBSPostReference{}).Error; err != nil {
			return err
		}
		if err := tx.Where("thread_id IN ?", threadIDs).Delete(&BBSThreadWatch{}).Error; err != nil {
			return err
		}
		if err := tx.Where("thread_id IN ?", threadIDs).Delete(&BBSThreadMirror{}).Error; err != nil {
			return err
		}
		if err := tx.Where("thread_id IN ?", threadIDs).Delete(&BBSPost{}).Error; err != nil {
			return err
		}
		if err := tx.Where("board_id = ?", boardID).Delete(&BBSThread{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("board_id = ?", boardID).Delete(&BBSBoardMount{}).Error; err != nil {
		return err
	}
	res := tx.Where("id = ?", boardID).Delete(&BBSBoard{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return bbs.ErrNotFound
	}
	return nil
}

func (p *Provider) DeleteBoard(ctx context.Context, id uuid.UUID) error {
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		return deleteBoardTx(tx, id)
	})
}

// --- ThreadProvider ---

func (p *Provider) GetThread(ctx context.Context, id uuid.UUID) (*bbs.Thread, error) {
	var row BBSThread
	if err := p.session(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainThread(&row), nil
}

func (p *Provider) ListThreads(ctx context.Context, filter bbs.ThreadFilter) ([]bbs.Thread, error) {
	q := p.session(ctx).Model(&BBSThread{}).Where("board_id = ?", filter.BoardID).Order("bumped_at DESC")
	if filter.Status != nil {
		q = q.Where("status = ?", string(*filter.Status))
	}
	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}
	var rows []BBSThread
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.Thread, len(rows))
	for i := range rows {
		out[i] = *toDomainThread(&rows[i])
	}
	return out, nil
}

func (p *Provider) CreateThread(ctx context.Context, input bbs.CreateThreadRecord) (*bbs.Thread, *bbs.Post, error) {
	rules := input.Rules.Normalize()
	now := time.Now()
	threadID := uuidv7.New()
	postID := uuidv7.New()

	thRow := BBSThread{
		ID:                 threadID,
		BoardID:            input.BoardID,
		ProviderKey:        p.Key(),
		Title:              input.Title,
		Status:             string(bbs.ThreadStatusActive),
		ResCount:           1,
		DefaultName:        rules.DefaultName,
		HideID:             rules.HideID,
		SLIPLevel:          int(rules.SLIP),
		MaxRes:             rules.MaxRes,
		DatOchiIdleHours:   rules.DatOchiIdleHours,
		DatOchiLowResHours: rules.DatOchiLowResHours,
		ExtDat:             input.ExtDat,
		OwnerPosterHash:    input.OwnerPosterHash,
		OwnerAnonID:        input.OwnerAnonID,
		BumpedAt:           now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	postRow := BBSPost{
		ID:           postID,
		ThreadID:     threadID,
		ResNumber:    1,
		Name:         input.FirstPost.Name,
		Trip:         input.FirstPost.Trip,
		SlipText:     input.FirstPost.SlipText,
		AnonID:       input.FirstPost.AnonID,
		PosterHash:   input.FirstPost.PosterHash,
		IsOP:         true,
		IsSage:       input.FirstPost.IsSage,
		Body:         input.FirstPost.Body,
		AuthorUserID: input.FirstPost.AuthorUserID,
		CreatedAt:    now,
	}

	err := p.session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&thRow).Error; err != nil {
			return err
		}
		return tx.Create(&postRow).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return toDomainThread(&thRow), toDomainPost(&postRow), nil
}

func (p *Provider) ArchiveThread(ctx context.Context, id uuid.UUID) (*bbs.Thread, error) {
	var row BBSThread
	err := p.session(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Where("id = ?", id)
		if tx.Name() == "mysql" {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := q.First(&row).Error; err != nil {
			return mapErr(err)
		}
		now := time.Now()
		archivedAt := row.ArchivedAt
		if archivedAt == nil {
			archivedAt = &now
		}
		updates := map[string]any{
			"status":      string(bbs.ThreadStatusArchived),
			"archived_at": archivedAt,
			"updated_at":  now,
		}
		if err := tx.Model(&BBSThread{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		row.Status = string(bbs.ThreadStatusArchived)
		row.ArchivedAt = archivedAt
		row.UpdatedAt = now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toDomainThread(&row), nil
}

func (p *Provider) ListStaleThreads(ctx context.Context, now time.Time) ([]bbs.Thread, error) {
	var active []BBSThread
	if err := p.session(ctx).Where("status = ?", string(bbs.ThreadStatusActive)).Find(&active).Error; err != nil {
		return nil, err
	}
	var stale []bbs.Thread
	for i := range active {
		th := &active[i]
		isIdle := th.DatOchiIdleHours > 0 && th.UpdatedAt.Add(time.Duration(th.DatOchiIdleHours)*time.Hour).Before(now)
		isLowRes := th.DatOchiLowResHours > 0 && th.ResCount < 10 && th.CreatedAt.Add(time.Duration(th.DatOchiLowResHours)*time.Hour).Before(now)
		isOverMax := th.MaxRes > 0 && th.ResCount >= th.MaxRes
		if isIdle || isLowRes || isOverMax {
			stale = append(stale, *toDomainThread(th))
		}
	}
	return stale, nil
}

// --- PostProvider ---

func (p *Provider) GetPost(ctx context.Context, id uuid.UUID) (*bbs.Post, error) {
	var row BBSPost
	if err := p.session(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainPost(&row), nil
}

func (p *Provider) GetPostByNumber(ctx context.Context, threadID uuid.UUID, resNumber uint32) (*bbs.Post, error) {
	var row BBSPost
	if err := p.session(ctx).Where("thread_id = ? AND res_number = ?", threadID, resNumber).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainPost(&row), nil
}

func (p *Provider) ListPosts(ctx context.Context, threadID uuid.UUID, offset, limit int) ([]bbs.Post, error) {
	q := p.session(ctx).Model(&BBSPost{}).Where("thread_id = ?", threadID).Order("res_number ASC")
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []BBSPost
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.Post, len(rows))
	for i := range rows {
		out[i] = *toDomainPost(&rows[i])
	}
	return out, nil
}

func (p *Provider) CreatePost(ctx context.Context, input bbs.CreatePostRecord) (*bbs.Post, *bbs.Thread, error) {
	var thRow BBSThread
	var postRow BBSPost
	now := time.Now()

	err := p.session(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Where("id = ?", input.ThreadID)
		if tx.Name() == "mysql" {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := q.First(&thRow).Error; err != nil {
			return mapErr(err)
		}
		if thRow.Status != string(bbs.ThreadStatusActive) {
			return bbs.ErrThreadArchived
		}
		maxRes := thRow.MaxRes
		if maxRes == 0 {
			maxRes = bbs.DefaultMaxRes
		}
		if thRow.ResCount >= maxRes {
			return bbs.ErrMaxResReached
		}

		nextRes := thRow.ResCount + 1
		postRow = BBSPost{
			ID:           uuidv7.New(),
			ThreadID:     thRow.ID,
			ResNumber:    nextRes,
			Name:         input.Name,
			Trip:         input.Trip,
			SlipText:     input.SlipText,
			AnonID:       input.AnonID,
			PosterHash:   input.PosterHash,
			IsOP:         input.IsOP,
			IsSage:       input.IsSage,
			Body:         input.Body,
			AuthorUserID: input.AuthorUserID,
			CreatedAt:    now,
		}
		if err := tx.Create(&postRow).Error; err != nil {
			return err
		}

		thRow.ResCount = nextRes
		thRow.UpdatedAt = now
		if !input.IsSage {
			thRow.BumpedAt = now
		}
		if nextRes >= maxRes {
			thRow.Status = string(bbs.ThreadStatusArchived)
			thRow.ArchivedAt = &now
		}
		return tx.Save(&thRow).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return toDomainPost(&postRow), toDomainThread(&thRow), nil
}

func (p *Provider) AbonePost(ctx context.Context, id uuid.UUID, reason string) (*bbs.Post, error) {
	var row BBSPost
	err := p.session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
			return mapErr(err)
		}
		row.Abone = true
		row.AboneReason = reason
		return tx.Save(&row).Error
	})
	if err != nil {
		return nil, err
	}
	return toDomainPost(&row), nil
}

// --- Store: GuildConfig & UserConsent ---

func (p *Provider) GetGuildConfig(ctx context.Context, guildID snowflake.ID) (*bbs.GuildConfig, error) {
	var row BBSGuildConfig
	if err := p.session(ctx).Where("guild_id = ?", guildID).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return &bbs.GuildConfig{
		GuildID:         row.GuildID,
		Enabled:         row.Enabled,
		AgreedVersion:   row.AgreedVersion,
		AgreedByUserID:  row.AgreedByUserID,
		AgreedAt:        row.AgreedAt,
		ViewerRoleID:    row.ViewerRoleID,
		DefaultUseForum: row.DefaultUseForum,
		ModLogChannelID: row.ModLogChannelID,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (p *Provider) SaveGuildConfig(ctx context.Context, cfg *bbs.GuildConfig) error {
	row := BBSGuildConfig{
		GuildID:         cfg.GuildID,
		Enabled:         cfg.Enabled,
		AgreedVersion:   cfg.AgreedVersion,
		AgreedByUserID:  cfg.AgreedByUserID,
		AgreedAt:        cfg.AgreedAt,
		ViewerRoleID:    cfg.ViewerRoleID,
		DefaultUseForum: cfg.DefaultUseForum,
		ModLogChannelID: cfg.ModLogChannelID,
		UpdatedAt:       cfg.UpdatedAt,
	}
	return p.session(ctx).Save(&row).Error
}

func (p *Provider) GetUserConsent(ctx context.Context, userID snowflake.ID) (*bbs.UserConsent, error) {
	var row BBSUserConsent
	if err := p.session(ctx).Where("user_id = ?", userID).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return &bbs.UserConsent{
		UserID:        row.UserID,
		PolicyVersion: row.PolicyVersion,
		AgreedAt:      row.AgreedAt,
	}, nil
}

func (p *Provider) SaveUserConsent(ctx context.Context, consent *bbs.UserConsent) error {
	row := BBSUserConsent{
		UserID:        consent.UserID,
		PolicyVersion: consent.PolicyVersion,
		AgreedAt:      consent.AgreedAt,
		UpdatedAt:     time.Now(),
	}
	return p.session(ctx).Save(&row).Error
}

func (p *Provider) DeleteUserConsent(ctx context.Context, userID snowflake.ID) error {
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&BBSThreadWatch{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userID).Delete(&BBSUserConsent{}).Error
	})
}

// --- Store: BoardMount ---

func (p *Provider) CreateBoardMount(ctx context.Context, mount *bbs.BoardMount) error {
	if mount.ID == uuid.Nil {
		mount.ID = uuidv7.New()
	}
	if mount.CreatedAt.IsZero() {
		mount.CreatedAt = time.Now()
	}
	loc := mount.Locale
	if loc == "" {
		loc = "ja"
	}
	row := BBSBoardMount{
		ID:               mount.ID,
		BoardID:          mount.BoardID,
		GuildID:          mount.GuildID,
		ChannelID:        mount.ChannelID,
		Locale:           loc,
		IsForum:          mount.IsForum,
		PinnedPostID:     mount.PinnedPostID,
		ControlMessageID: mount.ControlMessageID,
		Active:           mount.Active,
		CreatedAt:        mount.CreatedAt,
	}
	return p.session(ctx).Create(&row).Error
}

func (p *Provider) GetBoardMount(ctx context.Context, id uuid.UUID) (*bbs.BoardMount, error) {
	var row BBSBoardMount
	if err := p.session(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainBoardMount(&row), nil
}

func (p *Provider) GetBoardMountByGuildAndBoard(ctx context.Context, guildID snowflake.ID, boardID uuid.UUID) (*bbs.BoardMount, error) {
	var row BBSBoardMount
	if err := p.session(ctx).Where("guild_id = ? AND board_id = ? AND active = ?", guildID, boardID, true).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainBoardMount(&row), nil
}

func (p *Provider) GetBoardMountByChannel(ctx context.Context, channelID snowflake.ID) (*bbs.BoardMount, error) {
	var row BBSBoardMount
	if err := p.session(ctx).Where("channel_id = ?", channelID).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainBoardMount(&row), nil
}

func (p *Provider) ListActiveBoardMountsByBoard(ctx context.Context, boardID uuid.UUID) ([]bbs.BoardMount, error) {
	var rows []BBSBoardMount
	if err := p.session(ctx).Where("board_id = ? AND active = ?", boardID, true).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.BoardMount, len(rows))
	for i := range rows {
		out[i] = *toDomainBoardMount(&rows[i])
	}
	return out, nil
}

func (p *Provider) ListBoardMountsByGuild(ctx context.Context, guildID snowflake.ID) ([]bbs.BoardMount, error) {
	var rows []BBSBoardMount
	if err := p.session(ctx).Where("guild_id = ? AND active = ?", guildID, true).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.BoardMount, len(rows))
	for i := range rows {
		out[i] = *toDomainBoardMount(&rows[i])
	}
	return out, nil
}

func (p *Provider) UpdateBoardMount(ctx context.Context, mount *bbs.BoardMount) error {
	loc := mount.Locale
	if loc == "" {
		loc = "ja"
	}
	row := BBSBoardMount{
		ID:               mount.ID,
		BoardID:          mount.BoardID,
		GuildID:          mount.GuildID,
		ChannelID:        mount.ChannelID,
		Locale:           loc,
		IsForum:          mount.IsForum,
		PinnedPostID:     mount.PinnedPostID,
		ControlMessageID: mount.ControlMessageID,
		Active:           mount.Active,
		CreatedAt:        mount.CreatedAt,
	}
	return p.session(ctx).Save(&row).Error
}

func (p *Provider) UpdateBoardMountControl(ctx context.Context, id uuid.UUID, controlMsgID *snowflake.ID) (bool, error) {
	res := p.session(ctx).Model(&BBSBoardMount{}).
		Where("id = ? AND active = ?", id, true).
		Update("control_message_id", controlMsgID)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (p *Provider) DeactivateBoardMount(ctx context.Context, id uuid.UUID) error {
	return p.session(ctx).Model(&BBSBoardMount{}).Where("id = ?", id).Update("active", false).Error
}

// --- Store: ThreadMirror ---

func (p *Provider) CreateThreadMirror(ctx context.Context, mirror *bbs.ThreadMirror) error {
	if mirror.ID == uuid.Nil {
		mirror.ID = uuidv7.New()
	}
	if mirror.CreatedAt.IsZero() {
		mirror.CreatedAt = time.Now()
	}
	row := BBSThreadMirror{
		ID:               mirror.ID,
		ThreadID:         mirror.ThreadID,
		BoardMountID:     mirror.BoardMountID,
		GuildID:          mirror.GuildID,
		DiscordThreadID:  mirror.DiscordThreadID,
		ControlMessageID: mirror.ControlMessageID,
		SyncedUpToRes:    mirror.SyncedUpToRes,
		CreatedAt:        mirror.CreatedAt,
	}
	return p.session(ctx).Create(&row).Error
}

func (p *Provider) CreateThreadAndPostMirror(ctx context.Context, threadMirror *bbs.ThreadMirror, postMirror *bbs.PostMirror) error {
	now := time.Now()
	if threadMirror.ID == uuid.Nil {
		threadMirror.ID = uuidv7.New()
	}
	if threadMirror.CreatedAt.IsZero() {
		threadMirror.CreatedAt = now
	}
	if postMirror.ID == uuid.Nil {
		postMirror.ID = uuidv7.New()
	}
	if postMirror.ThreadMirrorID == uuid.Nil {
		postMirror.ThreadMirrorID = threadMirror.ID
	}
	if postMirror.CreatedAt.IsZero() {
		postMirror.CreatedAt = now
	}
	tmRow := BBSThreadMirror{
		ID:               threadMirror.ID,
		ThreadID:         threadMirror.ThreadID,
		BoardMountID:     threadMirror.BoardMountID,
		GuildID:          threadMirror.GuildID,
		DiscordThreadID:  threadMirror.DiscordThreadID,
		ControlMessageID: threadMirror.ControlMessageID,
		SyncedUpToRes:    threadMirror.SyncedUpToRes,
		CreatedAt:        threadMirror.CreatedAt,
	}
	pmRow := BBSPostMirror{
		ID:               postMirror.ID,
		PostID:           postMirror.PostID,
		ThreadMirrorID:   postMirror.ThreadMirrorID,
		GuildID:          postMirror.GuildID,
		DiscordMessageID: postMirror.DiscordMessageID,
		CreatedAt:        postMirror.CreatedAt,
	}
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tmRow).Error; err != nil {
			return err
		}
		return tx.Create(&pmRow).Error
	})
}

func (p *Provider) GetThreadMirror(ctx context.Context, id uuid.UUID) (*bbs.ThreadMirror, error) {
	var row BBSThreadMirror
	if err := p.session(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainThreadMirror(&row), nil
}

func (p *Provider) GetThreadMirrorByDiscordThread(ctx context.Context, discordThreadID snowflake.ID) (*bbs.ThreadMirror, error) {
	var row BBSThreadMirror
	if err := p.session(ctx).Where("discord_thread_id = ?", discordThreadID).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainThreadMirror(&row), nil
}

func (p *Provider) GetThreadMirrorByMountAndThread(ctx context.Context, mountID, threadID uuid.UUID) (*bbs.ThreadMirror, error) {
	var row BBSThreadMirror
	if err := p.session(ctx).Where("board_mount_id = ? AND thread_id = ?", mountID, threadID).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainThreadMirror(&row), nil
}

func (p *Provider) ListThreadMirrorsByThread(ctx context.Context, threadID uuid.UUID) ([]bbs.ThreadMirror, error) {
	var rows []BBSThreadMirror
	if err := p.session(ctx).Where("thread_id = ?", threadID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.ThreadMirror, len(rows))
	for i := range rows {
		out[i] = *toDomainThreadMirror(&rows[i])
	}
	return out, nil
}

func (p *Provider) UpdateThreadMirror(ctx context.Context, mirror *bbs.ThreadMirror) error {
	row := BBSThreadMirror{
		ID:               mirror.ID,
		ThreadID:         mirror.ThreadID,
		BoardMountID:     mirror.BoardMountID,
		GuildID:          mirror.GuildID,
		DiscordThreadID:  mirror.DiscordThreadID,
		ControlMessageID: mirror.ControlMessageID,
		SyncedUpToRes:    mirror.SyncedUpToRes,
		CreatedAt:        mirror.CreatedAt,
	}
	return p.session(ctx).Save(&row).Error
}

func (p *Provider) AdvanceThreadMirrorSync(ctx context.Context, id uuid.UUID, syncedUpTo uint32) error {
	return p.session(ctx).Model(&BBSThreadMirror{}).
		Where("id = ? AND synced_up_to_res < ?", id, syncedUpTo).
		Update("synced_up_to_res", syncedUpTo).Error
}

func (p *Provider) UpdateThreadMirrorControl(ctx context.Context, id uuid.UUID, controlMsgID *snowflake.ID) error {
	return p.session(ctx).Model(&BBSThreadMirror{}).
		Where("id = ?", id).
		Update("control_message_id", controlMsgID).Error
}

func (p *Provider) DeleteThreadMirror(ctx context.Context, id uuid.UUID) error {
	return p.session(ctx).Where("id = ?", id).Delete(&BBSThreadMirror{}).Error
}

// --- Store: PostMirror ---

func (p *Provider) CreatePostMirror(ctx context.Context, mirror *bbs.PostMirror) error {
	if mirror.ID == uuid.Nil {
		mirror.ID = uuidv7.New()
	}
	if mirror.CreatedAt.IsZero() {
		mirror.CreatedAt = time.Now()
	}
	row := BBSPostMirror{
		ID:               mirror.ID,
		PostID:           mirror.PostID,
		ThreadMirrorID:   mirror.ThreadMirrorID,
		GuildID:          mirror.GuildID,
		DiscordMessageID: mirror.DiscordMessageID,
		CreatedAt:        mirror.CreatedAt,
	}
	return p.session(ctx).Create(&row).Error
}

func (p *Provider) GetPostMirrorByDiscordMessage(ctx context.Context, discordMessageID snowflake.ID) (*bbs.PostMirror, error) {
	var row BBSPostMirror
	if err := p.session(ctx).Where("discord_message_id = ?", discordMessageID).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainPostMirror(&row), nil
}

func (p *Provider) GetPostMirrorByPostAndGuild(ctx context.Context, postID uuid.UUID, guildID snowflake.ID) (*bbs.PostMirror, error) {
	var row BBSPostMirror
	if err := p.session(ctx).Where("post_id = ? AND guild_id = ?", postID, guildID).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainPostMirror(&row), nil
}

func (p *Provider) GetPostMirrorByResAndGuild(ctx context.Context, threadID uuid.UUID, resNumber uint32, guildID snowflake.ID) (*bbs.PostMirror, error) {
	var post BBSPost
	if err := p.session(ctx).Where("thread_id = ? AND res_number = ?", threadID, resNumber).First(&post).Error; err != nil {
		return nil, mapErr(err)
	}
	return p.GetPostMirrorByPostAndGuild(ctx, post.ID, guildID)
}

func (p *Provider) ListPostMirrorsByPost(ctx context.Context, postID uuid.UUID) ([]bbs.PostMirror, error) {
	var rows []BBSPostMirror
	if err := p.session(ctx).Where("post_id = ?", postID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.PostMirror, len(rows))
	for i := range rows {
		out[i] = *toDomainPostMirror(&rows[i])
	}
	return out, nil
}

// --- Store: PostReference ---

func (p *Provider) SavePostReferences(ctx context.Context, refs []bbs.PostReference) error {
	if len(refs) == 0 {
		return nil
	}
	rows := make([]BBSPostReference, len(refs))
	for i, r := range refs {
		rows[i] = BBSPostReference{
			ThreadID: r.ThreadID,
			FromRes:  r.FromRes,
			ToRes:    r.ToRes,
		}
	}
	return p.session(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func (p *Provider) ListReferencingRes(ctx context.Context, threadID uuid.UUID, targetRes uint32) ([]uint32, error) {
	var rows []BBSPostReference
	if err := p.session(ctx).Where("thread_id = ? AND to_res = ?", threadID, targetRes).Order("from_res ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]uint32, len(rows))
	for i, r := range rows {
		out[i] = r.FromRes
	}
	return out, nil
}

// --- Store: ThreadWatch ---

func (p *Provider) GetThreadWatch(ctx context.Context, threadID uuid.UUID, userID snowflake.ID) (*bbs.ThreadWatch, error) {
	var row BBSThreadWatch
	if err := p.session(ctx).Where("thread_id = ? AND user_id = ?", threadID, userID).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return &bbs.ThreadWatch{
		ID:        row.ID,
		ThreadID:  row.ThreadID,
		UserID:    row.UserID,
		Locale:    row.Locale,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (p *Provider) SaveThreadWatch(ctx context.Context, watch *bbs.ThreadWatch) error {
	if watch.ID == uuid.Nil {
		watch.ID = uuidv7.New()
	}
	if watch.CreatedAt.IsZero() {
		watch.CreatedAt = time.Now()
	}
	loc := watch.Locale
	if loc == "" {
		loc = "ja"
	}
	row := BBSThreadWatch{
		ID:        watch.ID,
		ThreadID:  watch.ThreadID,
		UserID:    watch.UserID,
		Locale:    loc,
		CreatedAt: watch.CreatedAt,
	}
	return p.session(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func (p *Provider) DeleteThreadWatch(ctx context.Context, threadID uuid.UUID, userID snowflake.ID) error {
	return p.session(ctx).Where("thread_id = ? AND user_id = ?", threadID, userID).Delete(&BBSThreadWatch{}).Error
}

func (p *Provider) ListThreadWatches(ctx context.Context, threadID uuid.UUID) ([]bbs.ThreadWatch, error) {
	var rows []BBSThreadWatch
	if err := p.session(ctx).Where("thread_id = ?", threadID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.ThreadWatch, len(rows))
	for i, r := range rows {
		out[i] = bbs.ThreadWatch{
			ID:        r.ID,
			ThreadID:  r.ThreadID,
			UserID:    r.UserID,
			Locale:    r.Locale,
			CreatedAt: r.CreatedAt,
		}
	}
	return out, nil
}

// --- Store: Ban ---

func (p *Provider) CreateBan(ctx context.Context, ban *bbs.Ban) error {
	if ban.ID == uuid.Nil {
		ban.ID = uuidv7.New()
	}
	if ban.CreatedAt.IsZero() {
		ban.CreatedAt = time.Now()
	}
	row := BBSBan{
		ID:           ban.ID,
		Scope:        string(ban.Scope),
		GuildID:      ban.GuildID,
		PosterHash:   ban.PosterHash,
		TargetUserID: ban.TargetUserID,
		Reason:       ban.Reason,
		CreatedBy:    ban.CreatedBy,
		ExpiresAt:    ban.ExpiresAt,
		CreatedAt:    ban.CreatedAt,
	}
	return p.session(ctx).Create(&row).Error
}

func (p *Provider) DeleteBan(ctx context.Context, id uuid.UUID) error {
	res := p.session(ctx).Where("id = ?", id).Delete(&BBSBan{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return bbs.ErrNotFound
	}
	return nil
}

func (p *Provider) ListBans(ctx context.Context, filter bbs.BanFilter) ([]bbs.Ban, error) {
	now := filter.Now
	if now.IsZero() {
		now = time.Now()
	}
	q := p.session(ctx).Model(&BBSBan{}).
		Where("expires_at IS NULL OR expires_at > ?", now).
		Order("created_at DESC")
	if filter.Scope != nil {
		q = q.Where("scope = ?", string(*filter.Scope))
	}
	if filter.GuildID != nil {
		q = q.Where("guild_id = ?", *filter.GuildID)
	}
	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}
	var rows []BBSBan
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.Ban, len(rows))
	for i := range rows {
		out[i] = *toDomainBan(&rows[i])
	}
	return out, nil
}

func (p *Provider) IsBanned(ctx context.Context, scope bbs.Scope, guildID snowflake.ID, posterHash string, userID snowflake.ID, now time.Time) (bool, error) {
	// Check global bans first (by TargetUserID or PosterHash)
	var globalCount int64
	if err := p.session(ctx).Model(&BBSBan{}).
		Where("scope = ? AND (expires_at IS NULL OR expires_at > ?) AND (target_user_id = ? OR poster_hash = ?)",
			string(bbs.ScopeGlobal), now, userID, posterHash).
		Count(&globalCount).Error; err != nil {
		return false, err
	}
	if globalCount > 0 {
		return true, nil
	}
	if scope == bbs.ScopeLocal {
		var localCount int64
		if err := p.session(ctx).Model(&BBSBan{}).
			Where("scope = ? AND guild_id = ? AND (expires_at IS NULL OR expires_at > ?) AND poster_hash = ?",
				string(bbs.ScopeLocal), guildID, now, posterHash).
			Count(&localCount).Error; err != nil {
			return false, err
		}
		return localCount > 0, nil
	}
	return false, nil
}

// --- Store: Report & Operator Audit ---

func (p *Provider) CreateReport(ctx context.Context, report *bbs.Report) error {
	if report.ID == uuid.Nil {
		report.ID = uuidv7.New()
	}
	row := BBSReport{
		ID:             report.ID,
		Scope:          string(report.Scope),
		GuildID:        report.GuildID,
		PostID:         report.PostID,
		ThreadID:       report.ThreadID,
		BoardID:        report.BoardID,
		ReporterUserID: report.ReporterUserID,
		Reason:         report.Reason,
		Status:         string(report.Status),
		HandledBy:      report.HandledBy,
		ResolutionNote: report.ResolutionNote,
		CreatedAt:      report.CreatedAt,
		UpdatedAt:      report.UpdatedAt,
	}
	return p.session(ctx).Create(&row).Error
}

func (p *Provider) GetReport(ctx context.Context, id uuid.UUID) (*bbs.Report, error) {
	var row BBSReport
	if err := p.session(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainReport(&row), nil
}

func (p *Provider) ListReports(ctx context.Context, filter bbs.ReportFilter) ([]bbs.Report, error) {
	q := p.session(ctx).Model(&BBSReport{}).Order("created_at DESC")
	if filter.Scope != nil {
		q = q.Where("scope = ?", string(*filter.Scope))
	}
	if filter.Status != nil {
		q = q.Where("status = ?", string(*filter.Status))
	}
	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}
	var rows []BBSReport
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.Report, len(rows))
	for i := range rows {
		out[i] = *toDomainReport(&rows[i])
	}
	return out, nil
}

func (p *Provider) UpdateReport(ctx context.Context, report *bbs.Report) error {
	return p.session(ctx).Model(&BBSReport{}).Where("id = ?", report.ID).Updates(map[string]any{
		"status":          string(report.Status),
		"handled_by":      report.HandledBy,
		"resolution_note": report.ResolutionNote,
		"updated_at":      time.Now(),
	}).Error
}

func (p *Provider) GetAuditPostInfo(ctx context.Context, postID uuid.UUID) (*bbs.AuditPostInfo, error) {
	var row BBSPost
	if err := p.session(ctx).Where("id = ?", postID).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	var mirror BBSPostMirror
	var guildID snowflake.ID
	if err := p.session(ctx).Where("post_id = ?", postID).Order("created_at ASC").First(&mirror).Error; err == nil {
		guildID = mirror.GuildID
	}
	return &bbs.AuditPostInfo{
		Post:         *toDomainPost(&row),
		AuthorUserID: row.AuthorUserID,
		GuildID:      guildID,
	}, nil
}

// --- Store: System Secret ---

func (p *Provider) GetOrCreateSystemSecret(ctx context.Context, secretKey string, fallback []byte) ([]byte, error) {
	var row BBSSystemSecret
	err := p.session(ctx).Where("secret_key = ?", secretKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		newRow := BBSSystemSecret{
			SecretKey: secretKey,
			SecretHex: hex.EncodeToString(fallback),
			CreatedAt: time.Now(),
		}
		if cErr := p.session(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&newRow).Error; cErr != nil {
			return nil, cErr
		}
		err = p.session(ctx).Where("secret_key = ?", secretKey).First(&row).Error
	}
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(row.SecretHex)
}

// --- Mappers ---

func toDomainCategory(r *BBSCategory) *bbs.Category {
	return &bbs.Category{
		ID:                r.ID,
		Scope:             bbs.Scope(r.Scope),
		GuildID:           r.GuildID,
		ProviderKey:       r.ProviderKey,
		ExternalID:        r.ExternalID,
		Slug:              r.Slug,
		Name:              r.Name,
		Description:       r.Description,
		SortOrder:         r.SortOrder,
		DiscordCategoryID: r.DiscordCategoryID,
		CreatedAt:         r.CreatedAt,
	}
}

func toDomainBoard(r *BBSBoard) *bbs.Board {
	return &bbs.Board{
		ID:          r.ID,
		CategoryID:  r.CategoryID,
		Scope:       bbs.Scope(r.Scope),
		GuildID:     r.GuildID,
		ProviderKey: r.ProviderKey,
		ExternalID:  r.ExternalID,
		Slug:        r.Slug,
		Name:        r.Name,
		Description: r.Description,
		Rules: bbs.BoardRules{
			DefaultName:        r.DefaultName,
			HideID:             r.HideID,
			SLIP:               bbs.SLIPLevel(r.SLIPLevel),
			MaxRes:             r.MaxRes,
			DatOchiIdleHours:   r.DatOchiIdleHours,
			DatOchiLowResHours: r.DatOchiLowResHours,
		},
		Locked:    r.Locked,
		CreatedAt: r.CreatedAt,
	}
}

func toDomainThread(r *BBSThread) *bbs.Thread {
	return &bbs.Thread{
		ID:          r.ID,
		BoardID:     r.BoardID,
		ProviderKey: r.ProviderKey,
		ExternalID:  r.ExternalID,
		Title:       r.Title,
		Status:      bbs.ThreadStatus(r.Status),
		ResCount:    r.ResCount,
		Rules: bbs.BoardRules{
			DefaultName:        r.DefaultName,
			HideID:             r.HideID,
			SLIP:               bbs.SLIPLevel(r.SLIPLevel),
			MaxRes:             r.MaxRes,
			DatOchiIdleHours:   r.DatOchiIdleHours,
			DatOchiLowResHours: r.DatOchiLowResHours,
		},
		ExtDat:          r.ExtDat,
		OwnerPosterHash: r.OwnerPosterHash,
		OwnerAnonID:     r.OwnerAnonID,
		BumpedAt:        r.BumpedAt,
		ArchivedAt:      r.ArchivedAt,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

func toDomainPost(r *BBSPost) *bbs.Post {
	return &bbs.Post{
		ID:          r.ID,
		ThreadID:    r.ThreadID,
		ResNumber:   r.ResNumber,
		Name:        r.Name,
		Trip:        r.Trip,
		SlipText:    r.SlipText,
		AnonID:      r.AnonID,
		PosterHash:  r.PosterHash,
		IsOP:        r.IsOP,
		IsSage:      r.IsSage,
		Body:        r.Body,
		Abone:       r.Abone,
		AboneReason: r.AboneReason,
		CreatedAt:   r.CreatedAt,
	}
}

func toDomainBoardMount(r *BBSBoardMount) *bbs.BoardMount {
	return &bbs.BoardMount{
		ID:               r.ID,
		BoardID:          r.BoardID,
		GuildID:          r.GuildID,
		ChannelID:        r.ChannelID,
		Locale:           r.Locale,
		IsForum:          r.IsForum,
		PinnedPostID:     r.PinnedPostID,
		ControlMessageID: r.ControlMessageID,
		Active:           r.Active,
		CreatedAt:        r.CreatedAt,
	}
}

func toDomainThreadMirror(r *BBSThreadMirror) *bbs.ThreadMirror {
	return &bbs.ThreadMirror{
		ID:               r.ID,
		ThreadID:         r.ThreadID,
		BoardMountID:     r.BoardMountID,
		GuildID:          r.GuildID,
		DiscordThreadID:  r.DiscordThreadID,
		ControlMessageID: r.ControlMessageID,
		SyncedUpToRes:    r.SyncedUpToRes,
		CreatedAt:        r.CreatedAt,
	}
}

func toDomainPostMirror(r *BBSPostMirror) *bbs.PostMirror {
	return &bbs.PostMirror{
		ID:               r.ID,
		PostID:           r.PostID,
		ThreadMirrorID:   r.ThreadMirrorID,
		GuildID:          r.GuildID,
		DiscordMessageID: r.DiscordMessageID,
		CreatedAt:        r.CreatedAt,
	}
}

func toDomainBan(r *BBSBan) *bbs.Ban {
	return &bbs.Ban{
		ID:           r.ID,
		Scope:        bbs.Scope(r.Scope),
		GuildID:      r.GuildID,
		PosterHash:   r.PosterHash,
		TargetUserID: r.TargetUserID,
		Reason:       r.Reason,
		CreatedBy:    r.CreatedBy,
		ExpiresAt:    r.ExpiresAt,
		CreatedAt:    r.CreatedAt,
	}
}

func toDomainReport(r *BBSReport) *bbs.Report {
	return &bbs.Report{
		ID:             r.ID,
		Scope:          bbs.Scope(r.Scope),
		GuildID:        r.GuildID,
		PostID:         r.PostID,
		ThreadID:       r.ThreadID,
		BoardID:        r.BoardID,
		ReporterUserID: r.ReporterUserID,
		Reason:         r.Reason,
		Status:         bbs.ReportStatus(r.Status),
		HandledBy:      r.HandledBy,
		ResolutionNote: r.ResolutionNote,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}
