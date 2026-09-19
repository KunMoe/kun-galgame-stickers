package app

import (
	"strings"

	stickerhandler "kun-galgame-sticker-api/internal/platform/sticker/handler"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
)

// The public developer-platform face (NextMoe doc 08 §16, face name `sticker`).
//
// The gateway terminates authentication: Traefik runs ForwardAuth against
// oauth's /internal/devapi/forward-auth?face=sticker, which validates the
// nmk_ key, applies the key's rate and quota budget and meters the admitted
// request. Nothing reaches these routes unkeyed from the public internet, and
// the three X-NextMoe-* headers Traefik sets are the only trustworthy
// statement of who is calling. This service reads none of them -- every answer
// here is the same for every caller.
//
// There is deliberately no scope. The face shipped on 2026-09-08 behind a
// `sticker:read` scope and lost it the same day (infra PR #168): a free
// read-only B-tier face admits any valid key, because the gate's job here is
// identity, metering and rate limits, not authorization. The constant is gone
// from infra's vocabulary, so a key can no longer be minted holding it.
//
// Traefik does not rewrite the path, so the routes must be mounted on the
// public prefix verbatim.

// facePrefix is that public prefix. §16.2 first charted the downstream faces
// onto `/v1/<site>/*`, written while /v1 was the live public namespace; wave R3
// retired /v1 wholesale on 2026-08-27, and infra re-ruled the faces onto /v2 --
// where every live namespace already sits -- rather than open a new face on a
// prefix whose every sibling answers 410.
const facePrefix = "/v2/sticker"

func isFacePath(c fiber.Ctx) bool {
	return strings.HasPrefix(c.Path(), facePrefix)
}

// faceCache matches catalog /v2's public lane verbatim. Every response here is
// viewer-independent, so a shared cache is correct; the cost is that a request
// Cloudflare answers is a request the platform never meters. A problem
// document overrides it with no-store (see problem.Write).
const faceCache = "public, max-age=300, s-maxage=1800, stale-while-revalidate=3600"

// faceCORS is deliberately not the site's CORS. The site allows one origin and
// sends credentials; the face is a public read API with no cookies, so it
// allows any origin and no credentials. Preflight is answered here for
// completeness, though a browser cannot get one past the gateway: an OPTIONS
// request carries no API key, so ForwardAuth refuses it before it arrives.
func faceCORS() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{fiber.MethodGet, fiber.MethodOptions},
		AllowHeaders: []string{"X-API-Key", "Authorization", "Accept", "If-None-Match"},
		ExposeHeaders: []string{
			"ETag", "X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset",
			"X-Quota-Limit", "X-Quota-Remaining", "Retry-After",
		},
		MaxAge: 86400,
	})
}

func faceHeaders(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, faceCache)
	c.Set(fiber.HeaderVary, "Accept-Encoding, Origin")
	return c.Next()
}

// mountFace registers the read face. There is no local rate limiter: the only
// client address this service would ever see is Traefik's, so an IP limiter
// would put every application on one budget. The gateway holds the real
// per-key budget.
func mountFace(app *fiber.App, h *stickerhandler.Handler) {
	face := app.Group(facePrefix, faceCORS(), faceHeaders, etag.New())

	// Parameters carry the spec's names, which is what a problem document's
	// errors[0].parameter reports back.
	face.Get("/packs", h.FaceListPacks)
	face.Get("/packs/:pack_id", h.FaceGetPack)
	face.Get("/stickers/:sticker_id", h.FaceGetSticker)
	face.Get("/characters", h.FaceListCharacters)
	face.Get("/characters/:character_id", h.FaceGetCharacter)
	face.Get("/characters/:character_id/stickers", h.FaceCharacterStickers)
	face.Get("/works", h.FaceListWorks)
	face.Get("/works/:work_id/packs", h.FaceWorkPacks)
	face.Get("/tags", h.FaceListTags)
}
