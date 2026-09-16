package service

import (
	"testing"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/pkg/communityclient"
	"kun-galgame-sticker-api/pkg/userclient"
)

func TestNotificationRowsKeepEveryRow(t *testing.T) {
	const packID = "11111111-1111-1111-1111-111111111111"
	packs := map[string]dto.CommentPackRef{packID: {ID: packID}}
	users := map[int]userclient.User{7: {ID: 7, Name: "kun"}}

	type rowCase struct {
		name  string
		in    communityclient.Notification
		check func(*testing.T, dto.NotificationItem)
	}
	cases := []rowCase{
		{
			name: "published pack",
			in: communityclient.Notification{
				ID: 1, AnchorKind: communityclient.AnchorSiteResource, AnchorID: packID, ActorID: 7,
			},
			check: func(t *testing.T, got dto.NotificationItem) {
				if got.Pack == nil || got.Pack.ID != packID {
					t.Errorf("want Pack set to the published pack, got %+v", got.Pack)
				}
			},
		},
		{
			name: "pack not in the map",
			in: communityclient.Notification{
				ID: 2, AnchorKind: communityclient.AnchorSiteResource, AnchorID: "22222222-2222-2222-2222-222222222222",
			},
			check: func(t *testing.T, got dto.NotificationItem) {
				if got.Pack != nil {
					t.Errorf("want Pack nil for an unknown pack, got %+v", got.Pack)
				}
			},
		},
		{
			name: "catalog work whose id equals a published pack uuid",
			in: communityclient.Notification{
				ID: 3, AnchorKind: communityclient.AnchorCatalogWork, AnchorID: packID, ActorID: 7,
			},
			check: func(t *testing.T, got dto.NotificationItem) {
				if got.Pack != nil {
					t.Errorf("want Pack nil for a catalog-work row, got %+v", got.Pack)
				}
			},
		},
		{
			name: "actor erased",
			in: communityclient.Notification{
				ID: 4, AnchorKind: communityclient.AnchorSiteResource, AnchorID: packID, ActorID: 0,
			},
			check: func(t *testing.T, got dto.NotificationItem) {
				if got.Actor != nil {
					t.Errorf("want Actor nil when ActorID is 0, got %+v", got.Actor)
				}
			},
		},
		{
			name: "actor missing from the users map",
			in: communityclient.Notification{
				ID: 5, AnchorKind: communityclient.AnchorSiteResource, AnchorID: packID, ActorID: 99,
			},
			check: func(t *testing.T, got dto.NotificationItem) {
				if got.Actor == nil || got.Actor.ID != 99 || got.Actor.Name == "" {
					t.Errorf("want a named placeholder for user 99, got %+v", got.Actor)
				}
			},
		},
		{
			name: "read_at set",
			in:   communityclient.Notification{ID: 6, ReadAt: "2026-09-16T08:00:00Z"},
			check: func(t *testing.T, got dto.NotificationItem) {
				if !got.Read {
					t.Errorf("want Read true when ReadAt is set")
				}
			},
		},
		{
			name: "read_at empty",
			in:   communityclient.Notification{ID: 7, ReadAt: ""},
			check: func(t *testing.T, got dto.NotificationItem) {
				if got.Read {
					t.Errorf("want Read false when ReadAt is empty")
				}
			},
		},
	}

	input := make([]communityclient.Notification, len(cases))
	for i, tc := range cases {
		input[i] = tc.in
	}
	got := notificationRows(input, packs, users)
	if len(got) != len(input) {
		t.Fatalf("want every row kept, got %d from %d", len(got), len(input))
	}
	for i, tc := range cases {
		if got[i].ID != tc.in.ID {
			t.Fatalf("order not preserved at %d: got id %d, want %d", i, got[i].ID, tc.in.ID)
		}
		t.Run(tc.name, func(t *testing.T) { tc.check(t, got[i]) })
	}
}

func TestValidateNotificationMark(t *testing.T) {
	hundred := make([]int64, 100)
	for i := range hundred {
		hundred[i] = int64(i + 1)
	}
	tooMany := append(append([]int64{}, hundred...), 101)

	cases := []struct {
		name string
		ids  []int64
		all  bool
		ok   bool
	}{
		{"nil and not all", nil, false, false},
		{"empty and not all", []int64{}, false, false},
		{"ids and all", []int64{1}, true, false},
		{"one id", []int64{1}, false, true},
		{"all", nil, true, true},
		{"101 ids", tooMany, false, false},
		{"100 ids", hundred, false, true},
		{"zero id", []int64{0}, false, false},
		{"negative id", []int64{-1}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNotificationMark(tc.ids, tc.all)
			if tc.ok && err != nil {
				t.Errorf("want ok, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Errorf("want an error")
			}
		})
	}
}
