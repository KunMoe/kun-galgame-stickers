package service

import (
	"testing"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/pkg/communityclient"
	"kun-galgame-sticker-api/pkg/userclient"
)

// The feed lanes are tenant-scoped upstream, not site-scoped: they reach
// catalog-anchored threads that belong to the whole network, and they keep
// answering for a pack long after it stops being published. A row this site
// cannot open is worse than a shorter feed, so each of these is dropped.
func TestCommentFeedRowsDropsWhatThisSiteCannotOpen(t *testing.T) {
	const packID = "11111111-1111-1111-1111-111111111111"
	packs := map[string]dto.CommentPackRef{packID: {ID: packID}}
	authors := map[int]userclient.User{7: {ID: 7, Name: "kun"}}

	post := func(id int64, status int) communityclient.Post {
		return communityclient.Post{ID: id, AuthorID: 7, Status: status}
	}
	resource := communityclient.PostContext{AnchorKind: communityclient.AnchorSiteResource, AnchorID: packID}

	cases := []struct {
		name string
		row  communityclient.FeedPost
		keep bool
	}{
		{"visible comment on a published pack", communityclient.FeedPost{Post: post(1, communityclient.StatusVisible), Thread: resource}, true},
		{"held for review upstream", communityclient.FeedPost{Post: post(2, communityclient.StatusHeld), Thread: resource}, false},
		{"tombstoned by its author", communityclient.FeedPost{Post: post(3, communityclient.StatusDeleted), Thread: resource}, false},
		{"another site's catalog conversation", communityclient.FeedPost{
			Post:   post(4, communityclient.StatusVisible),
			Thread: communityclient.PostContext{AnchorKind: communityclient.AnchorCatalogWork, AnchorID: "1024"},
		}, false},
		{"pack no longer published", communityclient.FeedPost{
			Post:   post(5, communityclient.StatusVisible),
			Thread: communityclient.PostContext{AnchorKind: communityclient.AnchorSiteResource, AnchorID: "22222222-2222-2222-2222-222222222222"},
		}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := commentFeedRows([]communityclient.FeedPost{tc.row}, packs, authors)
			if tc.keep && len(got) != 1 {
				t.Fatalf("want the row kept, got %d rows", len(got))
			}
			if !tc.keep && len(got) != 0 {
				t.Fatalf("want the row dropped, got %+v", got)
			}
		})
	}
}

// An author community knows about but the OP has since forgotten still wrote
// the comment. Dropping the row would make a conversation lose a turn; the
// placeholder keeps it readable.
func TestCommentFeedRowsKeepsAPostWhoseAuthorIsUnknown(t *testing.T) {
	const packID = "11111111-1111-1111-1111-111111111111"
	rows := commentFeedRows(
		[]communityclient.FeedPost{{
			Post:   communityclient.Post{ID: 1, AuthorID: 99, Status: communityclient.StatusVisible},
			Thread: communityclient.PostContext{AnchorKind: communityclient.AnchorSiteResource, AnchorID: packID},
		}},
		map[string]dto.CommentPackRef{packID: {ID: packID}},
		map[int]userclient.User{},
	)
	if len(rows) != 1 {
		t.Fatalf("want the row kept, got %d", len(rows))
	}
	if rows[0].Author.ID != 99 || rows[0].Author.Name == "" {
		t.Errorf("want a named placeholder for user 99, got %+v", rows[0].Author)
	}
}
