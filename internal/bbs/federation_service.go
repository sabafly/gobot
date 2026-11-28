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
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/uuidv7"
)

// SetEventSink registers an optional sink notified of native global-board mutations.
func (s *Service) SetEventSink(sink EventSink) {
	s.eventSinkMu.Lock()
	s.eventSink = sink
	s.eventSinkMu.Unlock()
}

func (s *Service) emitEvent(ev Event) {
	s.eventSinkMu.RLock()
	sink := s.eventSink
	s.eventSinkMu.RUnlock()
	if sink == nil {
		return
	}
	if ev.OccurredAt.IsZero() {
		ev.OccurredAt = time.Now()
	}
	sink.Publish(ev)
}

func (s *Service) isNativeGlobalBoard(board *Board) bool {
	if board == nil || board.Scope != ScopeGlobal {
		return false
	}
	return board.ProviderKey == "" || board.ProviderKey == s.Registry.DefaultKey()
}

func validateBasicThreadInput(title, rawName, rawBody string) (string, error) {
	cleanTitle := SanitizeSingleLine(title)
	if cleanTitle == "" || strings.TrimSpace(ClearFormat(cleanTitle)) == "" || utf8.RuneCountInString(cleanTitle) > MaxTitleRunes {
		return "", ErrInvalidTitleLength
	}
	if utf8.RuneCountInString(SanitizeSingleLine(rawName)) > MaxNameRunes {
		return "", ErrInvalidNameLength
	}
	cleanBody := strings.TrimSpace(SanitizeBody(rawBody))
	if cleanBody == "" || utf8.RuneCountInString(cleanBody) > MaxBodyRunes {
		return "", ErrInvalidBodyLength
	}
	return cleanTitle, nil
}

func validateBasicPostInput(rawName, rawBody string) error {
	if utf8.RuneCountInString(SanitizeSingleLine(rawName)) > MaxNameRunes {
		return ErrInvalidNameLength
	}
	cleanBody := strings.TrimSpace(SanitizeBody(rawBody))
	if cleanBody == "" || utf8.RuneCountInString(cleanBody) > MaxBodyRunes {
		return ErrInvalidBodyLength
	}
	return nil
}

