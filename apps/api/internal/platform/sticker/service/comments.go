package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"strconv"
	"strings"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/pkg/communityclient"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/perm"
	"kun-galgame-sticker-api/pkg/userclient"

	"github.com/google/uuid"
)

const (
	// MaxCommentRunes is this site's own cap. Community has its own limits for
	// newcomers; this one keeps a comment a comment.
	MaxCommentRunes = 2000
	commentPageSize = 30
)

// PackComments returns a pack's comment thread. The thread is addressed by the
// pack, not by an id this site stores, and it does not exist until somebody
// comments: reading a pack page must not mint one.
func (s *Service) PackComments(
	ctx context.Context,
	packID uuid.UUID,
	after string,
	v Viewer,
) (*dto.CommentPage, *errors.AppError) {
	pack, appErr := s.visiblePack(packID, v)
	if appErr != nil {
		return nil, appErr
	}
	if !s.community.Configured() {
		return &dto.CommentPage{Comments: []dto.Comment{}, Enabled: false}, nil
	}
	// The cursor is a post_number upstream, not an opaque token. A malformed
	// one is a client bug, and passing it through comes back as a 422 that
	// reads to the reader as "comments are down".
	if after != "" {
		if n, convErr := strconv.Atoi(after); convErr != nil || n < 0 {
			return nil, errors.ErrInvalidParams("after must be a post number")
		}
	}

	page, err := s.community.Comments(
		ctx, communityclient.AnchorSiteResource, pack.ID.String(), after, commentPageSize,
	)
	if err != nil {
		return nil, communityError(err)
	}
	// No thread yet is an empty comment section, not a failure -- the reader
	// still gets a composer, and using it is what creates the thread.
	if page.Thread == nil {
		return &dto.CommentPage{Comments: []dto.Comment{}, Enabled: true}, nil
	}

	out := s.commentPage(ctx, *page.Thread, page.Posts, page.NextCursor, v)
	out.Viewer = s.viewerState(ctx, page.Thread.ID, v)
	return out, nil
}

// viewerState is the reader's own row on a thread: how far they read and
// whether they are subscribed. Only a signed-in reader costs the round trip,
// and a failure costs the page a badge rather than its comments.
func (s *Service) viewerState(ctx context.Context, threadID int64, v Viewer) *dto.CommentViewerState {
	if v.UID <= 0 || threadID <= 0 {
		return nil
	}
	states, err := s.community.ThreadStates(ctx, v.UID, []int64{threadID})
	if err != nil {
		slog.Warn("comment thread state lookup failed",
			"thread_id", threadID, "user_id", v.UID, "error", err)
		return nil
	}
	for _, state := range states {
		if state.ThreadID == threadID {
			return &dto.CommentViewerState{
				LastReadPostNumber: state.LastReadPostNumber,
				UnreadCount:        state.UnreadCount,
				NotificationLevel:  state.NotificationLevel,
			}
		}
	}
	// The row is sparse: a reader who has never touched this thread has no
	// state at all, which is not the same as having read none of it.
	return nil
}

func (s *Service) commentPage(
	ctx context.Context,
	thread communityclient.Thread,
	posts []communityclient.Post,
	cursor string,
	v Viewer,
) *dto.CommentPage {
	authorIDs := make([]int, 0, len(posts))
	postIDs := make([]int64, 0, len(posts))
	for _, post := range posts {
		authorIDs = append(authorIDs, post.AuthorID)
		postIDs = append(postIDs, post.ID)
	}
	authors := s.users.Users(ctx, authorIDs)
	// community's post projection carries no reaction fields, so the counts
	// come from this site's mirror -- two queries for the whole page.
	likeCounts, _ := s.likes.Counts(postIDs)
	liked, _ := s.likes.LikedSet(v.UID, postIDs)

	// "in reply to X" needs the name of a post that may be on another page, so
	// the ones on this page are indexed first and anything else stays unnamed.
	nameByPost := make(map[int64]string, len(posts))
	for _, post := range posts {
		if author, ok := authors[post.AuthorID]; ok {
			nameByPost[post.ID] = author.Name
		}
	}

	out := &dto.CommentPage{
		ThreadID:          thread.ID,
		Total:             thread.PostsCount,
		HighestPostNumber: thread.HighestPostNumber,
		NextCursor:        cursor,
		Comments:          make([]dto.Comment, 0, len(posts)),
		Enabled:           true,
	}
	for _, post := range posts {
		// A held or tombstoned post is not this site's to display: it is either
		// awaiting review upstream or deliberately gone.
		if post.Status != communityclient.StatusVisible {
			continue
		}
		author, ok := authors[post.AuthorID]
		if !ok {
			author = userclient.Placeholder(post.AuthorID)
		}
		out.Comments = append(out.Comments, dto.Comment{
			ID:          post.ID,
			PostNumber:  post.PostNumber,
			ContentHTML: post.ContentHTML,
			ContentRaw:  post.ContentRaw,
			CreatedAt:   post.CreatedAt,
			EditedAt:    post.EditedAt,
			Author:      dto.Author{ID: author.ID, Name: author.Name, Avatar: author.Avatar},
			CanEdit:     v.UID > 0 && post.AuthorID == v.UID,
			CanDelete:   v.UID > 0 && (post.AuthorID == v.UID || perm.Can(v.Roles, perm.PackDeleteAny)),
			LikeCount:   likeCounts[post.ID],
			IsLiked:     liked[post.ID],
			ReplyTo:     post.ReplyToPostID,
			RootID:      post.RootPostID,
			ReplyToName: nameByPost[post.ReplyToPostID],
		})
	}
	return out
}

