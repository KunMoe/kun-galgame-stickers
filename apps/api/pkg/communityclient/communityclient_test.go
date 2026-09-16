package communityclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// The lanes do not share a payload shape, and getting it wrong fails silently:
// an unwrapped decode yields a zero-valued post that still passes a
// status==visible check, so a reply looks accepted and comes back blank. These
// bodies are copied from the live service.
func TestResponseShapesPerLane(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/community/comments":
			// The read lane wraps the thread in a page and may omit it entirely.
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{
				"thread":{"id":57182,"kind":1,"anchor_kind":2,"anchor_id":"pack-uuid",
				"posts_count":3,"highest_post_number":3},
				"posts":[{"id":11048,"post_number":1,"author_id":3,"content_html":"<p>hi</p>","status":0,
				"reaction_count":2,"viewer_reacted":true}],
				"next_cursor":"1"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/community/comments":
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{
				"thread":{"id":57182,"posts_count":4,"highest_post_number":4},
				"post":{"id":11049,"thread_id":57182,"post_number":4,"author_id":3,
				"content_raw":"hey","content_html":"<p>hey</p>","status":0}}}`))
		case r.Method == http.MethodPatch:
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"post":{"id":11049,"content_html":"<p>edited</p>","status":0,
				"reaction_count":5}}}`))
		case r.URL.Path == "/api/v1/community/posts/11049/reaction":
			// The toggle answers flat -- no {"post": …} -- and carries the count.
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{
				"added":true,"reaction_count":6,"author_id":3,"thread_id":57182,"anchor_kind":2,"anchor_id":"pack-uuid"}}`))
		case r.URL.Path == "/api/v1/community/threads/57182/read":
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{
				"thread_id":57182,"user_id":3,"last_read_post_number":4,
				"highest_post_number":4,"unread_count":0,"notification_level":3}}`))
		case r.URL.Path == "/api/v1/community/threads/states":
			// data is {"states": …}, not the array itself.
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"states":[
				{"thread_id":57182,"user_id":3,"last_read_post_number":2,"unread_count":2,"notification_level":3}]}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/community/users/3/notifications":
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"notifications":[
				{"id":41,"user_id":3,"kind":5,"thread_id":57182,"anchor_kind":2,"anchor_id":"pack-uuid",
				 "post_id":11049,"post_number":4,"first_post_number":4,"actor_id":4,"actor_count":2,"item_count":1,
				 "read_at":"2026-09-16T08:00:00Z","created_at":"2026-09-16T07:00:00Z","updated_at":"2026-09-16T07:30:00Z","seq":90},
				{"id":40,"user_id":3,"kind":3,"thread_id":57182,"anchor_kind":2,"anchor_id":"pack-uuid",
				 "post_id":null,"post_number":null,"first_post_number":null,"actor_count":1,"item_count":3,
				 "created_at":"2026-09-16T06:00:00Z","updated_at":"2026-09-16T06:00:00Z","seq":89}],
				"next_cursor":"89","unread_count":1}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/community/users/3/notifications/read":
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"marked":1,"unread_count":0}}`))
		case r.URL.Path == "/api/v1/community/anchors/notification":
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{
				"user_id":3,"anchor_kind":2,"anchor_id":"pack-uuid","notification_level":3}}`))
		case r.URL.Path == "/api/v1/community/anchors/states":
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"states":[
				{"user_id":3,"anchor_kind":2,"anchor_id":"pack-uuid","notification_level":3}]}}`))
		case r.URL.Path == "/api/v1/community/search/posts",
			r.URL.Path == "/api/v1/community/posts":
			// Both feed lanes nest the post under its thread context.
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"posts":[
				{"post":{"id":11050,"post_number":2,"author_id":4,"content_html":"<p>p2</p>","status":0},
				 "thread":{"thread_id":57182,"anchor_kind":2,"anchor_id":"pack-uuid"}}],
				"next_cursor":"opaque"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "id", ClientSecret: "secret"})
	if !c.Configured() {
		t.Fatal("client should be configured")
	}
	ctx := context.Background()

	page, err := c.Comments(ctx, AnchorSiteResource, "pack-uuid", "", 30, 3)
	if err != nil {
		t.Fatalf("comments: %v", err)
	}
	if page.Thread == nil || page.Thread.ID != 57182 || page.Thread.HighestPostNumber != 3 {
		t.Errorf("comments decoded the thread wrong: %+v", page.Thread)
	}
	if len(page.Posts) != 1 || page.Posts[0].ContentHTML != "<p>hi</p>" {
		t.Errorf("comments decoded posts wrong: %+v", page.Posts)
	}
	if page.Posts[0].ReactionCount != 2 || !page.Posts[0].ViewerReacted {
		t.Errorf("comments dropped the likes: %+v", page.Posts[0])
	}

	written, err := c.Comment(ctx, CommentParams{
		AnchorKind: AnchorSiteResource, AnchorID: "pack-uuid", AuthorID: 3, Body: "hey",
	})
	if err != nil {
		t.Fatalf("comment: %v", err)
	}
	if written.Post.ID != 11049 || written.Thread.ID != 57182 {
		t.Errorf("comment decoded wrong (data is {thread, post}): %+v", written)
	}

	edited, err := c.Edit(ctx, 11049, 3, "edited")
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if edited.ContentHTML != "<p>edited</p>" || edited.ReactionCount != 5 {
		t.Errorf("edit decoded wrong: %+v", edited)
	}

	toggled, err := c.ToggleReaction(ctx, 11049, 3, ReactionLike)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if !toggled.Added || toggled.ReactionCount != 6 || toggled.AnchorID != "pack-uuid" {
		t.Errorf("toggle decoded wrong (data is flat): %+v", toggled)
	}

	state, err := c.MarkRead(ctx, 57182, 3, 4)
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if state.LastReadPostNumber != 4 || state.NotificationLevel != NotifyWatching {
		t.Errorf("read receipt decoded wrong: %+v", state)
	}

	states, err := c.ThreadStates(ctx, 3, []int64{57182})
	if err != nil {
		t.Fatalf("thread states: %v", err)
	}
	if len(states) != 1 || states[0].UnreadCount != 2 {
		t.Errorf("states decoded wrong (data is {\"states\": …}): %+v", states)
	}

	inbox, err := c.Notifications(ctx, 3, "", 0, false)
	if err != nil {
		t.Fatalf("notifications: %v", err)
	}
	if len(inbox.Notifications) != 2 {
		t.Fatalf("notifications decoded wrong: %+v", inbox.Notifications)
	}
	first, second := inbox.Notifications[0], inbox.Notifications[1]
	if first.ID != 41 || first.Kind != NotificationLiked || first.ThreadID != 57182 ||
		first.PostID != 11049 || first.PostNumber != 4 || first.FirstPostNumber != 4 || first.ActorID != 4 {
		t.Errorf("full notification row decoded wrong: %+v", first)
	}
	if first.ReadAt == "" {
		t.Errorf("full notification row dropped read_at: %+v", first)
	}
	if second.ID != 40 || second.Kind != NotificationPosted ||
		second.PostID != 0 || second.PostNumber != 0 || second.FirstPostNumber != 0 || second.ActorID != 0 {
		t.Errorf("null/absent notification fields must be zero, got %+v", second)
	}
	if second.ReadAt != "" {
		t.Errorf("absent read_at must be empty, got %q", second.ReadAt)
	}
	if inbox.UnreadCount != 1 || inbox.NextCursor != "89" {
		t.Errorf("notification list cursor/count decoded wrong: cursor=%q unread=%d", inbox.NextCursor, inbox.UnreadCount)
	}

	unread, err := c.MarkNotificationsRead(ctx, 3, []int64{41}, false)
	if err != nil {
		t.Fatalf("mark notifications read: %v", err)
	}
	if unread != 0 {
		t.Errorf("mark notifications read decoded wrong: unread=%d", unread)
	}

	anchor, err := c.SetAnchorNotification(ctx, 3, AnchorRef{AnchorKind: AnchorSiteResource, AnchorID: "pack-uuid"}, NotifyWatching)
	if err != nil {
		t.Fatalf("set anchor notification: %v", err)
	}
	if anchor.UserID != 3 || anchor.AnchorKind != AnchorSiteResource ||
		anchor.AnchorID != "pack-uuid" || anchor.NotificationLevel != NotifyWatching {
		t.Errorf("anchor notification decoded wrong (data is flat): %+v", anchor)
	}

	anchorStates, err := c.AnchorStates(ctx, 3, []AnchorRef{{AnchorKind: AnchorSiteResource, AnchorID: "pack-uuid"}})
	if err != nil {
		t.Fatalf("anchor states: %v", err)
	}
	if len(anchorStates) != 1 || anchorStates[0].NotificationLevel != NotifyWatching {
		t.Errorf("anchor states decoded wrong (data is {\"states\": …}): %+v", anchorStates)
	}

	for name, feed := range map[string]func() (*PostFeed, error){
		"search": func() (*PostFeed, error) { return c.SearchPosts(ctx, "p2", KindComments, "", 20) },
		"latest": func() (*PostFeed, error) {
			return c.LatestPosts(ctx, FeedParams{Kind: KindComments, AnchorKind: AnchorSiteResource, Limit: 20})
		},
	} {
		got, err := feed()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(got.Posts) != 1 || got.Posts[0].Post.ID != 11050 || got.Posts[0].Thread.AnchorID != "pack-uuid" {
			t.Errorf("%s decoded wrong (data is {posts:[{post, thread}]}): %+v", name, got.Posts)
		}
		if got.NextCursor != "opaque" {
			t.Errorf("%s dropped the cursor: %q", name, got.NextCursor)
		}
	}
}

// A pack nobody has commented on has no thread, and the read lane says so
// rather than inventing one. Decoding that as a zero-valued thread would make
// the site report thread 0 and page against it forever.
func TestCommentsOnAnUntouchedAnchorHasNoThread(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"posts":[]}}`))
	}))
	defer srv.Close()

	page, err := New(Config{BaseURL: srv.URL, ClientID: "id", ClientSecret: "s"}).
		Comments(context.Background(), AnchorSiteResource, "pack-uuid", "", 30, 0)
	if err != nil {
		t.Fatalf("comments: %v", err)
	}
	if page.Thread != nil {
		t.Errorf("an anchor with no comments must decode to a nil thread, got %+v", page.Thread)
	}
	if len(page.Posts) != 0 {
		t.Errorf("want no posts, got %d", len(page.Posts))
	}
}

