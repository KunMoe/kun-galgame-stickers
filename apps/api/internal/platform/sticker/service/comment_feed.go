package service

import (
	"context"
	"log/slog"
	"strings"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/internal/platform/sticker/model"
	"kun-galgame-sticker-api/pkg/communityclient"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/userclient"

	"github.com/google/uuid"
)

const (
	commentFeedSize = 20
	// community's own bounds on a search term. Below two characters the query
	// matches most of the corpus; above a hundred it is not a search.
	minCommentQuery = 2
	maxCommentQuery = 100
	// unreadScanLimit is community's page maximum. The unread list is capped
	// rather than paged: it exists to be cleared, not browsed, and a reader
	// with more than a hundred unread packs is served by the number alone.
	unreadScanLimit = 100
	// feedScanPages bounds the refill loops below. Every community lane this
	// file reads reaches wider than this site can render: each carries the
	// caller's own site plus every catalog-anchored thread on the network, and
	// the search lane cannot filter by anchor at all. So a page can arrive full
	// and leave nothing after the mapping back to a pack. Stopping at one page
	// would answer "nothing here" while still holding a cursor; walking without
	// a bound would let one quiet query scan the whole corpus.
	feedScanPages = 5
)

// packRefs turns comment anchors back into packs. Only published packs come
// back: a comment on a draft or a retired pack has no page to open, and listing
// it would advertise something the reader cannot reach.
func (s *Service) packRefs(anchorIDs []string) (map[string]dto.CommentPackRef, error) {
	if len(anchorIDs) == 0 {
		return nil, nil
	}
	// An anchor that is not a uuid cannot be one of this site's packs, and
	// leaving it in makes `id IN (…)` fail for the whole batch rather than for
	// the one row.
	ids := make([]string, 0, len(anchorIDs))
	for _, raw := range anchorIDs {
		if _, ok := parseUUID(raw); ok {
			ids = append(ids, raw)
		}
	}
	rows, err := s.packs.ByIDs(ids)
	if err != nil {
		// Every row is keyed on this lookup, so swallowing it would empty the
		// feed. An empty feed is a statement about the site; this is a database
		// that did not answer, and the reader is owed the difference.
		return nil, err
	}

	coverIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		if row.CoverStickerID != nil {
			coverIDs = append(coverIDs, *row.CoverStickerID)
		}
	}
	covers, err := s.stickers.ByIDs(coverIDs)
	if err != nil {
		// A row without its cover still reads; without its pack it does not.
		slog.Warn("comment feed cover lookup failed", "count", len(coverIDs), "error", err)
	}

	out := make(map[string]dto.CommentPackRef, len(rows))
	for _, row := range rows {
		if row.Status != model.PackPublished {
			continue
		}
		ref := dto.CommentPackRef{ID: row.ID.String(), Title: decodeML(row.Title)}
		if row.CoverStickerID != nil {
			if cover, ok := covers[*row.CoverStickerID]; ok {
				_, ref.CoverThumbURL = s.urls(cover.ImageHash)
			}
		}
		out[ref.ID] = ref
	}
	return out, nil
}

// feedItems renders a community post feed as rows this site can show. It
// gathers what the rows need -- the packs behind their anchors, the people who
// wrote them -- and hands the rendering to commentFeedRows.
func (s *Service) feedItems(ctx context.Context, posts []communityclient.FeedPost) ([]dto.CommentFeedItem, error) {
	anchorIDs := make([]string, 0, len(posts))
	authorIDs := make([]int, 0, len(posts))
	for _, row := range posts {
		if row.Thread.AnchorKind != communityclient.AnchorSiteResource {
			continue
		}
		anchorIDs = append(anchorIDs, row.Thread.AnchorID)
		authorIDs = append(authorIDs, row.Post.AuthorID)
	}
	packs, err := s.packRefs(anchorIDs)
	if err != nil {
		return nil, err
	}
	return commentFeedRows(posts, packs, s.users.Users(ctx, authorIDs)), nil
}

