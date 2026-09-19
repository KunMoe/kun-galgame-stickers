package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"kun-galgame-sticker-api/pkg/problem"

	"github.com/gofiber/fiber/v3"
)

// The face's parameter parsing gets its own test because the first version of
// it was silently wrong in a way no build or vet could see: the helpers
// returned the result of problem.Write, which is nil when the write succeeds,
// so `if err != nil` never fired. Every refused parameter wrote a problem
// document, then ran the query anyway and replaced that document with a
// success body -- a 400 status line over a 200 body. These cases pin the shape
// of a refusal, not just its status.

type probed struct {
	status  int
	headers map[string]string
	doc     problem.Problem
	body    map[string]any
}

// probe mounts the helpers on their own app so a refusal can be observed
// without a database behind it. A helper that refuses must leave a problem
// document and nothing else; one that accepts must reach the handler body.
func probe(t *testing.T, url string) probed {
	t.Helper()
	app := fiber.New()
	app.Get("/packs", func(c fiber.Ctx) error {
		work, bad := faceCatalogFilter(c, "work")
		if bad != nil {
			return problem.Write(c, bad)
		}
		page, bad := facePage(c)
		if bad != nil {
			return problem.Write(c, bad)
		}
		if _, bad = faceSort(c); bad != nil {
			return problem.Write(c, bad)
		}
		if _, bad = faceRatingFilter(c); bad != nil {
			return problem.Write(c, bad)
		}
		return c.JSON(fiber.Map{
			"reached": true, "limit": page.Limit, "offset": page.Offset,
			"include_total": page.IncludeTotal, "work": work,
		})
	})
	app.Get("/characters/:character_id", func(c fiber.Ctx) error {
		id, bad := faceCatalogID(c, "character_id")
		if bad != nil {
			return problem.Write(c, bad)
		}
		return c.JSON(fiber.Map{"reached": true, "id": id})
	})
	app.Get("/packs/:pack_id", func(c fiber.Ctx) error {
		if _, bad := faceUUID(c, "pack_id"); bad != nil {
			return problem.Write(c, bad)
		}
		return c.JSON(fiber.Map{"reached": true})
	})

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, url, nil))
	if err != nil {
		t.Fatalf("request %s: %v", url, err)
	}
	defer res.Body.Close()

	var raw json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
	out := probed{status: res.StatusCode, headers: map[string]string{}}
	for _, name := range []string{fiber.HeaderContentType, fiber.HeaderCacheControl, problem.HeaderRequestID} {
		out.headers[name] = res.Header.Get(name)
	}
	if err := json.Unmarshal(raw, &out.doc); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
	if err := json.Unmarshal(raw, &out.body); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
	if out.body["reached"] == true && res.StatusCode != fiber.StatusOK {
		t.Fatalf("%s: status %d but the handler body ran anyway", url, res.StatusCode)
	}
	return out
}