func (s *Service) AddComment(
	ctx context.Context,
	packID uuid.UUID,
	v Viewer,
	body string,
	replyTo int64,
) (*dto.Comment, *errors.AppError) {
	pack, appErr := s.visiblePack(packID, v)
	if appErr != nil {
		return nil, appErr
	}
	if !s.community.Configured() {
		return nil, errors.ErrCommunityUnavailable()
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, errors.ErrInvalidParams("a comment cannot be empty")
	}
	if len([]rune(body)) > MaxCommentRunes {
		return nil, errors.ErrInvalidParams("comment is too long")
	}

	// One call, addressed by the anchor: community creates the thread with this
	// comment when it is the first. author_id is the session's, never the
	// request body's -- community trusts this site's assertion of who is
	// speaking.
	result, err := s.community.Comment(ctx, communityclient.CommentParams{
		AnchorKind:    communityclient.AnchorSiteResource,
		AnchorID:      pack.ID.String(),
		ContentRating: int(pack.ContentRating),
		AuthorID:      v.UID,
		Body:          body,
		ReplyToPostID: replyTo,
	})
	if err != nil {
		return nil, communityError(err)
	}
	page := s.commentPage(ctx, result.Thread, []communityclient.Post{result.Post}, "", v)
	if len(page.Comments) == 0 {
		// A newcomer's first posts are held for review upstream; the author is
		// told rather than shown a comment that is not there.
		return nil, errors.ErrCommentHeld()
	}
	return &page.Comments[0], nil
}

func (s *Service) EditComment(
	ctx context.Context,
	postID int64,
	v Viewer,
	body string,
) (*dto.Comment, *errors.AppError) {
	if !s.community.Configured() {
		return nil, errors.ErrCommunityUnavailable()
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, errors.ErrInvalidParams("a comment cannot be empty")
	}
	if len([]rune(body)) > MaxCommentRunes {
		return nil, errors.ErrInvalidParams("comment is too long")
	}
	post, err := s.community.Edit(ctx, postID, v.UID, body)
	if err != nil {
		return nil, communityError(err)
	}
	page := s.commentPage(ctx, communityclient.Thread{}, []communityclient.Post{*post}, "", v)
	if len(page.Comments) == 0 {
		return nil, errors.ErrCommentHeld()
	}
	return &page.Comments[0], nil
}

func (s *Service) DeleteComment(ctx context.Context, postID int64, v Viewer) *errors.AppError {
	if !s.community.Configured() {
		return errors.ErrCommunityUnavailable()
	}
	if err := s.community.Delete(ctx, postID, v.UID); err != nil {
		return communityError(err)
	}
	return nil
}

// communityError keeps the upstream's distinctions: a reader must be able to
// tell "not yours" from "the service is down", because only one of them is
// worth retrying.
func communityError(err error) *errors.AppError {
	switch {
	case communityclient.Missing(err):
		return errors.ErrCommentNotFound()
	case stderrors.Is(err, communityclient.ErrForbidden):
		return errors.ErrNotOwner()
	case stderrors.Is(err, communityclient.ErrRateLimited):
		return errors.ErrCommentRateLimited()
	case stderrors.Is(err, communityclient.ErrConflict):
		return errors.ErrInvalidParams("this comment can no longer be changed")
	default:
		return errors.ErrCommunityUnavailable()
	}
}

