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
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
)

func (s *Server) requireBBSSvc(w http.ResponseWriter) bool {
	if s.bbsSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "bbs_unavailable", "BBS system is not initialized")
		return false
	}
	return true
}

func writeBBSError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, bbs.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, bbs.ErrAlreadyExists):
		writeError(w, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, bbs.ErrUnauthorized):
		writeError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, bbs.ErrUnsupportedOperation):
		writeError(w, http.StatusBadRequest, "unsupported_operation", err.Error())
	case errors.Is(err, bbs.ErrInvalidInput),
		errors.Is(err, bbs.ErrInvalidCapConfig),
		errors.Is(err, bbs.ErrInvalidSlipConfig),
		errors.Is(err, bbs.ErrInvalidCategoryName),
		errors.Is(err, bbs.ErrInvalidBoardName),
		errors.Is(err, bbs.ErrInvalidBoardRules),
		errors.Is(err, bbs.ErrInvalidNameLength),
		errors.Is(err, bbs.ErrGuildIDRequired),
		errors.Is(err, bbs.ErrGuildNotEnabled),
		errors.Is(err, bbs.ErrBoardScopeMismatch),
		errors.Is(err, bbs.ErrBoardProviderMismatch),
		errors.Is(err, bbs.ErrThreadArchived),
		errors.Is(err, bbs.ErrMaxResReached):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "bbs_error", "internal BBS error")
	}
}

func parseBBSUUID(raw string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(raw))
}

func parseBBSSnowflake(raw string) (snowflake.ID, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil || v == 0 {
		return 0, errors.New("invalid discord snowflake id")
	}
	return snowflake.ID(v), nil
}

func parseBBSLimit(raw string, def, max int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 || v > max {
		return 0, errors.New("limit out of range")
	}
	return v, nil
}

func parseBBSNonNegInt(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return 0, errors.New("invalid non-negative integer")
	}
	return v, nil
}