// kind 0 is topic and anchor_kind 0 is board, so a filter left off the wire is
// read upstream as a different filter rather than as no filter. Both feed
// lanes must write them out every time.
func TestFeedFiltersAreAlwaysOnTheWire(t *testing.T) {
	var seen url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Query()
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"posts":[]}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "id", ClientSecret: "s"})
	if _, err := c.LatestPosts(context.Background(), FeedParams{
		Kind: KindTopic, AnchorKind: AnchorBoard, Limit: 20,
	}); err != nil {
		t.Fatalf("latest: %v", err)
	}
	if seen.Get("kind") != "0" || seen.Get("anchor_kind") != "0" {
		t.Errorf("zero-valued filters dropped: %v", seen)
	}

	// The search lane has no anchor_kind upstream, so kind is the only filter
	// it gets. Dropping it would search every kind of thread on the network.
	if _, err := c.SearchPosts(context.Background(), "q", KindTopic, "", 20); err != nil {
		t.Fatalf("search: %v", err)
	}
	if seen.Get("kind") != "0" {
		t.Errorf("search dropped kind=0: %v", seen)
	}
}

// viewer_reacted is only filled for the viewer a read names, and "nobody" has
// to stay off the wire: viewer_id=0 means the same upstream today, but a
// reader who is not signed in is not user 0 and should not be sent as one.
func TestCommentsNameTheViewerOnlyWhenThereIsOne(t *testing.T) {
	var seen url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Query()
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"posts":[]}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "id", ClientSecret: "s"})
	if _, err := c.Comments(context.Background(), AnchorSiteResource, "pack-uuid", "30", 30, 42); err != nil {
		t.Fatalf("comments: %v", err)
	}
	if seen.Get("viewer_id") != "42" || seen.Get("after") != "30" {
		t.Errorf("signed-in read lost its viewer or cursor: %v", seen)
	}
	if _, err := c.Comments(context.Background(), AnchorSiteResource, "pack-uuid", "", 30, 0); err != nil {
		t.Fatalf("comments: %v", err)
	}
	if seen.Has("viewer_id") || seen.Has("after") {
		t.Errorf("anonymous first page sent a viewer or a cursor: %v", seen)
	}
}

