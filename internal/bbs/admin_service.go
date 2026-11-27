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
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

// LoadOrSeedAdminCaps validates optional startup YAML caps, seeds the DB if it has not yet
// been initialized, and loads the authoritative DB cap definitions into memory.
func (s *Service) LoadOrSeedAdminCaps(ctx context.Context, yamlDefs []CapDefinition) error {
	validatedYAML, err := ValidateCapDefinitions(yamlDefs)
	if err != nil {
		return err
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()

	defs, err := s.Store.SeedAdminCapsIfEmpty(ctx, validatedYAML)
	if err != nil {
		return err
	}
	return s.SetAdminCaps(defs)
}

// LoadOrSeedSlipEvents validates optional startup YAML SLIP events, seeds the DB if it has not
// yet been initialized, and loads the authoritative DB SLIP event definitions into memory.
func (s *Service) LoadOrSeedSlipEvents(ctx context.Context, yamlDefs []SlipEventDefinition) error {
	validatedYAML, err := ValidateSlipEventDefinitions(yamlDefs)
	if err != nil {
		return err
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()

	defs, err := s.Store.SeedSlipEventsIfEmpty(ctx, validatedYAML)
	if err != nil {
		return err
	}
	return s.SetSlipEvents(defs)
}

// ListAdminCaps returns all persisted admin/operator cap definitions.
func (s *Service) ListAdminCaps(ctx context.Context) ([]CapDefinition, error) {
	return s.Store.ListAdminCaps(ctx)
}

// GetAdminCap retrieves a persisted cap definition by ID.
func (s *Service) GetAdminCap(ctx context.Context, id string) (*CapDefinition, error) {
	return s.Store.GetAdminCap(ctx, strings.TrimSpace(id))
}

// CreateAdminCap validates and inserts a new persisted cap definition, refreshing memory immediately.
func (s *Service) CreateAdminCap(ctx context.Context, def CapDefinition) (*CapDefinition, error) {
	return s.upsertAdminCapLocked(ctx, def, false)
}

// UpdateAdminCap validates and updates an existing persisted cap definition, refreshing memory immediately.
func (s *Service) UpdateAdminCap(ctx context.Context, id string, def CapDefinition) (*CapDefinition, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: cap id is required", ErrInvalidCapConfig)
	}
	if def.ID != "" && !strings.EqualFold(strings.TrimSpace(def.ID), id) {
		return nil, fmt.Errorf("%w: cap id mismatch", ErrInvalidCapConfig)
	}
	def.ID = id
	return s.upsertAdminCapLocked(ctx, def, true)
}

func (s *Service) upsertAdminCapLocked(ctx context.Context, def CapDefinition, mustExist bool) (*CapDefinition, error) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	existing, err := s.Store.ListAdminCaps(ctx)
	if err != nil {
		return nil, err
	}
	targetID := strings.TrimSpace(def.ID)
	merged := make([]CapDefinition, 0, len(existing)+1)
	foundIdx := -1
	for _, cur := range existing {
		if strings.EqualFold(cur.ID, targetID) {
			foundIdx = len(merged)
			def.ID = cur.ID // preserve stored casing on update
			merged = append(merged, def)
			continue
		}
		merged = append(merged, cur)
	}
	if !mustExist && foundIdx >= 0 {
		return nil, ErrAlreadyExists
	}
	if mustExist && foundIdx < 0 {
		return nil, ErrNotFound
	}
	if foundIdx < 0 {
		foundIdx = len(merged)
		merged = append(merged, def)
	}
	validated, err := ValidateCapDefinitions(merged)
	if err != nil {
		return nil, err
	}
	saved := validated[foundIdx]
	if err := s.Store.UpsertAdminCap(ctx, saved); err != nil {
		return nil, err
	}
	all, err := s.Store.ListAdminCaps(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.SetAdminCaps(all); err != nil {
		return nil, err
	}
	return &saved, nil
}

// RemoveAdminCap deletes a persisted cap definition and refreshes memory immediately.
func (s *Service) RemoveAdminCap(ctx context.Context, id string) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	target := strings.TrimSpace(id)
	if target == "" {
		return ErrNotFound
	}
	existing, err := s.Store.ListAdminCaps(ctx)
	if err != nil {
		return err
	}
	storedID := ""
	for _, cur := range existing {
		if strings.EqualFold(cur.ID, target) {
			storedID = cur.ID
			break
		}
	}
	if storedID == "" {
		return ErrNotFound
	}
	if err := s.Store.DeleteAdminCap(ctx, storedID); err != nil {
		return err
	}
	all, err := s.Store.ListAdminCaps(ctx)
	if err != nil {
		return err
	}
	return s.SetAdminCaps(all)
}