// commentFeedRows keeps what this site can show and drops the rest. Three kinds
// of row are dropped for ordinary reasons: a post that is held or tombstoned,
// a post on an anchor that is not a site resource -- community's scope reaches
// catalog-anchored threads that belong to the whole network -- and a post on a
// pack that is no longer published, which has no page to open.
func commentFeedRows(
	posts []communityclient.FeedPost,
	packs map[string]dto.CommentPackRef,
	authors map[int]userclient.User,
) []dto.CommentFeedItem {
	out := make([]dto.CommentFeedItem, 0, len(posts))
	for _, row := range posts {
		if row.Post.Status != communityclient.StatusVisible {
			continue
		}
		if row.Thread.AnchorKind != communityclient.AnchorSiteResource {
			continue
		}
		pack, ok := packs[row.Thread.AnchorID]
		if !ok {
			continue
		}
		author, ok := authors[row.Post.AuthorID]
		if !ok {
			author = userclient.Placeholder(row.Post.AuthorID)
		}
		out = append(out, dto.CommentFeedItem{
			ID:          row.Post.ID,
			PostNumber:  row.Post.PostNumber,
			ContentHTML: row.Post.ContentHTML,
			CreatedAt:   row.Post.CreatedAt,
			Author:      dto.Author{ID: author.ID, Name: author.Name, Avatar: author.Avatar},
			Pack:        pack,
		})
	}
	return out
}

// collectFeed reads community pages until it holds a page worth of rows this
// site can render, or the lane runs out. The cursor it returns is the last page
// actually consumed, so resuming from it neither repeats nor skips a row.
//
// Rows past commentFeedSize are kept rather than trimmed: community's cursor is
// page-granular, so a trimmed row is one no later request could ask for again.
func (s *Service) collectFeed(
	ctx context.Context,
	cursor string,
	page func(string) (*communityclient.PostFeed, error),
) ([]dto.CommentFeedItem, string, *errors.AppError) {
	items := make([]dto.CommentFeedItem, 0, commentFeedSize)
	for range feedScanPages {
		feed, err := page(cursor)
		if err != nil {
			return nil, "", communityError(err)
		}
		rows, err := s.feedItems(ctx, feed.Posts)
		if err != nil {
			// The reader gets a failure rather than an empty feed; the cause
			// only exists here, so it is logged before it is flattened.
			slog.Error("comment feed pack lookup failed", "error", err)
			return nil, "", errors.ErrInternal("comment feed lookup failed")
		}
		items = append(items, rows...)
		cursor = feed.NextCursor
		if cursor == "" || len(items) >= commentFeedSize {
			break
		}
	}
	return items, cursor, nil
}

// LatestComments is the site's comment wall read sideways: every pack's newest
// comments in one stream. replies_only stays off -- on a comment wall
// post_number 1 is the first comment, not an opening post to skip.
func (s *Service) LatestComments(ctx context.Context, cursor string) (*dto.CommentFeed, *errors.AppError) {
	if !s.community.Configured() {
		return &dto.CommentFeed{Items: []dto.CommentFeedItem{}, Enabled: false}, nil
	}
	items, next, appErr := s.collectFeed(ctx, cursor, func(c string) (*communityclient.PostFeed, error) {
		return s.community.LatestPosts(ctx, communityclient.FeedParams{
			Kind:       communityclient.KindComments,
			AnchorKind: communityclient.AnchorSiteResource,
			Cursor:     c,
			Limit:      commentFeedSize,
		})
	})
	if appErr != nil {
		return nil, appErr
	}
	return &dto.CommentFeed{Items: items, NextCursor: next, Enabled: true}, nil
}

