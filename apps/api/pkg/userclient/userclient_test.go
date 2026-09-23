package userclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
)

func TestUsersCarriesCosmeticsThroughTheCache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/users/batch" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"users":[
			{"id":1,"uuid":"u1","name":"kun","avatar":"https://legacy/a.png","avatar_image_hash":"abcdef0123",
			 "roles":["user"],"site_roles":["editor"],
			 "cosmetics":{
				"avatar_frame":{"item_id":3,"name":"樱花","static_url":"https://img/decorations/f.png","animated_url":"https://img/decorations/f.webp"},
				"profile_background":{"item_id":9,"name":"星空","static_url":"https://img/decorations/b.jpg"},
				"nameplate":{"item_id":12,"name":"future","static_url":"https://img/decorations/n.png"}}},
			{"id":2,"uuid":"u2","name":"ren","avatar":""}
		],"not_found":[]}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "sticker", ClientSecret: "s", ImageCDNBase: "https://cdn"})
	first := c.Users(context.Background(), []int{1, 2})

	kun := first[1]
	if kun.Cosmetics == nil || kun.Cosmetics.AvatarFrame == nil || kun.Cosmetics.ProfileBackground == nil {
		t.Fatalf("cosmetics = %+v", kun.Cosmetics)
	}
	if got := *kun.Cosmetics.AvatarFrame; got != (Decoration{
		ItemID: 3, Name: "樱花",
		StaticURL: "https://img/decorations/f.png", AnimatedURL: "https://img/decorations/f.webp",
	}) {
		t.Fatalf("avatar_frame = %+v", got)
	}
	if got := kun.Cosmetics.ProfileBackground.AnimatedURL; got != "" {
		t.Fatalf("absent animated_url decoded as %q", got)
	}
	if first[2].Cosmetics != nil {
		t.Fatalf("a user wearing nothing got %+v", first[2].Cosmetics)
	}

	// A cache hit used to return the OP's raw avatar and roles without the
	// site roles: the loop resolved copies of the fetched users and cached the
	// originals.
	second := c.Users(context.Background(), []int{1, 2})
	if n := calls.Load(); n != 1 {
		t.Fatalf("second read made %d upstream calls, want it served from cache", n)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("cache hit differs from the fresh read:\nfresh  %+v\ncached %+v", first, second)
	}
	if want := "https://cdn/ab/cd/abcdef0123.webp"; second[1].Avatar != want {
		t.Fatalf("cached avatar = %q, want %q", second[1].Avatar, want)
	}
}