func optSnowflakePtr(id *snowflake.ID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

func optUUIDPtr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

func toBBSCategoryDTO(c bbs.Category) BBSCategoryDTO {
	return BBSCategoryDTO{
		ID:          c.ID.String(),
		Scope:       string(c.Scope),
		GuildID:     optSnowflakePtr(c.GuildID),
		ProviderKey: c.ProviderKey,
		Slug:        c.Slug,
		Name:        c.Name,
		Description: c.Description,
		SortOrder:   c.SortOrder,
		CreatedAt:   c.CreatedAt,
	}
}

func toBBSBoardRulesDTO(r bbs.BoardRules) BBSBoardRulesDTO {
	return BBSBoardRulesDTO{
		DefaultName:        r.DefaultName,
		HideID:             r.HideID,
		SLIPLevel:          int(r.SLIP),
		MaxRes:             r.MaxRes,
		DatOchiIdleHours:   r.DatOchiIdleHours,
		DatOchiLowResHours: r.DatOchiLowResHours,
	}
}

func toBBSBoardDTO(b bbs.Board) BBSBoardDTO {
	return BBSBoardDTO{
		ID:          b.ID.String(),
		CategoryID:  b.CategoryID.String(),
		Scope:       string(b.Scope),
		GuildID:     optSnowflakePtr(b.GuildID),
		ProviderKey: b.ProviderKey,
		Slug:        b.Slug,
		Name:        b.Name,
		Description: b.Description,
		Rules:       toBBSBoardRulesDTO(b.Rules),
		Locked:      b.Locked,
		CreatedAt:   b.CreatedAt,
	}
}

func toBBSBoardMountDTO(m bbs.BoardMount) BBSBoardMountDTO {
	return BBSBoardMountDTO{
		ID:        m.ID.String(),
		BoardID:   m.BoardID.String(),
		GuildID:   m.GuildID.String(),
		ChannelID: m.ChannelID.String(),
		Locale:    m.Locale,
		IsForum:   m.IsForum,
		Active:    m.Active,
		CreatedAt: m.CreatedAt,
	}
}

func toBBSThreadDTO(t bbs.Thread) BBSThreadDTO {
	return BBSThreadDTO{
		ID:              t.ID.String(),
		BoardID:         t.BoardID.String(),
		ProviderKey:     t.ProviderKey,
		ExternalID:      t.ExternalID,
		Title:           t.Title,
		Status:          string(t.Status),
		ResCount:        t.ResCount,
		Rules:           toBBSBoardRulesDTO(t.Rules),
		OwnerAnonID:     t.OwnerAnonID,
		OwnerPosterHash: t.OwnerPosterHash,
		BumpedAt:        t.BumpedAt,
		ArchivedAt:      t.ArchivedAt,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}

func toBBSPostDTO(p bbs.Post) BBSPostDTO {
	return BBSPostDTO{
		ID:          p.ID.String(),
		ThreadID:    p.ThreadID.String(),
		ResNumber:   p.ResNumber,
		Name:        p.Name,
		Trip:        p.Trip,
		Cap:         p.Cap,
		SlipText:    p.SlipText,
		AnonID:      p.AnonID,
		PosterHash:  p.PosterHash,
		IsOP:        p.IsOP,
		Body:        p.Body,
		Abone:       p.Abone,
		AboneReason: p.AboneReason,
		CreatedAt:   p.CreatedAt,
	}
}

func toBBSReportDTO(r bbs.Report) BBSReportDTO {
	return BBSReportDTO{
		ID:             r.ID.String(),
		Scope:          string(r.Scope),
		GuildID:        r.GuildID.String(),
		PostID:         r.PostID.String(),
		ThreadID:       r.ThreadID.String(),
		BoardID:        r.BoardID.String(),
		ReporterUserID: r.ReporterUserID.String(),
		Reason:         r.Reason,
		Status:         string(r.Status),
		BanID:          optUUIDPtr(r.BanID),
		HandledBy:      optSnowflakePtr(r.HandledBy),
		ResolutionNote: r.ResolutionNote,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func toBBSBanDTO(b bbs.Ban) BBSBanDTO {
	return BBSBanDTO{
		ID:           b.ID.String(),
		Scope:        string(b.Scope),
		GuildID:      optSnowflakePtr(b.GuildID),
		PosterHash:   b.PosterHash,
		TargetUserID: optSnowflakePtr(b.TargetUserID),
		Reason:       b.Reason,
		CreatedBy:    b.CreatedBy.String(),
		ExpiresAt:    b.ExpiresAt,
		CreatedAt:    b.CreatedAt,
	}
}

func toBBSCapDTO(c bbs.CapDefinition) BBSCapDTO {
	users := make([]string, 0, len(c.Users))
	for _, u := range c.Users {
		users = append(users, u.String())
	}
	return BBSCapDTO{
		ID:             c.ID,
		Name:           c.Name,
		Description:    c.Description,
		GrantOperators: c.GrantOperators,
		Users:          users,
	}
}

func toBBSSlipEventDTO(ev bbs.SlipEventDefinition) BBSSlipEventDTO {
	return BBSSlipEventDTO{
		ID:        ev.ID,
		Label:     ev.Label,
		StartDate: ev.StartDate,
		EndDate:   ev.EndDate,
		Mode:      string(ev.Mode),
	}
}

func (s *Server) handleBBSStats(w http.ResponseWriter, r *http.Request, _ authContext) {
	if s.bbsSvc == nil {
		writeJSON(w, http.StatusOK, BBSStatsResponse{Available: false})
		return
	}
	st, err := s.bbsSvc.GetStats(r.Context())
	if err != nil {
		writeBBSError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, BBSStatsResponse{
		Available:     true,
		Categories:    st.Categories,
		Boards:        st.Boards,
		ActiveThreads: st.ActiveThreads,
		OpenReports:   st.OpenReports,
		ActiveBans:    st.ActiveBans,
	})
}

func (s *Server) handleBBSListCategories(w http.ResponseWriter, r *http.Request, _ authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	q := r.URL.Query()
	var filter bbs.CategoryFilter
	if rawScope := strings.TrimSpace(q.Get("scope")); rawScope != "" {
		sc := bbs.Scope(rawScope)
		if sc != bbs.ScopeLocal && sc != bbs.ScopeGlobal {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid scope")
			return
		}
		filter.Scope = &sc
	}
	if rawGuild := strings.TrimSpace(q.Get("guild_id")); rawGuild != "" {
		gid, err := parseBBSSnowflake(rawGuild)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid guild_id")
			return
		}
		filter.GuildID = &gid
	}
	cats, err := s.bbsSvc.ListCategories(r.Context(), filter)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	out := make([]BBSCategoryDTO, 0, len(cats))
	for _, c := range cats {
		out = append(out, toBBSCategoryDTO(c))
	}
	writeJSON(w, http.StatusOK, BBSCategoriesResponse{Categories: out})
}

func (s *Server) handleBBSCreateCategory(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	var req CreateBBSCategoryRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	var guildIDPtr *snowflake.ID
	if strings.TrimSpace(req.GuildID) != "" {
		gid, err := parseBBSSnowflake(req.GuildID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid guild_id")
			return
		}
		guildIDPtr = &gid
	}
	created, err := s.bbsSvc.CreateCategory(r.Context(), bbs.CreateCategoryInput{
		Scope:       bbs.Scope(strings.TrimSpace(req.Scope)),
		GuildID:     guildIDPtr,
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
			"bbs.category.create", "bbs_category", "",
			"Failed to create BBS category", map[string]any{"name": req.Name, "error": err.Error()}, "error", s.clientRemoteAddr(r))
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.category.create", "bbs_category", created.ID.String(),
		"Created BBS category", map[string]any{"name": created.Name, "scope": created.Scope}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusCreated, toBBSCategoryDTO(*created))
}

func (s *Server) handleBBSUpdateCategory(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id, err := parseBBSUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid category id")
		return
	}
	var req UpdateBBSCategoryRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	updated, err := s.bbsSvc.UpdateCategory(r.Context(), id, bbs.UpdateCategoryInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.category.update", "bbs_category", id.String(),
		"Updated BBS category", map[string]any{"name": updated.Name}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, toBBSCategoryDTO(*updated))
}

func (s *Server) handleBBSDeleteCategory(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id, err := parseBBSUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid category id")
		return
	}
	if err := s.bbsSvc.DeleteCategory(r.Context(), id); err != nil {
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.category.delete", "bbs_category", id.String(),
		"Deleted BBS category", nil, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id.String()})
}