// ListSlipEvents returns all persisted SLIP event definitions.
func (s *Service) ListSlipEvents(ctx context.Context) ([]SlipEventDefinition, error) {
	return s.Store.ListSlipEvents(ctx)
}

// GetSlipEvent retrieves a persisted SLIP event definition by ID.
func (s *Service) GetSlipEvent(ctx context.Context, id string) (*SlipEventDefinition, error) {
	return s.Store.GetSlipEvent(ctx, strings.TrimSpace(id))
}

// CreateSlipEvent validates and inserts a new persisted SLIP event definition.
func (s *Service) CreateSlipEvent(ctx context.Context, def SlipEventDefinition) (*SlipEventDefinition, error) {
	return s.upsertSlipEventLocked(ctx, def, false)
}

// UpdateSlipEvent validates and updates an existing persisted SLIP event definition.
func (s *Service) UpdateSlipEvent(ctx context.Context, id string, def SlipEventDefinition) (*SlipEventDefinition, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: slip event id is required", ErrInvalidSlipConfig)
	}
	if def.ID != "" && !strings.EqualFold(strings.TrimSpace(def.ID), id) {
		return nil, fmt.Errorf("%w: slip event id mismatch", ErrInvalidSlipConfig)
	}
	def.ID = id
	return s.upsertSlipEventLocked(ctx, def, true)
}

func (s *Service) upsertSlipEventLocked(ctx context.Context, def SlipEventDefinition, mustExist bool) (*SlipEventDefinition, error) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	existing, err := s.Store.ListSlipEvents(ctx)
	if err != nil {
		return nil, err
	}
	targetID := strings.TrimSpace(def.ID)
	merged := make([]SlipEventDefinition, 0, len(existing)+1)
	foundIdx := -1
	for _, cur := range existing {
		if strings.EqualFold(cur.ID, targetID) {
			foundIdx = len(merged)
			def.ID = cur.ID
			merged = append(merged, def)
			continue
		}
		merged = append(merged, cur)
	}
	if !mustExist && foundIdx >= 0 {
		return nil, ErrAlreadyExists
	}
	if mustExist && foundIdx < 0 {
		return nil, ErrNotFound
	}
	if foundIdx < 0 {
		foundIdx = len(merged)
		merged = append(merged, def)
	}
	validated, err := ValidateSlipEventDefinitions(merged)
	if err != nil {
		return nil, err
	}
	saved := validated[foundIdx]
	if err := s.Store.UpsertSlipEvent(ctx, saved); err != nil {
		return nil, err
	}
	all, err := s.Store.ListSlipEvents(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.SetSlipEvents(all); err != nil {
		return nil, err
	}
	return &saved, nil
}

// RemoveSlipEvent deletes a persisted SLIP event definition and refreshes memory immediately.
func (s *Service) RemoveSlipEvent(ctx context.Context, id string) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	target := strings.TrimSpace(id)
	if target == "" {
		return ErrNotFound
	}
	existing, err := s.Store.ListSlipEvents(ctx)
	if err != nil {
		return err
	}
	storedID := ""
	for _, cur := range existing {
		if strings.EqualFold(cur.ID, target) {
			storedID = cur.ID
			break
		}
	}
	if storedID == "" {
		return ErrNotFound
	}
	if err := s.Store.DeleteSlipEvent(ctx, storedID); err != nil {
		return err
	}
	all, err := s.Store.ListSlipEvents(ctx)
	if err != nil {
		return err
	}
	return s.SetSlipEvents(all)
}

