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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sabafly/gobot/internal/bbs"
)

func (p *Provider) ListBoardMounts(ctx context.Context, filter bbs.BoardMountFilter) ([]bbs.BoardMount, error) {
	q := p.session(ctx).Model(&BBSBoardMount{}).Order("created_at ASC")
	if filter.GuildID != nil {
		q = q.Where("guild_id = ?", *filter.GuildID)
	}
	if filter.BoardID != nil {
		q = q.Where("board_id = ?", *filter.BoardID)
	}
	if filter.Active != nil {
		q = q.Where("active = ?", *filter.Active)
	}
	var rows []BBSBoardMount
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.BoardMount, len(rows))
	for i := range rows {
		out[i] = *toDomainBoardMount(&rows[i])
	}
	return out, nil
}

const (
	adminCapsSeedKey  = "seed:admin_caps_v1"
	slipEventsSeedKey = "seed:slip_events_v1"
)

func encodeSnowflakesJSON(ids []snowflake.ID) (string, error) {
	strs := make([]string, 0, len(ids))
	for _, id := range ids {
		strs = append(strs, id.String())
	}
	b, err := json.Marshal(strs)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeSnowflakesJSON(raw string) ([]snowflake.ID, error) {
	if raw == "" {
		return nil, nil
	}
	var strs []string
	if err := json.Unmarshal([]byte(raw), &strs); err != nil {
		return nil, err
	}
	if len(strs) == 0 {
		return nil, nil
	}
	out := make([]snowflake.ID, 0, len(strs))
	for _, s := range strs {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid snowflake id %q: %w", s, err)
		}
		out = append(out, snowflake.ID(v))
	}
	return out, nil
}

func toDomainAdminCap(row *BBSAdminCap) (bbs.CapDefinition, error) {
	users, err := decodeSnowflakesJSON(row.UsersJSON)
	if err != nil {
		return bbs.CapDefinition{}, err
	}
	return bbs.CapDefinition{
		ID:             row.ID,
		Name:           row.Name,
		Description:    row.Description,
		GrantOperators: row.GrantOperators,
		Users:          users,
	}, nil
}

func toDomainSlipEvent(row *BBSSlipEvent) bbs.SlipEventDefinition {
	return bbs.SlipEventDefinition{
		ID:        row.ID,
		Label:     row.Label,
		StartDate: row.StartDate,
		EndDate:   row.EndDate,
		Mode:      bbs.SlipEventMode(row.Mode),
	}
}

