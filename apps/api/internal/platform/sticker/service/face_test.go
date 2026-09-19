package service

import (
	"encoding/json"
	"testing"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/internal/platform/sticker/model"
	"kun-galgame-sticker-api/internal/platform/sticker/repository"
)

func TestFaceRatingSpeaksCatalogVocabulary(t *testing.T) {
	if got := faceRating(model.RatingSFW); got != "all_ages" {
		t.Errorf("sfw = %q, want all_ages", got)
	}
	if got := faceRating(model.RatingNSFW); got != "r18" {
		t.Errorf("nsfw = %q, want r18", got)
	}
}

// The published contract must not grow fields by accident: the face DTO is
// built from the site DTO, and the site DTO carries things -- a draft status,
// an owner's uid on an unpublished row -- that the face has no business
// emitting. Marshalling one and reading its keys is the cheapest guard.
func TestFacePackEmitsNoStatus(t *testing.T) {
	raw, err := json.Marshal(facePack(dto.Pack{
		ID: "01a0", Status: model.PackDraft, ContentRating: model.RatingNSFW,
		Title: dto.MultilingualText{"zh-cn": "测试"},
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, forbidden := range []string{"status", "is_official", "search_text", "owner_uid"} {
		if _, found := keys[forbidden]; found {
			t.Errorf("face pack leaks %q", forbidden)
		}
	}
	for _, required := range []string{"object", "id", "title", "content_rating", "official"} {
		if _, found := keys[required]; !found {
			t.Errorf("face pack is missing %q", required)
		}
	}
}

// Catalog and account ids leave the face as decimal strings, as catalog's own
// /v2 spells them: a number above 2^53 would round in a JavaScript client.
func TestFaceIDsAreStrings(t *testing.T) {
	raw, err := json.Marshal(dto.FacePack{
		Author: dto.FaceAuthor{ID: "2"},
		Work:   faceWorkPtr(&dto.CatalogWork{ID: 9007199254740993}),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out struct {
		Author struct{ ID json.RawMessage } `json:"author"`
		Work   struct{ ID json.RawMessage } `json:"work"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := string(out.Work.ID); got != `"9007199254740993"` {
		t.Errorf("work id = %s, want a decimal string", got)
	}
	if got := string(out.Author.ID); got != `"2"` {
		t.Errorf("author id = %s, want a decimal string", got)
	}

	character := characterRowDTO(repository.CharacterRow{CatalogCharacterID: 7935, CatalogWorkID: ptr(int64(12))})
	if character.ID != "7935" || character.Work.ID != "12" {
		t.Errorf("character row = %q / %q, want 7935 / 12", character.ID, character.Work.ID)
	}
}

func ptr[T any](v T) *T { return &v }