func TestFaceRefusalsAreProblemDocuments(t *testing.T) {
	cases := []struct {
		name  string
		url   string
		code  string
		param string
	}{
		{"limit above the cap", "/packs?limit=101", problem.CodeLimitTooLarge, "limit"},
		{"limit not a number", "/packs?limit=many", problem.CodeInvalidParameter, "limit"},
		{"limit zero", "/packs?limit=0", problem.CodeInvalidParameter, "limit"},
		{"cursor not ours", "/packs?cursor=abc", problem.CodeInvalidCursor, "cursor"},
		{"cursor not base64", "/packs?cursor=cur_!!", problem.CodeInvalidCursor, "cursor"},
		{"cursor not an offset", "/packs?cursor=cur_YWJj", problem.CodeInvalidCursor, "cursor"},
		{"include_total is a closed vocabulary", "/packs?include_total=1", problem.CodeInvalidParameter, "include_total"},
		{"unknown sort", "/packs?sort=weird", problem.CodeInvalidParameter, "sort"},
		{"nsfw is a closed vocabulary", "/packs?nsfw=maybe", problem.CodeInvalidParameter, "nsfw"},
		{"work filter must be an id", "/packs?work=abc", problem.CodeInvalidParameter, "work"},
		{"catalog id must be positive", "/characters/0", problem.CodeInvalidParameter, "character_id"},
		{"catalog id must be a number", "/characters/abc", problem.CodeInvalidParameter, "character_id"},
		{"pack id must be a uuid", "/packs/not-a-uuid", problem.CodeInvalidParameter, "pack_id"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := probe(t, tc.url)
			doc := got.doc
			if got.status != fiber.StatusBadRequest || doc.Status != fiber.StatusBadRequest {
				t.Errorf("status = %d, body status = %d, want 400 for both", got.status, doc.Status)
			}
			if got.headers[fiber.HeaderContentType] != problem.ContentType {
				t.Errorf("content-type = %q, want %q", got.headers[fiber.HeaderContentType], problem.ContentType)
			}
			if doc.Code != tc.code {
				t.Errorf("code = %q, want %q", doc.Code, tc.code)
			}
			if !strings.HasPrefix(doc.Type, "https://developer.nextmoe.dev/problems/platform/") {
				t.Errorf("type = %q, want the platform domain", doc.Type)
			}
			if doc.Title == "" || doc.Detail == "" || doc.Instance != tc.url {
				t.Errorf("incomplete problem document: %+v", doc)
			}
			if len(doc.Errors) != 1 || doc.Errors[0].Parameter != tc.param ||
				doc.Errors[0].Reason == "" || doc.Errors[0].Detail == "" {
				t.Errorf("errors = %+v, want one entry naming %q", doc.Errors, tc.param)
			}
			if !strings.HasPrefix(doc.RequestID, "req_") || len(doc.RequestID) != 30 {
				t.Errorf("request_id = %q, want req_ plus a 26-character ULID", doc.RequestID)
			}
			if got.headers[problem.HeaderRequestID] != doc.RequestID {
				t.Errorf("X-Request-ID = %q, body request_id = %q; they must match",
					got.headers[problem.HeaderRequestID], doc.RequestID)
			}
			if got.headers[fiber.HeaderCacheControl] != "no-store" {
				t.Errorf("cache-control = %q, a problem must not be shared-cached", got.headers[fiber.HeaderCacheControl])
			}
		})
	}
}

// errors is [] rather than absent or null when the failure is not about one
// parameter, and detail is present even then.
func TestFaceProblemMembersAlwaysPresent(t *testing.T) {
	raw, err := json.Marshal(problem.OfStatus(fiber.StatusNotFound, "pack not found"))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(keys["errors"]) != "[]" {
		t.Errorf("errors = %s, want []", keys["errors"])
	}
	for _, required := range []string{"type", "title", "status", "detail", "instance", "code", "request_id"} {
		if _, found := keys[required]; !found {
			t.Errorf("problem is missing %q", required)
		}
	}
}

func TestFaceAcceptsWhatItShould(t *testing.T) {
	for url, want := range map[string]map[string]any{
		"/packs":                                      {"limit": 20.0, "offset": 0.0, "include_total": false},
		"/packs?limit=100&cursor=cur_MjA":             {"limit": 100.0, "offset": 20.0},
		"/packs?include_total=true&work=12":           {"include_total": true, "work": 12.0},
		"/packs?sort=hot&nsfw=true":                   {},
		"/packs?sort=new&nsfw=false":                  {},
		"/characters/7935":                            {"id": 7935.0},
		"/packs/0199a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b": {},
	} {
		got := probe(t, url)
		if got.status != fiber.StatusOK {
			t.Errorf("%s: status = %d (%s), want 200", url, got.status, got.doc.Code)
			continue
		}
		for key, value := range want {
			if got.body[key] != value {
				t.Errorf("%s: %s = %v, want %v", url, key, got.body[key], value)
			}
		}
	}
}