func (s *Server) handleBBSListBoards(w http.ResponseWriter, r *http.Request, _ authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	q := r.URL.Query()
	var filter bbs.BoardFilter
	if rawScope := strings.TrimSpace(q.Get("scope")); rawScope != "" {
		sc := bbs.Scope(rawScope)
		if sc != bbs.ScopeLocal && sc != bbs.ScopeGlobal {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid scope")
			return
		}
		filter.Scope = &sc
	}
	if rawGuild := strings.TrimSpace(q.Get("guild_id")); rawGuild != "" {
		gid, err := parseBBSSnowflake(rawGuild)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid guild_id")
			return
		}
		filter.GuildID = &gid
	}
	if rawCat := strings.TrimSpace(q.Get("category_id")); rawCat != "" {
		cid, err := parseBBSUUID(rawCat)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid category_id")
			return
		}
		filter.CategoryID = &cid
	}
	boards, err := s.bbsSvc.ListBoards(r.Context(), filter)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	out := make([]BBSBoardDTO, 0, len(boards))
	for _, b := range boards {
		out = append(out, toBBSBoardDTO(b))
	}
	writeJSON(w, http.StatusOK, BBSBoardsResponse{Boards: out})
}

func (s *Server) handleBBSCreateBoard(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	var req CreateBBSBoardRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	catID, err := parseBBSUUID(req.CategoryID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid category_id")
		return
	}
	cat, err := s.bbsSvc.GetCategory(r.Context(), catID)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	scope := cat.Scope
	if strings.TrimSpace(req.Scope) != "" {
		scope = bbs.Scope(strings.TrimSpace(req.Scope))
	}
	guildIDPtr := cat.GuildID
	if strings.TrimSpace(req.GuildID) != "" {
		gid, gErr := parseBBSSnowflake(req.GuildID)
		if gErr != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid guild_id")
			return
		}
		guildIDPtr = &gid
	}
	created, err := s.bbsSvc.CreateBoard(r.Context(), bbs.CreateBoardInput{
		CategoryID:  catID,
		Scope:       scope,
		GuildID:     guildIDPtr,
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		Rules: bbs.BoardRules{
			DefaultName:        req.Rules.DefaultName,
			HideID:             req.Rules.HideID,
			SLIP:               bbs.SLIPLevel(req.Rules.SLIPLevel),
			MaxRes:             req.Rules.MaxRes,
			DatOchiIdleHours:   req.Rules.DatOchiIdleHours,
			DatOchiLowResHours: req.Rules.DatOchiLowResHours,
		},
	})
	if err != nil {
		s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
			"bbs.board.create", "bbs_board", "",
			"Failed to create BBS board", map[string]any{"name": req.Name, "error": err.Error()}, "error", s.clientRemoteAddr(r))
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.board.create", "bbs_board", created.ID.String(),
		"Created BBS board", map[string]any{"name": created.Name, "category_id": created.CategoryID.String()}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusCreated, toBBSBoardDTO(*created))
}