func markSeededTx(tx *gorm.DB, key string) error {
	row := BBSSystemSecret{
		SecretKey: key,
		SecretHex: "31",
		CreatedAt: time.Now(),
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func isSeededTx(tx *gorm.DB, key string) (bool, error) {
	var count int64
	if err := tx.Model(&BBSSystemSecret{}).Where("secret_key = ?", key).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (p *Provider) ListAdminCaps(ctx context.Context) ([]bbs.CapDefinition, error) {
	return listAdminCapsTx(p.session(ctx))
}

func listAdminCapsTx(db *gorm.DB) ([]bbs.CapDefinition, error) {
	var rows []BBSAdminCap
	if err := db.Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.CapDefinition, 0, len(rows))
	for i := range rows {
		c, err := toDomainAdminCap(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (p *Provider) GetAdminCap(ctx context.Context, id string) (*bbs.CapDefinition, error) {
	var row BBSAdminCap
	if err := p.session(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	c, err := toDomainAdminCap(&row)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (p *Provider) UpsertAdminCap(ctx context.Context, def bbs.CapDefinition) error {
	usersJSON, err := encodeSnowflakesJSON(def.Users)
	if err != nil {
		return err
	}
	now := time.Now()
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		var existing BBSAdminCap
		findErr := tx.Where("id = ?", def.ID).First(&existing).Error
		switch {
		case findErr == nil:
			if err := tx.Model(&BBSAdminCap{}).Where("id = ?", def.ID).Updates(map[string]any{
				"name":            def.Name,
				"description":     def.Description,
				"grant_operators": def.GrantOperators,
				"users_json":      usersJSON,
				"updated_at":      now,
			}).Error; err != nil {
				return err
			}
		case errors.Is(findErr, gorm.ErrRecordNotFound):
			var maxSort int
			_ = tx.Model(&BBSAdminCap{}).Select("COALESCE(MAX(sort_order), -1)").Scan(&maxSort).Error
			row := BBSAdminCap{
				ID:             def.ID,
				Name:           def.Name,
				Description:    def.Description,
				GrantOperators: def.GrantOperators,
				UsersJSON:      usersJSON,
				SortOrder:      maxSort + 1,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		default:
			return findErr
		}
		return markSeededTx(tx, adminCapsSeedKey)
	})
}

func (p *Provider) DeleteAdminCap(ctx context.Context, id string) error {
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ?", id).Delete(&BBSAdminCap{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return bbs.ErrNotFound
		}
		return markSeededTx(tx, adminCapsSeedKey)
	})
}

func (p *Provider) SeedAdminCapsIfEmpty(ctx context.Context, defs []bbs.CapDefinition) ([]bbs.CapDefinition, error) {
	var out []bbs.CapDefinition
	err := p.session(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := listAdminCapsTx(tx)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			out = existing
			return nil
		}
		seeded, err := isSeededTx(tx, adminCapsSeedKey)
		if err != nil {
			return err
		}
		if seeded || len(defs) == 0 {
			out = []bbs.CapDefinition{}
			return nil
		}
		now := time.Now()
		for i, def := range defs {
			usersJSON, err := encodeSnowflakesJSON(def.Users)
			if err != nil {
				return err
			}
			row := BBSAdminCap{
				ID:             def.ID,
				Name:           def.Name,
				Description:    def.Description,
				GrantOperators: def.GrantOperators,
				UsersJSON:      usersJSON,
				SortOrder:      i,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		if err := markSeededTx(tx, adminCapsSeedKey); err != nil {
			return err
		}
		out, err = listAdminCapsTx(tx)
		return err
	})
	return out, err
}

func (p *Provider) ListSlipEvents(ctx context.Context) ([]bbs.SlipEventDefinition, error) {
	return listSlipEventsTx(p.session(ctx))
}

func listSlipEventsTx(db *gorm.DB) ([]bbs.SlipEventDefinition, error) {
	var rows []BBSSlipEvent
	if err := db.Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.SlipEventDefinition, 0, len(rows))
	for i := range rows {
		out = append(out, toDomainSlipEvent(&rows[i]))
	}
	return out, nil
}

func (p *Provider) GetSlipEvent(ctx context.Context, id string) (*bbs.SlipEventDefinition, error) {
	var row BBSSlipEvent
	if err := p.session(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	ev := toDomainSlipEvent(&row)
	return &ev, nil
}

func (p *Provider) UpsertSlipEvent(ctx context.Context, def bbs.SlipEventDefinition) error {
	now := time.Now()
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		var existing BBSSlipEvent
		findErr := tx.Where("id = ?", def.ID).First(&existing).Error
		switch {
		case findErr == nil:
			if err := tx.Model(&BBSSlipEvent{}).Where("id = ?", def.ID).Updates(map[string]any{
				"label":      def.Label,
				"start_date": def.StartDate,
				"end_date":   def.EndDate,
				"mode":       string(def.Mode),
				"updated_at": now,
			}).Error; err != nil {
				return err
			}
		case errors.Is(findErr, gorm.ErrRecordNotFound):
			var maxSort int
			_ = tx.Model(&BBSSlipEvent{}).Select("COALESCE(MAX(sort_order), -1)").Scan(&maxSort).Error
			row := BBSSlipEvent{
				ID:        def.ID,
				Label:     def.Label,
				StartDate: def.StartDate,
				EndDate:   def.EndDate,
				Mode:      string(def.Mode),
				SortOrder: maxSort + 1,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		default:
			return findErr
		}
		return markSeededTx(tx, slipEventsSeedKey)
	})
}

func (p *Provider) DeleteSlipEvent(ctx context.Context, id string) error {
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ?", id).Delete(&BBSSlipEvent{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return bbs.ErrNotFound
		}
		return markSeededTx(tx, slipEventsSeedKey)
	})
}

func (p *Provider) SeedSlipEventsIfEmpty(ctx context.Context, defs []bbs.SlipEventDefinition) ([]bbs.SlipEventDefinition, error) {
	var out []bbs.SlipEventDefinition
	err := p.session(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := listSlipEventsTx(tx)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			out = existing
			return nil
		}
		seeded, err := isSeededTx(tx, slipEventsSeedKey)
		if err != nil {
			return err
		}
		if seeded || len(defs) == 0 {
			out = []bbs.SlipEventDefinition{}
			return nil
		}
		now := time.Now()
		for i, def := range defs {
			row := BBSSlipEvent{
				ID:        def.ID,
				Label:     def.Label,
				StartDate: def.StartDate,
				EndDate:   def.EndDate,
				Mode:      string(def.Mode),
				SortOrder: i,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		if err := markSeededTx(tx, slipEventsSeedKey); err != nil {
			return err
		}
		out, err = listSlipEventsTx(tx)
		return err
	})
	return out, err
}

func (p *Provider) GetBBSStats(ctx context.Context) (*bbs.Stats, error) {
	db := p.session(ctx)
	var out bbs.Stats
	if err := db.Model(&BBSCategory{}).Count(&out.Categories).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&BBSBoard{}).Count(&out.Boards).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&BBSThread{}).Where("status = ?", string(bbs.ThreadStatusActive)).Count(&out.ActiveThreads).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&BBSReport{}).Where("status = ?", string(bbs.ReportStatusOpen)).Count(&out.OpenReports).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	if err := db.Model(&BBSBan{}).Where("expires_at IS NULL OR expires_at > ?", now).Count(&out.ActiveBans).Error; err != nil {
		return nil, err
	}
	return &out, nil
}
