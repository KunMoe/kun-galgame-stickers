package service

import (
	"context"
	"log/slog"
	"strconv"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/pkg/communityclient"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/userclient"

	"github.com/google/uuid"
)

const (
	notificationPageSize = 30
	maxNotificationMark  = 100
)

// Notifications is the reader's inbox on this site, newest first. community
// writes the rows and scopes them to this site; all this does is name the
// packs and the people behind them.
func (s *Service) Notifications(ctx context.Context, v Viewer, cursor string) (*dto.NotificationPage, *errors.AppError) {
	if !s.community.Configured() {
		return &dto.NotificationPage{Items: []dto.NotificationItem{}, Enabled: false}, nil
	}
	// The cursor is a notification seq upstream, not an opaque token. A
	// malformed one is a client bug, and passing it through comes back as a
	// 422 that reads to the reader as "notifications are down".
	if cursor != "" {
		if n, convErr := strconv.ParseInt(cursor, 10, 64); convErr != nil || n <= 0 {
			return nil, errors.ErrInvalidParams("cursor must be a notification sequence")
		}
	}

	list, err := s.community.Notifications(ctx, v.UID, cursor, notificationPageSize, false)
	if err != nil {
		return nil, communityError(err)
	}

	anchorIDs := make([]string, 0, len(list.Notifications))
	actorIDs := make([]int, 0, len(list.Notifications))
	for _, row := range list.Notifications {
		if row.AnchorKind == communityclient.AnchorSiteResource {
			anchorIDs = append(anchorIDs, row.AnchorID)
		}
		if row.ActorID != 0 {
			actorIDs = append(actorIDs, row.ActorID)
		}
	}
	packs, lookupErr := s.packRefs(anchorIDs)
	if lookupErr != nil {
		slog.Error("notification lookup failed", "error", lookupErr)
		return nil, errors.ErrInternal("notification lookup failed")
	}

	return &dto.NotificationPage{
		Items:       notificationRows(list.Notifications, packs, s.users.Users(ctx, actorIDs)),
		NextCursor:  list.NextCursor,
		UnreadCount: list.UnreadCount,
		Enabled:     true,
	}, nil
}

// notificationRows keeps every row, in order. A notification whose pack is no
// longer published still occupies a slot the unread_count covers; dropping it
// would make the red dot lead to a shorter list.
func notificationRows(
	rows []communityclient.Notification,
	packs map[string]dto.CommentPackRef,
	users map[int]userclient.User,
) []dto.NotificationItem {
	out := make([]dto.NotificationItem, 0, len(rows))
	for _, row := range rows {
		item := dto.NotificationItem{
			ID:              row.ID,
			Kind:            row.Kind,
			ThreadID:        row.ThreadID,
			ActorCount:      row.ActorCount,
			ItemCount:       row.ItemCount,
			PostID:          row.PostID,
			PostNumber:      row.PostNumber,
			FirstPostNumber: row.FirstPostNumber,
			Read:            row.ReadAt != "",
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		}
		if row.AnchorKind == communityclient.AnchorSiteResource {
			if pack, ok := packs[row.AnchorID]; ok {
				item.Pack = &pack
			}
		}
		if row.ActorID != 0 {
			author, ok := users[row.ActorID]
			if !ok {
				author = userclient.Placeholder(row.ActorID)
			}
			actor := authorDTO(author)
			item.Actor = &actor
		}
		out = append(out, item)
	}
	return out
}

// UnreadNotificationCount is the header's red dot. It resolves no packs and no
// users: the header asks on every page, and a number needs neither.
func (s *Service) UnreadNotificationCount(ctx context.Context, v Viewer) (*dto.NotificationCount, *errors.AppError) {
	if !s.community.Configured() {
		return &dto.NotificationCount{}, nil
	}
	list, err := s.community.Notifications(ctx, v.UID, "", 1, true)
	if err != nil {
		return nil, communityError(err)
	}
	return &dto.NotificationCount{UnreadCount: list.UnreadCount, Enabled: true}, nil
}

// validateNotificationMark mirrors community's own rules, so a malformed
// request is refused here rather than reported as the service being down.
func validateNotificationMark(ids []int64, all bool) *errors.AppError {
	if all == (len(ids) > 0) {
		return errors.ErrInvalidParams("name the notifications to mark, or all of them")
	}
	if len(ids) > maxNotificationMark {
		return errors.ErrInvalidParams("too many notifications")
	}
	for _, id := range ids {
		if id <= 0 {
			return errors.ErrInvalidParams("notification ids must be positive")
		}
	}
	return nil
}

func (s *Service) MarkNotificationsRead(ctx context.Context, v Viewer, ids []int64, all bool) (*dto.NotificationCount, *errors.AppError) {
	if !s.community.Configured() {
		return nil, errors.ErrCommunityUnavailable()
	}
	if appErr := validateNotificationMark(ids, all); appErr != nil {
		return nil, appErr
	}
	n, err := s.community.MarkNotificationsRead(ctx, v.UID, ids, all)
	if err != nil {
		return nil, communityError(err)
	}
	return &dto.NotificationCount{UnreadCount: n, Enabled: true}, nil
}

// SetPackCommentNotification is the follow button for a wall that has no
// thread yet; once a thread exists the thread-level endpoint is the one that
// counts, because community ranks a thread row above an anchor row.
func (s *Service) SetPackCommentNotification(
	ctx context.Context,
	packID uuid.UUID,
	level int,
	v Viewer,
) (*dto.CommentViewerState, *errors.AppError) {
	pack, appErr := s.visiblePack(packID, v)
	if appErr != nil {
		return nil, appErr
	}
	if !s.community.Configured() {
		return nil, errors.ErrCommunityUnavailable()
	}
	if level != communityclient.NotifyNormal && level != communityclient.NotifyWatching {
		return nil, errors.ErrInvalidParams("a pack's comments can only be followed or unfollowed")
	}
	state, err := s.community.SetAnchorNotification(ctx, v.UID, communityclient.AnchorRef{
		AnchorKind: communityclient.AnchorSiteResource,
		AnchorID:   pack.ID.String(),
	}, level)
	if err != nil {
		return nil, communityError(err)
	}
	return &dto.CommentViewerState{NotificationLevel: state.NotificationLevel}, nil
}

// anchorViewerState is the reader's follow on a pack whose wall has no thread
// yet. community keeps the row sparse, so no row means normal. Like
// viewerState, a failure costs the page its button state, not its comments.
func (s *Service) anchorViewerState(ctx context.Context, packID string, v Viewer) *dto.CommentViewerState {
	if v.UID <= 0 {
		return nil
	}
	states, err := s.community.AnchorStates(ctx, v.UID, []communityclient.AnchorRef{{
		AnchorKind: communityclient.AnchorSiteResource,
		AnchorID:   packID,
	}})
	if err != nil {
		slog.Warn("comment anchor state lookup failed",
			"pack_id", packID, "user_id", v.UID, "error", err)
		return nil
	}
	level := communityclient.NotifyNormal
	for _, state := range states {
		if state.AnchorKind == communityclient.AnchorSiteResource && state.AnchorID == packID {
			level = state.NotificationLevel
			break
		}
	}
	return &dto.CommentViewerState{NotificationLevel: level}
}