// UpdateCategory validates and updates mutable fields on a BBS category.
func (s *Service) UpdateCategory(ctx context.Context, id uuid.UUID, input UpdateCategoryInput) (*Category, error) {
	if input.Name != nil {
		trimmed := SanitizeSingleLine(*input.Name)
		if trimmed == "" || utf8.RuneCountInString(trimmed) > 64 {
			return nil, ErrInvalidCategoryName
		}
		input.Name = &trimmed
	}
	if input.Slug != nil {
		trimmed := strings.ToLower(SanitizeSingleLine(*input.Slug))
		if trimmed == "" || utf8.RuneCountInString(trimmed) > 64 {
			return nil, fmt.Errorf("%w: invalid slug", ErrInvalidInput)
		}
		input.Slug = &trimmed
	}
	if input.Description != nil {
		trimmed := SanitizeSingleLine(*input.Description)
		if utf8.RuneCountInString(trimmed) > 256 {
			return nil, fmt.Errorf("%w: description too long", ErrInvalidInput)
		}
		input.Description = &trimmed
	}
	cat, err := s.GetCategory(ctx, id)
	if err != nil {
		return nil, err
	}
	p, err := s.Registry.Get(cat.ProviderKey)
	if err != nil {
		return nil, err
	}
	return p.UpdateCategory(ctx, id, input)
}

// ListBoardMounts lists Discord channel mounts matching filter.
func (s *Service) ListBoardMounts(ctx context.Context, filter BoardMountFilter) ([]BoardMount, error) {
	return s.Store.ListBoardMounts(ctx, filter)
}

// GetStats returns aggregate operational counts for the BBS admin dashboard.
func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	return s.Store.GetBBSStats(ctx)
}

// InspectReportForAdmin loads full moderation audit details for a report (for RBAC-authorized admins).
func (s *Service) InspectReportForAdmin(ctx context.Context, reportID uuid.UUID) (*Report, *AuditPostInfo, error) {
	rep, err := s.Store.GetReport(ctx, reportID)
	if err != nil {
		return nil, nil, err
	}
	audit, err := s.Store.GetAuditPostInfo(ctx, rep.PostID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, nil, err
	}
	return rep, audit, nil
}

// ResolveReportForAdmin updates a report status and resolution metadata via the admin API.
func (s *Service) ResolveReportForAdmin(ctx context.Context, reportID uuid.UUID, status ReportStatus, handledBy snowflake.ID, resolutionNote string) (*Report, error) {
	if status != ReportStatusResolved && status != ReportStatusDismissed {
		return nil, fmt.Errorf("%w: status must be resolved or dismissed", ErrInvalidInput)
	}
	rep, err := s.Store.GetReport(ctx, reportID)
	if err != nil {
		return nil, err
	}
	rep.Status = status
	if handledBy != 0 {
		hb := handledBy
		rep.HandledBy = &hb
	}
	rep.ResolutionNote = SanitizeSingleLine(resolutionNote)
	if err := s.Store.UpdateReport(ctx, rep); err != nil {
		return nil, err
	}
	return s.Store.GetReport(ctx, reportID)
}

// AbonePostByAdmin soft-deletes a post from either local or global scope for RBAC-authorized admins.
func (s *Service) AbonePostByAdmin(ctx context.Context, postID uuid.UUID, reason string) (*Post, *Thread, error) {
	post, err := s.GetPost(ctx, postID)
	if err != nil {
		return nil, nil, err
	}
	thread, err := s.GetThread(ctx, post.ThreadID)
	if err != nil {
		return nil, nil, err
	}
	p, err := s.Registry.Get(thread.ProviderKey)
	if err != nil {
		return nil, nil, err
	}
	aboned, err := p.AbonePost(ctx, postID, SanitizeSingleLine(reason))
	if err != nil {
		return nil, nil, err
	}
	return aboned, thread, nil
}