func (s *Server) handleBBSUpdateBoard(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id, err := parseBBSUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid board id")
		return
	}
	var req UpdateBBSBoardRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	var slipPtr *bbs.SLIPLevel
	if req.SLIPLevel != nil {
		lv := bbs.SLIPLevel(*req.SLIPLevel)
		slipPtr = &lv
	}
	updated, err := s.bbsSvc.UpdateBoard(r.Context(), id, bbs.UpdateBoardInput{
		Name:        req.Name,
		Description: req.Description,
		DefaultName: req.DefaultName,
		HideID:      req.HideID,
		SLIP:        slipPtr,
		Locked:      req.Locked,
	})
	if err != nil {
		writeBBSError(w, err)
		return
	}
	if s.bbsSync != nil && s.client != nil {
		if mounts, mErr := s.bbsSvc.ListBoardMounts(r.Context(), bbs.BoardMountFilter{BoardID: &updated.ID}); mErr == nil {
			for _, m := range mounts {
				if m.Active {
					s.bbsSync.RefreshBoardGuide(r.Context(), s.client, m.GuildID, updated.ID, "")
				}
			}
		}
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.board.update", "bbs_board", id.String(),
		"Updated BBS board", map[string]any{"name": updated.Name}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, toBBSBoardDTO(*updated))
}

func (s *Server) handleBBSDeleteBoard(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id, err := parseBBSUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid board id")
		return
	}
	if err := s.bbsSvc.DeleteBoard(r.Context(), id); err != nil {
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.board.delete", "bbs_board", id.String(),
		"Deleted BBS board", nil, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id.String()})
}

func (s *Server) handleBBSListBoardMounts(w http.ResponseWriter, r *http.Request, _ authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id, err := parseBBSUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid board id")
		return
	}
	mounts, err := s.bbsSvc.ListBoardMounts(r.Context(), bbs.BoardMountFilter{BoardID: &id})
	if err != nil {
		writeBBSError(w, err)
		return
	}
	out := make([]BBSBoardMountDTO, 0, len(mounts))
	for _, m := range mounts {
		out = append(out, toBBSBoardMountDTO(m))
	}
	writeJSON(w, http.StatusOK, BBSBoardMountsResponse{Mounts: out})
}

func (s *Server) handleBBSListThreads(w http.ResponseWriter, r *http.Request, _ authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	q := r.URL.Query()
	var filter bbs.ThreadFilter
	if rawBoard := strings.TrimSpace(q.Get("board_id")); rawBoard != "" {
		bid, err := parseBBSUUID(rawBoard)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid board_id")
			return
		}
		filter.BoardID = bid
	}
	if rawStatus := strings.TrimSpace(q.Get("status")); rawStatus != "" {
		st := bbs.ThreadStatus(rawStatus)
		if st != bbs.ThreadStatusActive && st != bbs.ThreadStatusArchived {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid thread status")
			return
		}
		filter.Status = &st
	}
	filter.Query = strings.TrimSpace(q.Get("q"))
	limit, err := parseBBSLimit(q.Get("limit"), 50, 200)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid limit")
		return
	}
	offset, err := parseBBSNonNegInt(q.Get("offset"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid offset")
		return
	}
	filter.Limit = limit
	filter.Offset = offset

	threads, err := s.bbsSvc.ListThreads(r.Context(), filter)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	out := make([]BBSThreadDTO, 0, len(threads))
	for _, th := range threads {
		out = append(out, toBBSThreadDTO(th))
	}
	writeJSON(w, http.StatusOK, BBSThreadsResponse{Threads: out})
}

