package dto

import (
	"encoding/json"
	"testing"

	"kun-galgame-sticker-api/pkg/userclient"
)

func TestMeFlattensTheUserAndOmitsBareCosmetics(t *testing.T) {
	user := User{Sub: "s", ID: 7, Name: "kun", Picture: "p", Roles: []string{}}

	worn, err := json.Marshal(Me{User: user, Cosmetics: &userclient.Cosmetics{
		AvatarFrame: &userclient.Decoration{ItemID: 3, Name: "樱花", StaticURL: "https://img/f.png"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"sub":"s","id":7,"name":"kun","picture":"p","roles":[],` +
		`"cosmetics":{"avatar_frame":{"item_id":3,"name":"樱花","static_url":"https://img/f.png"}}}`
	if string(worn) != want {
		t.Fatalf("got  %s\nwant %s", worn, want)
	}

	bare, err := json.Marshal(Me{User: user})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"sub":"s","id":7,"name":"kun","picture":"p","roles":[]}`; string(bare) != want {
		t.Fatalf("got  %s\nwant %s", bare, want)
	}
}