// ToggleCommentLike flips a reaction. community decides the new state -- its
// toggle is authoritative and feeds the trust engine -- and this site mirrors
// the outcome so it has something to count. A mirror write that fails leaves
// the count stale rather than the reaction lost, so it is logged, not raised.
func (s *Service) ToggleCommentLike(ctx context.Context, postID int64, v Viewer) (*dto.CommentLikeResult, *errors.AppError) {
	if !s.community.Configured() {
		return nil, errors.ErrCommunityUnavailable()
	}
	result, err := s.community.ToggleReaction(ctx, postID, v.UID, communityclient.ReactionLike)
	if err != nil {
		return nil, communityError(err)
	}

	if result.Added {
		if mirrorErr := s.likes.Ensure(postID, v.UID); mirrorErr != nil {
			slog.Warn("comment like mirror insert failed", "post_id", postID, "user_id", v.UID, "error", mirrorErr)
		}
	} else if mirrorErr := s.likes.Remove(postID, v.UID); mirrorErr != nil {
		slog.Warn("comment like mirror delete failed", "post_id", postID, "user_id", v.UID, "error", mirrorErr)
	}

	counts, err := s.likes.Counts([]int64{postID})
	if err != nil {
		return nil, errors.ErrInternal("failed to count likes")
	}
	return &dto.CommentLikeResult{Liked: result.Added, LikeCount: counts[postID]}, nil
}

// FlagComment hands a report to community's review queue. Nothing about it is
// stored here: the queue, the decision and the audit trail are all upstream.
func (s *Service) FlagComment(ctx context.Context, postID int64, v Viewer, reason int, note string) *errors.AppError {
	if !s.community.Configured() {
		return errors.ErrCommunityUnavailable()
	}
	if !validFlagReason(reason) {
		return errors.ErrInvalidParams("unknown report reason")
	}
	note = strings.TrimSpace(note)
	if len([]rune(note)) > MaxTextRunes {
		note = string([]rune(note)[:MaxTextRunes])
	}
	if err := s.community.Flag(ctx, postID, v.UID, reason, note); err != nil {
		return communityError(err)
	}
	return nil
}

// The reasons community accepts. Anything else is a client bug, and passing it
// through would put an unclassifiable report in a moderator's queue.
func validFlagReason(reason int) bool {
	switch reason {
	case communityclient.FlagSpam, communityclient.FlagAbuse, communityclient.FlagOffTopic,
		communityclient.FlagOther, communityclient.FlagNSFWMislabel:
		return true
	}
	return false
}

// MarkCommentsRead is the reader's receipt. It is a write on purpose: a read
// face that marks things read cannot be cached, retried or prefetched safely,
// so nothing about rendering the comment section moves this number.
func (s *Service) MarkCommentsRead(
	ctx context.Context,
	threadID int64,
	postNumber int,
	v Viewer,
) (*dto.CommentViewerState, *errors.AppError) {
	if !s.community.Configured() {
		return nil, errors.ErrCommunityUnavailable()
	}
	if postNumber < 0 {
		return nil, errors.ErrInvalidParams("post_number cannot be negative")
	}
	// A forged number cannot do damage: community clamps it to the thread's
	// highest post and never lets it go backwards, and the row it writes is
	// the caller's own.
	state, err := s.community.MarkRead(ctx, threadID, v.UID, postNumber)
	if err != nil {
		return nil, communityError(err)
	}
	return &dto.CommentViewerState{
		LastReadPostNumber: state.LastReadPostNumber,
		UnreadCount:        state.UnreadCount,
		NotificationLevel:  state.NotificationLevel,
	}, nil
}

// SetCommentNotification subscribes to or mutes a pack's comment wall.
// Commenting already subscribes the author upstream, so this exists for the
// two deliberate choices: muting a thread you are in, and following one you
// have not spoken in.
func (s *Service) SetCommentNotification(
	ctx context.Context,
	threadID int64,
	level int,
	v Viewer,
) (*dto.CommentViewerState, *errors.AppError) {
	if !s.community.Configured() {
		return nil, errors.ErrCommunityUnavailable()
	}
	switch level {
	case communityclient.NotifyMuted, communityclient.NotifyNormal,
		communityclient.NotifyTracking, communityclient.NotifyWatching:
	default:
		return nil, errors.ErrInvalidParams("unknown notification level")
	}
	state, err := s.community.SetNotification(ctx, threadID, v.UID, level)
	if err != nil {
		return nil, communityError(err)
	}
	return &dto.CommentViewerState{
		LastReadPostNumber: state.LastReadPostNumber,
		UnreadCount:        state.UnreadCount,
		NotificationLevel:  state.NotificationLevel,
	}, nil
}