func (s *Server) handleBBSGetThread(w http.ResponseWriter, r *http.Request, _ authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id, err := parseBBSUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid thread id")
		return
	}
	q := r.URL.Query()
	limit, err := parseBBSLimit(q.Get("limit"), 100, 500)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid limit")
		return
	}
	offset, err := parseBBSNonNegInt(q.Get("offset"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid offset")
		return
	}
	th, err := s.bbsSvc.GetThread(r.Context(), id)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	posts, err := s.bbsSvc.ListPosts(r.Context(), id, offset, limit)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	postDTOs := make([]BBSPostDTO, 0, len(posts))
	for _, p := range posts {
		postDTOs = append(postDTOs, toBBSPostDTO(p))
	}
	writeJSON(w, http.StatusOK, BBSThreadDetailResponse{
		Thread: toBBSThreadDTO(*th),
		Posts:  postDTOs,
	})
}

func (s *Server) handleBBSArchiveThread(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id, err := parseBBSUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid thread id")
		return
	}
	board, archived, err := s.bbsSvc.ArchiveThreadByAdmin(r.Context(), id)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	if s.bbsSync != nil && s.client != nil {
		s.bbsSync.SyncArchive(r.Context(), s.client, *board, *archived, false)
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.thread.archive", "bbs_thread", id.String(),
		"Archived BBS thread", map[string]any{"board_id": board.ID.String()}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, BBSArchiveThreadResponse{
		Thread: toBBSThreadDTO(*archived),
	})
}

func (s *Server) handleBBSAbonePost(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id, err := parseBBSUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid post id")
		return
	}
	var req AboneBBSPostRequest
	if err := decodeJSONBody(w, r, &req, true); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	aboned, thread, err := s.bbsSvc.AbonePostByAdmin(r.Context(), id, req.Reason)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	resp := BBSAbonePostResponse{
		Post: toBBSPostDTO(*aboned),
	}
	if s.bbsSync != nil && s.client != nil {
		if syncErr := s.bbsSync.SyncAbone(r.Context(), s.client, *aboned, *thread); syncErr != nil {
			resp.SyncError = syncErr.Error()
		}
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.post.abone", "bbs_post", id.String(),
		"Soft-deleted (abone) BBS post", map[string]any{"thread_id": thread.ID.String(), "reason": aboned.AboneReason, "sync_error": resp.SyncError}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleBBSListReports(w http.ResponseWriter, r *http.Request, _ authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	q := r.URL.Query()
	var filter bbs.ReportFilter
	if rawScope := strings.TrimSpace(q.Get("scope")); rawScope != "" {
		sc := bbs.Scope(rawScope)
		if sc != bbs.ScopeLocal && sc != bbs.ScopeGlobal {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid scope")
			return
		}
		filter.Scope = &sc
	}
	if rawStatus := strings.TrimSpace(q.Get("status")); rawStatus != "" {
		st := bbs.ReportStatus(rawStatus)
		if st != bbs.ReportStatusOpen && st != bbs.ReportStatusResolved && st != bbs.ReportStatusDismissed {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid report status")
			return
		}
		filter.Status = &st
	}
	limit, err := parseBBSLimit(q.Get("limit"), 50, 200)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid limit")
		return
	}
	filter.Limit = limit

	reps, err := s.bbsSvc.Store.ListReports(r.Context(), filter)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	out := make([]BBSReportDTO, 0, len(reps))
	for _, rep := range reps {
		out = append(out, toBBSReportDTO(rep))
	}
	writeJSON(w, http.StatusOK, BBSReportsResponse{Reports: out})
}