// SearchComments matches the markdown a commenter typed, not the HTML it was
// cooked into. It is a separate lane from the pack search behind the command
// palette: that one answers from this site's own tables, and this one is a
// round trip to community on every query.
//
// This lane has no anchor filter upstream, so it is the one that most needs the
// refill: a query whose first page is all catalog comments still has this
// site's hits on the page after.
func (s *Service) SearchComments(
	ctx context.Context,
	query, cursor string,
) (*dto.CommentFeed, *errors.AppError) {
	if !s.community.Configured() {
		return &dto.CommentFeed{Items: []dto.CommentFeedItem{}, Enabled: false}, nil
	}
	query = strings.TrimSpace(query)
	if runes := []rune(query); len(runes) < minCommentQuery {
		// Not an error: a palette that fires on the first keystroke gets an
		// empty answer rather than a red box.
		return &dto.CommentFeed{Items: []dto.CommentFeedItem{}, Enabled: true}, nil
	} else if len(runes) > maxCommentQuery {
		query = string(runes[:maxCommentQuery])
	}

	items, next, appErr := s.collectFeed(ctx, cursor, func(c string) (*communityclient.PostFeed, error) {
		return s.community.SearchPosts(ctx, query, communityclient.KindComments, c, commentFeedSize)
	})
	if appErr != nil {
		return nil, appErr
	}
	return &dto.CommentFeed{Items: items, NextCursor: next, Enabled: true}, nil
}

// UnreadComments lists the packs whose comment wall has moved on since this
// reader last looked. Community counts every thread the user can reach from
// this tenant, which is wider than this site can render, so the total reported
// here is the one computed from the rows that survive the mapping back to a
// pack -- a red dot that leads to an empty list is worse than no red dot. For
// the same reason it keeps reading pages: a reader whose first hundred unread
// threads are all catalog discussions elsewhere on the network still has unread
// comments here.
func (s *Service) UnreadComments(ctx context.Context, v Viewer) (*dto.UnreadComments, *errors.AppError) {
	if !s.community.Configured() {
		return &dto.UnreadComments{Packs: []dto.UnreadPack{}, Enabled: false}, nil
	}

	out := &dto.UnreadComments{Packs: make([]dto.UnreadPack, 0, commentFeedSize), Enabled: true}
	cursor := ""
	for range feedScanPages {
		list, err := s.community.Unread(ctx, v.UID, cursor, unreadScanLimit)
		if err != nil {
			return nil, communityError(err)
		}
		packs, lookupErr := s.unreadPacks(list.Threads)
		if lookupErr != nil {
			slog.Error("unread pack lookup failed", "user", v.UID, "error", lookupErr)
			return nil, errors.ErrInternal("unread lookup failed")
		}
		out.Packs = append(out.Packs, packs...)
		cursor = list.NextCursor
		if cursor == "" || len(out.Packs) >= unreadScanLimit {
			break
		}
	}
	if len(out.Packs) > unreadScanLimit {
		out.Packs = out.Packs[:unreadScanLimit]
	}
	out.Total = len(out.Packs)
	return out, nil
}

// unreadPacks maps one community page of unread threads onto this site's packs.
// The anchor kind is tested again here and not only where the ids were
// gathered: a catalog thread whose anchor id happened to equal a published
// pack's uuid would otherwise be rendered as that pack.
func (s *Service) unreadPacks(threads []communityclient.UnreadThread) ([]dto.UnreadPack, error) {
	anchorIDs := make([]string, 0, len(threads))
	for _, row := range threads {
		if row.Thread.AnchorKind == communityclient.AnchorSiteResource {
			anchorIDs = append(anchorIDs, row.Thread.AnchorID)
		}
	}
	packs, err := s.packRefs(anchorIDs)
	if err != nil {
		return nil, err
	}

	out := make([]dto.UnreadPack, 0, len(threads))
	for _, row := range threads {
		if row.Thread.AnchorKind != communityclient.AnchorSiteResource {
			continue
		}
		pack, ok := packs[row.Thread.AnchorID]
		if !ok {
			continue
		}
		out = append(out, dto.UnreadPack{
			ThreadID:     row.Thread.ID,
			Pack:         pack,
			UnreadCount:  row.State.UnreadCount,
			PostsCount:   row.Thread.PostsCount,
			LastPostedAt: row.Thread.LastPostedAt,
		})
	}
	return out, nil
}