// Query filters whose zero value means "omit" must stay off the wire, and a
// mark-read body that names both `ids` and `all` is refused upstream. The
// empty-anchor states call is the same short-circuit as ThreadStates: no
// round trip for an empty screen.
func TestNotificationRequestsCarryOnlyWhatWasAsked(t *testing.T) {
	type seen struct {
		path  string
		query string
		body  map[string]any
	}
	var calls []seen
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := seen{path: r.URL.Path, query: r.URL.RawQuery}
		raw, _ := io.ReadAll(r.Body)
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &rec.body)
		}
		calls = append(calls, rec)
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "id", ClientSecret: "s"})
	ctx := context.Background()

	if _, err := c.Notifications(ctx, 3, "", 0, false); err != nil {
		t.Fatalf("notifications empty: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("want 1 call, got %d", len(calls))
	}
	if calls[0].path != "/api/v1/community/users/3/notifications" {
		t.Errorf("notifications path: %s", calls[0].path)
	}
	if calls[0].query != "" {
		t.Errorf("unfiltered notifications sent a query: %q", calls[0].query)
	}

	if _, err := c.Notifications(ctx, 3, "89", 30, true); err != nil {
		t.Fatalf("notifications filtered: %v", err)
	}
	q, _ := url.ParseQuery(calls[1].query)
	if q.Get("cursor") != "89" || q.Get("limit") != "30" || q.Get("unread_only") != "true" {
		t.Errorf("filtered notifications lost a param: %v", q)
	}

	if _, err := c.MarkNotificationsRead(ctx, 3, []int64{40, 41}, false); err != nil {
		t.Fatalf("mark ids: %v", err)
	}
	idsBody := calls[2].body
	rawIDs, _ := json.Marshal(idsBody["ids"])
	if string(rawIDs) != "[40,41]" {
		t.Errorf("mark ids body: %v", idsBody)
	}
	if _, ok := idsBody["all"]; ok {
		t.Errorf("ids mark must not send all: %v", idsBody)
	}

	if _, err := c.MarkNotificationsRead(ctx, 3, nil, true); err != nil {
		t.Fatalf("mark all: %v", err)
	}
	allBody := calls[3].body
	if allBody["all"] != true {
		t.Errorf("all mark body: %v", allBody)
	}
	if _, ok := allBody["ids"]; ok {
		t.Errorf("all mark must not send ids: %v", allBody)
	}

	if _, err := c.SetAnchorNotification(ctx, 3, AnchorRef{AnchorKind: AnchorSiteResource, AnchorID: "pack-uuid"}, NotifyWatching); err != nil {
		t.Fatalf("set anchor: %v", err)
	}
	anchorBody := calls[4].body
	if len(anchorBody) != 4 {
		t.Errorf("set anchor extra keys: %v", anchorBody)
	}
	if anchorBody["user_id"] != float64(3) || anchorBody["anchor_kind"] != float64(AnchorSiteResource) ||
		anchorBody["anchor_id"] != "pack-uuid" || anchorBody["level"] != float64(NotifyWatching) {
		t.Errorf("set anchor body: %v", anchorBody)
	}

	before := len(calls)
	states, err := c.AnchorStates(ctx, 3, nil)
	if err != nil {
		t.Fatalf("empty anchor states: %v", err)
	}
	if states != nil {
		t.Errorf("empty anchor states must return nil, got %+v", states)
	}
	if len(calls) != before {
		t.Errorf("AnchorStates with no anchors made a request: %+v", calls[before:])
	}
}