func (s *Server) handleBBSInspectReport(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id, err := parseBBSUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid report id")
		return
	}
	rep, auditInfo, err := s.bbsSvc.InspectReportForAdmin(r.Context(), id)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	resp := BBSReportInspectResponse{
		Report: toBBSReportDTO(*rep),
	}
	if auditInfo != nil {
		pDTO := toBBSPostDTO(auditInfo.Post)
		resp.Post = &pDTO
		if auditInfo.AuthorUserID != 0 {
			resp.AuthorUserID = auditInfo.AuthorUserID.String()
		}
		if auditInfo.GuildID != 0 {
			resp.PostGuildID = auditInfo.GuildID.String()
		}
		resp.PosterHash = auditInfo.Post.PosterHash
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.report.inspect", "bbs_report", id.String(),
		"Inspected BBS report author attribution", map[string]any{"post_id": rep.PostID.String()}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleBBSResolveReport(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id, err := parseBBSUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid report id")
		return
	}
	var req ResolveBBSReportRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	updated, err := s.bbsSvc.ResolveReportForAdmin(
		r.Context(),
		id,
		bbs.ReportStatus(strings.TrimSpace(req.Status)),
		auth.Session.UserID,
		req.ResolutionNote,
	)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.report.resolve", "bbs_report", id.String(),
		"Resolved BBS report", map[string]any{"status": updated.Status, "note": updated.ResolutionNote}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, toBBSReportDTO(*updated))
}

func (s *Server) handleBBSListBans(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	q := r.URL.Query()
	var filter bbs.BanFilter
	if rawScope := strings.TrimSpace(q.Get("scope")); rawScope != "" {
		sc := bbs.Scope(rawScope)
		if sc != bbs.ScopeLocal && sc != bbs.ScopeGlobal {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid scope")
			return
		}
		filter.Scope = &sc
	}
	if rawGuild := strings.TrimSpace(q.Get("guild_id")); rawGuild != "" {
		gid, err := parseBBSSnowflake(rawGuild)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid guild_id")
			return
		}
		filter.GuildID = &gid
	}
	limit, err := parseBBSLimit(q.Get("limit"), 100, 500)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid limit")
		return
	}
	filter.Limit = limit

	bans, err := s.bbsSvc.Store.ListBans(r.Context(), filter)
	if err != nil {
		writeBBSError(w, err)
		return
	}
	canSeeUserID := HasPermission(auth.Principal.Permissions, PermBBSModerate)
	out := make([]BBSBanDTO, 0, len(bans))
	for _, b := range bans {
		dto := toBBSBanDTO(b)
		if !canSeeUserID {
			dto.TargetUserID = nil
		}
		out = append(out, dto)
	}
	writeJSON(w, http.StatusOK, BBSBansResponse{Bans: out})
}

func (s *Server) handleBBSCreateBan(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	var req CreateBBSBanRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	input := bbs.AdminBanInput{
		Scope:            bbs.Scope(strings.TrimSpace(req.Scope)),
		TargetPosterHash: req.TargetPosterHash,
		Reason:           req.Reason,
		DurationHours:    req.DurationHours,
		ActorID:          auth.Session.UserID,
	}
	if strings.TrimSpace(req.GuildID) != "" {
		gid, err := parseBBSSnowflake(req.GuildID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid guild_id")
			return
		}
		input.GuildID = gid
	}
	if strings.TrimSpace(req.PostID) != "" {
		pid, err := parseBBSUUID(req.PostID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid post_id")
			return
		}
		input.PostID = &pid
	}
	if strings.TrimSpace(req.TargetUserID) != "" {
		uid, err := parseBBSSnowflake(req.TargetUserID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid target_user_id")
			return
		}
		input.TargetUserID = &uid
	}
	created, err := s.bbsSvc.CreateBanByAdmin(r.Context(), input)
	if err != nil {
		s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
			"bbs.ban.create", "bbs_ban", "",
			"Failed to create BBS ban", map[string]any{"scope": req.Scope, "error": err.Error()}, "error", s.clientRemoteAddr(r))
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.ban.create", "bbs_ban", created.ID.String(),
		"Created BBS ban", map[string]any{"scope": created.Scope, "poster_hash": created.PosterHash, "reason": created.Reason}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusCreated, toBBSBanDTO(*created))
}

func (s *Server) handleBBSDeleteBan(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id, err := parseBBSUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid ban id")
		return
	}
	if err := s.bbsSvc.DeleteBan(r.Context(), id); err != nil {
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.ban.delete", "bbs_ban", id.String(),
		"Deleted BBS ban", nil, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id.String()})
}

func parseCapUsers(raw []string) ([]snowflake.ID, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]snowflake.ID, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		uid, err := parseBBSSnowflake(s)
		if err != nil {
			return nil, err
		}
		out = append(out, uid)
	}
	return out, nil
}