// CreateThreadFromFederation creates a thread on a native global board on behalf of an
// OAuth-delegated user from a federated peer, applying Origin's consent, identity, bans, caps, and audit.
func (s *Service) CreateThreadFromFederation(
	ctx context.Context,
	peerID string,
	allowCaps bool,
	input CreateThreadInput,
) (*Board, *Thread, *Post, error) {
	if input.UserID == 0 {
		return nil, nil, nil, ErrUnauthorized
	}
	if _, err := s.CheckConsent(ctx, input.UserID); err != nil {
		return nil, nil, nil, err
	}
	title, err := validateBasicThreadInput(input.Title, input.RawName, input.RawBody)
	if err != nil {
		return nil, nil, nil, err
	}

	board, err := s.GetBoard(ctx, input.BoardID)
	if err != nil {
		return nil, nil, nil, err
	}
	if !s.isNativeGlobalBoard(board) {
		return nil, nil, nil, ErrNotFound
	}
	if board.Locked {
		return nil, nil, nil, ErrBoardLocked
	}
	p, err := s.Registry.Get(board.ProviderKey)
	if err != nil {
		return nil, nil, nil, err
	}
	if !p.Capabilities().CanCreateThread {
		return nil, nil, nil, ErrUnsupportedOperation
	}

	var cap *Cap
	if cleanCap := SanitizeSingleLine(input.CapID); cleanCap != "" && cleanCap != CapSelectNoneValue {
		if !allowCaps {
			return nil, nil, nil, ErrCapNotAuthorized
		}
		cap, err = s.ResolveUserCap(ctx, input.UserID, board.ID, cleanCap)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	parsed, err := s.Identity.ParsePostIdentity(
		board.ID,
		board.Scope,
		0,
		input.UserID,
		input.RawName,
		input.RawBody,
		board.Rules,
		true,
		cap,
		input.SlipContext,
		now,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	if parsed.CleanBody == "" || utf8.RuneCountInString(parsed.CleanBody) > MaxBodyRunes {
		return nil, nil, nil, ErrInvalidBodyLength
	}
	formattedBody := ClearFormat(parsed.CleanBody)
	if strings.TrimSpace(formattedBody) == "" {
		return nil, nil, nil, ErrInvalidBodyLength
	}
	if utf8.RuneCountInString(formattedBody) > MaxEscapedBodyRunes {
		return nil, nil, nil, ErrFormattedBodyTooLong
	}

	banned, err := s.Store.IsBanned(ctx, ScopeGlobal, 0, parsed.PosterHash, input.UserID, now)
	if err != nil {
		return nil, nil, nil, err
	}
	if banned {
		return nil, nil, nil, ErrBanned
	}

	if cap == nil && !s.checkThreadRateLimit(input.UserID) {
		return nil, nil, nil, ErrRateLimited
	}

	thread, post, err := p.CreateThread(ctx, CreateThreadRecord{
		BoardID: board.ID,
		Title:   title,
		Rules:   parsed.ThreadRules,
		ExtDat:  parsed.ExtDat,
		FirstPost: CreatePostRecord{
			Name:         parsed.DisplayName,
			Trip:         parsed.Trip,
			Cap:          parsed.CapName,
			SlipText:     parsed.SlipText,
			AnonID:       parsed.AnonID,
			PosterHash:   parsed.PosterHash,
			Body:         parsed.CleanBody,
			AuthorUserID: input.UserID,
		},
	})
	if err != nil {
		return nil, nil, nil, err
	}

	s.recordPostReferences(ctx, thread.ID, 1, parsed.CleanBody, parsed.ThreadRules.MaxRes)
	bCopy := *board
	tCopy := *thread
	pCopy := *post
	s.emitEvent(Event{
		Type:       EventThreadCreated,
		BoardID:    board.ID,
		ThreadID:   thread.ID,
		PostID:     post.ID,
		Board:      &bCopy,
		Thread:     &tCopy,
		Post:       &pCopy,
		OriginPeer: peerID,
		OccurredAt: now,
	})
	return board, thread, post, nil
}

// CreatePostFromFederation appends a reply on a native global board on behalf of an
// OAuth-delegated user from a federated peer, applying Origin's consent, identity, bans, caps, and audit.
func (s *Service) CreatePostFromFederation(
	ctx context.Context,
	peerID string,
	allowCaps bool,
	input CreatePostInput,
) (*Board, *Thread, *Post, []uint32, error) {
	if input.UserID == 0 {
		return nil, nil, nil, nil, ErrUnauthorized
	}
	if _, err := s.CheckConsent(ctx, input.UserID); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := validateBasicPostInput(input.RawName, input.RawBody); err != nil {
		return nil, nil, nil, nil, err
	}

	thread, err := s.GetThread(ctx, input.ThreadID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	board, err := s.GetBoard(ctx, thread.BoardID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if !s.isNativeGlobalBoard(board) {
		return nil, nil, nil, nil, ErrNotFound
	}
	if board.Locked {
		return nil, nil, nil, nil, ErrBoardLocked
	}
	p, err := s.Registry.Get(board.ProviderKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if !p.Capabilities().CanCreatePost {
		return nil, nil, nil, nil, ErrUnsupportedOperation
	}

	var cap *Cap
	if cleanCap := SanitizeSingleLine(input.CapID); cleanCap != "" && cleanCap != CapSelectNoneValue {
		if !allowCaps {
			return nil, nil, nil, nil, ErrCapNotAuthorized
		}
		cap, err = s.ResolveUserCap(ctx, input.UserID, board.ID, cleanCap)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	mu := s.threadMutex(thread.ID)
	mu.Lock()
	defer mu.Unlock()

	thread, err = p.GetThread(ctx, input.ThreadID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if thread.Status != ThreadStatusActive {
		return nil, nil, nil, nil, ErrThreadArchived
	}
	maxRes := thread.Rules.Normalize().MaxRes
	if thread.ResCount >= maxRes {
		return nil, nil, nil, nil, ErrMaxResReached
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	parsed, err := s.Identity.ParsePostIdentity(
		board.ID,
		board.Scope,
		0,
		input.UserID,
		input.RawName,
		input.RawBody,
		thread.Rules,
		false,
		cap,
		input.SlipContext,
		now,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if parsed.CleanBody == "" || utf8.RuneCountInString(parsed.CleanBody) > MaxBodyRunes {
		return nil, nil, nil, nil, ErrInvalidBodyLength
	}
	formattedBody := ClearFormat(parsed.CleanBody)
	if strings.TrimSpace(formattedBody) == "" {
		return nil, nil, nil, nil, ErrInvalidBodyLength
	}
	if utf8.RuneCountInString(formattedBody) > MaxEscapedBodyRunes {
		return nil, nil, nil, nil, ErrFormattedBodyTooLong
	}

	banned, err := s.Store.IsBanned(ctx, ScopeGlobal, 0, parsed.PosterHash, input.UserID, now)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if banned {
		return nil, nil, nil, nil, ErrBanned
	}

	if cap == nil && !s.checkPostRateLimit(input.UserID) {
		return nil, nil, nil, nil, ErrRateLimited
	}

	post, updatedThread, err := p.CreatePost(ctx, CreatePostRecord{
		ThreadID:     thread.ID,
		Name:         parsed.DisplayName,
		Trip:         parsed.Trip,
		Cap:          parsed.CapName,
		SlipText:     parsed.SlipText,
		AnonID:       parsed.AnonID,
		PosterHash:   parsed.PosterHash,
		Body:         parsed.CleanBody,
		AuthorUserID: input.UserID,
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}

	s.recordPostReferences(ctx, thread.ID, post.ResNumber, parsed.CleanBody, maxRes)
	backwardRefs, refErr := s.Store.ListReferencingRes(ctx, thread.ID, post.ResNumber)
	if refErr != nil {
		slog.Warn("bbs failed to list referencing res", "thread_id", thread.ID, "res_number", post.ResNumber, "err", refErr)
	}

	bCopy := *board
	tCopy := *updatedThread
	pCopy := *post
	s.emitEvent(Event{
		Type:         EventPostCreated,
		BoardID:      board.ID,
		ThreadID:     updatedThread.ID,
		PostID:       post.ID,
		Board:        &bCopy,
		Thread:       &tCopy,
		Post:         &pCopy,
		BackwardRefs: slices.Clone(backwardRefs),
		OriginPeer:   peerID,
		OccurredAt:   now,
	})
	if updatedThread.Status == ThreadStatusArchived {
		s.emitEvent(Event{
			Type:       EventThreadArchived,
			BoardID:    board.ID,
			ThreadID:   updatedThread.ID,
			Board:      &bCopy,
			Thread:     &tCopy,
			HitMaxRes:  post.ResNumber >= maxRes,
			OriginPeer: peerID,
			OccurredAt: now,
		})
	}
	return board, updatedThread, post, backwardRefs, nil
}

// CreateReportFromFederation records a user report on a native global board post via federation.
func (s *Service) CreateReportFromFederation(
	ctx context.Context,
	peerID string,
	guildID snowflake.ID,
	reporterID snowflake.ID,
	postID uuid.UUID,
	reason string,
) (*Report, error) {
	if reporterID == 0 {
		return nil, ErrUnauthorized
	}
	if _, err := s.CheckConsent(ctx, reporterID); err != nil {
		return nil, err
	}
	reason = SanitizeSingleLine(reason)
	if reason == "" {
		return nil, ErrReportReasonRequired
	}
	post, err := s.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	thread, err := s.GetThread(ctx, post.ThreadID)
	if err != nil {
		return nil, err
	}
	board, err := s.GetBoard(ctx, thread.BoardID)
	if err != nil {
		return nil, err
	}
	if !s.isNativeGlobalBoard(board) {
		return nil, ErrNotFound
	}
	now := time.Now()
	report := &Report{
		ID:             uuidv7.New(),
		Scope:          ScopeGlobal,
		OriginPeer:     strings.TrimSpace(peerID),
		GuildID:        guildID,
		PostID:         post.ID,
		ThreadID:       thread.ID,
		BoardID:        board.ID,
		ReporterUserID: reporterID,
		Reason:         reason,
		Status:         ReportStatusOpen,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.Store.CreateReport(ctx, report); err != nil {
		return nil, err
	}
	return report, nil
}

// ArchiveStaleThread archives a stale thread via its owning provider and emits a federation event for global boards.
func (s *Service) ArchiveStaleThread(ctx context.Context, threadID uuid.UUID) (*Board, *Thread, error) {
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
	if !p.Capabilities().CanModerate {
		return nil, nil, ErrUnsupportedOperation
	}
	archived, err := p.ArchiveThread(ctx, threadID)
	if err != nil {
		return nil, nil, err
	}
	if s.isNativeGlobalBoard(board) {
		bCopy := *board
		tCopy := *archived
		s.emitEvent(Event{
			Type:      EventThreadArchived,
			BoardID:   board.ID,
			ThreadID:  archived.ID,
			Board:     &bCopy,
			Thread:    &tCopy,
			HitMaxRes: archived.ResCount >= archived.Rules.Normalize().MaxRes,
		})
	}
	return board, archived, nil
}

// AbonePostSystem soft-deletes a post from automated system checks (e.g. media scan) and emits a federation event for global boards.
func (s *Service) AbonePostSystem(ctx context.Context, postID uuid.UUID, reason string) (*Post, *Thread, error) {
	post, err := s.GetPost(ctx, postID)
	if err != nil {
		return nil, nil, err
	}
	thread, err := s.GetThread(ctx, post.ThreadID)
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
	if !p.Capabilities().CanModerate {
		return nil, nil, ErrUnsupportedOperation
	}
	aboned, err := p.AbonePost(ctx, postID, SanitizeSingleLine(reason))
	if err != nil {
		return nil, nil, err
	}
	if s.isNativeGlobalBoard(board) {
		pCopy := *aboned
		tCopy := *thread
		s.emitEvent(Event{
			Type:     EventPostAboned,
			BoardID:  board.ID,
			ThreadID: thread.ID,
			PostID:   aboned.ID,
			Thread:   &tCopy,
			Post:     &pCopy,
		})
	}
	return aboned, thread, nil
}