// ArchiveThreadByAdmin archives a thread across its mirrors for RBAC-authorized admins.
func (s *Service) ArchiveThreadByAdmin(ctx context.Context, threadID uuid.UUID) (*Board, *Thread, error) {
	thread, err := s.GetThread(ctx, threadID)
	if err != nil {
		return nil, nil, err
	}
	board, err := s.GetBoard(ctx, thread.BoardID)
	if err != nil {
		return nil, nil, err
	}
	p, err := s.Registry.Get(board.ProviderKey)
	if err != nil {
		return nil, nil, err
	}
	archived, err := p.ArchiveThread(ctx, threadID)
	if err != nil {
		return nil, nil, err
	}
	return board, archived, nil
}

// CreateBanByAdmin validates and persists a local or global BBS ban via the admin API.
func (s *Service) CreateBanByAdmin(ctx context.Context, input AdminBanInput) (*Ban, error) {
	if input.DurationHours < 0 || input.DurationHours > MaxBanHours {
		return nil, fmt.Errorf("%w: duration_hours out of range", ErrInvalidInput)
	}
	var scope Scope
	switch input.Scope {
	case ScopeLocal, ScopeGlobal:
		scope = input.Scope
	default:
		return nil, fmt.Errorf("%w: scope must be local or global", ErrInvalidInput)
	}

	var guildIDPtr *snowflake.ID
	var guildIDVal snowflake.ID
	if scope == ScopeLocal {
		if input.GuildID == 0 && input.PostID != nil {
			audit, err := s.Store.GetAuditPostInfo(ctx, *input.PostID)
			if err != nil {
				return nil, err
			}
			if audit.GuildID != 0 {
				input.GuildID = audit.GuildID
			} else if th, thErr := s.GetThread(ctx, audit.Post.ThreadID); thErr == nil {
				if bd, bdErr := s.GetBoard(ctx, th.BoardID); bdErr == nil && bd.Scope == ScopeLocal && bd.GuildID != nil {
					input.GuildID = *bd.GuildID
				}
			}
		}
		if input.GuildID == 0 {
			return nil, ErrGuildIDRequired
		}
		guildIDVal = input.GuildID
		guildIDPtr = &guildIDVal
	}

	targetPosterHash := strings.TrimSpace(input.TargetPosterHash)
	var targetUserID *snowflake.ID
	if input.TargetUserID != nil && *input.TargetUserID != 0 {
		uid := *input.TargetUserID
		targetUserID = &uid
	}

	if input.PostID != nil {
		audit, err := s.Store.GetAuditPostInfo(ctx, *input.PostID)
		if err != nil {
			return nil, err
		}
		if targetPosterHash == "" {
			targetPosterHash = audit.Post.PosterHash
		}
		if targetUserID == nil && audit.AuthorUserID != 0 {
			uid := audit.AuthorUserID
			targetUserID = &uid
		}
	}

	if targetPosterHash == "" && targetUserID != nil && s.Identity != nil {
		targetPosterHash = s.Identity.GeneratePosterHash(scope, guildIDVal, *targetUserID)
	}
	if targetPosterHash == "" && targetUserID == nil {
		return nil, fmt.Errorf("%w: post_id, target_user_id, or target_poster_hash is required", ErrInvalidInput)
	}

	var expiresAt *time.Time
	if input.DurationHours > 0 {
		t := time.Now().Add(time.Duration(input.DurationHours) * time.Hour)
		expiresAt = &t
	}
	ban := &Ban{
		Scope:        scope,
		GuildID:      guildIDPtr,
		PosterHash:   targetPosterHash,
		TargetUserID: targetUserID,
		Reason:       SanitizeSingleLine(input.Reason),
		CreatedBy:    input.ActorID,
		ExpiresAt:    expiresAt,
	}
	if err := s.Store.CreateBan(ctx, ban); err != nil {
		return nil, err
	}
	return ban, nil
}

// DeleteBan removes an active BBS ban by ID.
func (s *Service) DeleteBan(ctx context.Context, banID uuid.UUID) error {
	return s.Store.DeleteBan(ctx, banID)
}