func (s *Server) handleBBSListCaps(w http.ResponseWriter, r *http.Request, _ authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	defs, err := s.bbsSvc.ListAdminCaps(r.Context())
	if err != nil {
		writeBBSError(w, err)
		return
	}
	out := make([]BBSCapDTO, 0, len(defs))
	for _, d := range defs {
		out = append(out, toBBSCapDTO(d))
	}
	writeJSON(w, http.StatusOK, BBSCapsResponse{Caps: out})
}

func (s *Server) handleBBSCreateCap(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	var req SaveBBSCapRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	users, err := parseCapUsers(req.Users)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid user id in cap users")
		return
	}
	saved, err := s.bbsSvc.CreateAdminCap(r.Context(), bbs.CapDefinition{
		ID:             req.ID,
		Name:           req.Name,
		Description:    req.Description,
		GrantOperators: req.GrantOperators,
		Users:          users,
	})
	if err != nil {
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.cap.create", "bbs_admin_cap", saved.ID,
		"Created BBS admin cap", map[string]any{"name": saved.Name}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusCreated, toBBSCapDTO(*saved))
}

func (s *Server) handleBBSUpdateCap(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_id", "cap id is required")
		return
	}
	var req SaveBBSCapRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	users, err := parseCapUsers(req.Users)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid user id in cap users")
		return
	}
	saved, err := s.bbsSvc.UpdateAdminCap(r.Context(), id, bbs.CapDefinition{
		ID:             req.ID,
		Name:           req.Name,
		Description:    req.Description,
		GrantOperators: req.GrantOperators,
		Users:          users,
	})
	if err != nil {
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.cap.update", "bbs_admin_cap", saved.ID,
		"Updated BBS admin cap", map[string]any{"name": saved.Name}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, toBBSCapDTO(*saved))
}

func (s *Server) handleBBSDeleteCap(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_id", "cap id is required")
		return
	}
	if err := s.bbsSvc.RemoveAdminCap(r.Context(), id); err != nil {
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.cap.delete", "bbs_admin_cap", id,
		"Deleted BBS admin cap", nil, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}

func (s *Server) handleBBSListSlipEvents(w http.ResponseWriter, r *http.Request, _ authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	evs, err := s.bbsSvc.ListSlipEvents(r.Context())
	if err != nil {
		writeBBSError(w, err)
		return
	}
	out := make([]BBSSlipEventDTO, 0, len(evs))
	for _, ev := range evs {
		out = append(out, toBBSSlipEventDTO(ev))
	}
	writeJSON(w, http.StatusOK, BBSSlipEventsResponse{Events: out})
}

func (s *Server) handleBBSCreateSlipEvent(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	var req SaveBBSSlipEventRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	saved, err := s.bbsSvc.CreateSlipEvent(r.Context(), bbs.SlipEventDefinition{
		ID:        req.ID,
		Label:     req.Label,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Mode:      bbs.SlipEventMode(strings.TrimSpace(req.Mode)),
	})
	if err != nil {
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.slip_event.create", "bbs_slip_event", saved.ID,
		"Created BBS SLIP event", map[string]any{"label": saved.Label}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusCreated, toBBSSlipEventDTO(*saved))
}

func (s *Server) handleBBSUpdateSlipEvent(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_id", "slip event id is required")
		return
	}
	var req SaveBBSSlipEventRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	saved, err := s.bbsSvc.UpdateSlipEvent(r.Context(), id, bbs.SlipEventDefinition{
		ID:        req.ID,
		Label:     req.Label,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Mode:      bbs.SlipEventMode(strings.TrimSpace(req.Mode)),
	})
	if err != nil {
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.slip_event.update", "bbs_slip_event", saved.ID,
		"Updated BBS SLIP event", map[string]any{"label": saved.Label}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, toBBSSlipEventDTO(*saved))
}

func (s *Server) handleBBSDeleteSlipEvent(w http.ResponseWriter, r *http.Request, auth authContext) {
	if !s.requireBBSSvc(w) {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_id", "slip event id is required")
		return
	}
	if err := s.bbsSvc.RemoveSlipEvent(r.Context(), id); err != nil {
		writeBBSError(w, err)
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"bbs.slip_event.delete", "bbs_slip_event", id,
		"Deleted BBS SLIP event", nil, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}
