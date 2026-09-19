package dto

import (
	"encoding/json"
	"testing"
)

// The cursor is pinned to infra's collect.EncodeOffset byte for byte, so the
// expected strings here are what that function returns for the same offsets.
func TestFaceCursorMatchesInfraEncoding(t *testing.T) {
	if FaceCursor(0) != nil {
		t.Error("offset 0 must have no cursor")
	}
	for offset, want := range map[int]string{20: "cur_MjA", 100: "cur_MTAw", 7: "cur_Nw"} {
		got := FaceCursor(offset)
		if got == nil || *got != want {
			t.Errorf("FaceCursor(%d) = %v, want %s", offset, got, want)
			continue
		}
		if back, ok := ParseFaceCursor(*got); !ok || back != offset {
			t.Errorf("ParseFaceCursor(%s) = %d, %v; want %d", *got, back, ok, offset)
		}
	}
	for _, bad := range []string{"cur_", "MjA", "cur_!!", "cur_LTE", "cur_YWJj", "page_MjA"} {
		if _, ok := ParseFaceCursor(bad); ok {
			t.Errorf("ParseFaceCursor(%q) accepted a cursor this face could not have minted", bad)
		}
	}
}

func TestFaceListEnvelope(t *testing.T) {
	keys := func(t *testing.T, list any) map[string]json.RawMessage {
		t.Helper()
		raw, err := json.Marshal(list)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out map[string]json.RawMessage
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return out
	}

	t.Run("a middle page points onward and hides the total", func(t *testing.T) {
		got := keys(t, NewFaceList([]int{1, 2}, FacePage{Offset: 4, Limit: 2}, 10))
		if string(got["next_cursor"]) != `"cur_Ng"` {
			t.Errorf("next_cursor = %s, want the cursor for offset 6", got["next_cursor"])
		}
		if _, found := got["total"]; found {
			t.Error("total must be absent without include_total")
		}
	})

	t.Run("the last page omits next_cursor rather than sending null", func(t *testing.T) {
		got := keys(t, NewFaceList([]int{9, 10}, FacePage{Offset: 8, Limit: 2, IncludeTotal: true}, 10))
		if _, found := got["next_cursor"]; found {
			t.Errorf("next_cursor = %s on the last page", got["next_cursor"])
		}
		if string(got["total"]) != "10" {
			t.Errorf("total = %s, want 10 with include_total", got["total"])
		}
	})

	t.Run("an empty page never points onward", func(t *testing.T) {
		got := keys(t, NewFaceList([]int{}, FacePage{Offset: 4, Limit: 2}, 10))
		if _, found := got["next_cursor"]; found {
			t.Error("an empty page must not hand back a cursor")
		}
		if string(got["items"]) != "[]" || string(got["object"]) != `"list"` {
			t.Errorf("envelope = %v", got)
		}
	})
}