// community answers errors with a non-2xx status today, but the envelope
// carries its own code and the client's own doc comment says that code is the
// verdict. If the two ever disagree, a 200 whose code is non-zero must not be
// decoded into a zero-valued page and handed back as an empty answer.
func TestNonZeroCodeIsAFailureEvenOnTwoHundred(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":5001,"message":"thread is locked","data":null}`))
	}))
	defer srv.Close()

	_, err := New(Config{BaseURL: srv.URL, ClientID: "id", ClientSecret: "s"}).
		Comments(context.Background(), AnchorSiteResource, "pack-uuid", "", 30, 0)
	if err == nil {
		t.Fatal("a non-zero code on a 200 must not read as success")
	}
	if !errors.Is(err, ErrUpstream) {
		t.Errorf("got %v, want an ErrUpstream", err)
	}
}

// author_id is this site's assertion of who is speaking, and content_rating is
// stamped on a thread that may not exist yet. A write that omits either is
// rejected upstream, which is a 422 the reader sees as "comment failed".
func TestCommentCarriesTheAnchorAndTheAuthor(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"thread":{"id":1},"post":{"id":2,"status":0}}}`))
	}))
	defer srv.Close()

	_, err := New(Config{BaseURL: srv.URL, ClientID: "id", ClientSecret: "s"}).
		Comment(context.Background(), CommentParams{
			AnchorKind: AnchorSiteResource, AnchorID: "pack-uuid",
			ContentRating: 0, AuthorID: 42, Body: "hi",
		})
	if err != nil {
		t.Fatalf("comment: %v", err)
	}
	for _, key := range []string{"anchor_kind", "anchor_id", "content_rating", "author_id", "body"} {
		if _, ok := body[key]; !ok {
			t.Errorf("%s missing from the comment body: %v", key, body)
		}
	}
	if _, ok := body["reply_to_post_id"]; ok {
		t.Error("a top-level comment must not claim to answer post 0")
	}
}

// A site whose client carries no tenant binding is refused on every lane, and
// that has to stay distinguishable from the service being down: one is a
// configuration mistake, the other is worth retrying.
func TestForbiddenAndRateLimitAreDistinct(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   error
	}{
		{http.StatusForbidden, ErrForbidden},
		{http.StatusTooManyRequests, ErrRateLimited},
		{http.StatusNotFound, ErrNotFound},
		{http.StatusConflict, ErrConflict},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(`{"code":1,"message":"nope"}`))
		}))
		_, err := New(Config{BaseURL: srv.URL, ClientID: "id", ClientSecret: "s"}).
			Comments(context.Background(), AnchorSiteResource, "x", "", 30, 0)
		if err != tc.want {
			t.Errorf("status %d gave %v, want %v", tc.status, err, tc.want)
		}
		srv.Close()
	}
}

func TestUnconfiguredClientRefusesRatherThanCallingNowhere(t *testing.T) {
	c := New(Config{})
	if c.Configured() {
		t.Fatal("an empty config must not report configured")
	}
	if _, err := c.Comments(context.Background(), AnchorSiteResource, "x", "", 30, 0); err != ErrNotConfigured {
		t.Errorf("got %v, want ErrNotConfigured", err)
	}
}
