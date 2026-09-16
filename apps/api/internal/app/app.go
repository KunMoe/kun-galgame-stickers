package app

import (
	"context"
	stderrors "errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"kun-galgame-sticker-api/internal/infrastructure/database"
	"kun-galgame-sticker-api/internal/middleware"
	identityhandler "kun-galgame-sticker-api/internal/platform/identity/handler"
	"kun-galgame-sticker-api/internal/platform/identity/oauth"
	identityservice "kun-galgame-sticker-api/internal/platform/identity/service"
	stickerhandler "kun-galgame-sticker-api/internal/platform/sticker/handler"
	stickerrepo "kun-galgame-sticker-api/internal/platform/sticker/repository"
	stickerservice "kun-galgame-sticker-api/internal/platform/sticker/service"
	"kun-galgame-sticker-api/pkg/catalogclient"
	"kun-galgame-sticker-api/pkg/communityclient"
	"kun-galgame-sticker-api/pkg/config"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/imageclient"
	"kun-galgame-sticker-api/pkg/problem"
	"kun-galgame-sticker-api/pkg/response"
	"kun-galgame-sticker-api/pkg/userclient"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

type App struct {
	Fiber *fiber.App
	stop  context.CancelFunc
}

// Close stops the background reference ping. Without it the goroutine outlived
// the process shutdown and kept a database handle open.
func (a *App) Close() { a.stop() }

func New(cfg *config.Config) *App {
	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	oauthClient := oauth.NewClient(cfg.OAuth)
	authSvc := identityservice.New(oauthClient)
	authHandler := identityhandler.New(authSvc, cfg.Server.Secure)

	var imgCli *imageclient.Client
	if cfg.Image.BaseURL != "" {
		imgCli = imageclient.New(imageclient.Config{
			BaseURL:      cfg.Image.BaseURL,
			CDNBase:      cfg.Image.CDNBase,
			ClientID:     cfg.OAuth.ClientID,
			ClientSecret: cfg.OAuth.ClientSecret,
		})
	}

	users := userclient.New(userclient.Config{
		BaseURL:      cfg.OAuth.ServerURL,
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		ImageCDNBase: cfg.Image.CDNBase,
	})

	// catalog is optional: with no key the pickers answer 503 and every page
	// falls back to the free-text game and character names already stored.
	catalog := catalogclient.New(catalogclient.Config{
		BaseURL: cfg.Catalog.BaseURL,
		APIKey:  cfg.Catalog.APIKey,
	})
	if !catalog.Configured() {
		slog.Warn("catalog not configured: game and character linking is disabled")
	}

	// Comments live in the infra community service. Unconfigured means a pack
	// page simply has no comment section.
	community := communityclient.New(communityclient.Config{
		BaseURL:      cfg.Community.BaseURL,
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
	})
	if !community.Configured() {
		slog.Warn("community not configured: pack comments are disabled")
	}

	stickerSvc := stickerservice.New(
		stickerrepo.NewPackRepo(db),
		stickerrepo.NewStickerRepo(db),
		stickerrepo.NewTagRepo(db),
		stickerrepo.NewCommentLikeRepo(db),
		imgCli,
		users,
		catalog,
		community,
	)
	ctx, stop := context.WithCancel(context.Background())
	stickerSvc.StartRefPing(ctx)
	h := stickerhandler.New(stickerSvc)

	fiberApp := fiber.New(fiber.Config{
		AppName:        "kun-galgame-sticker-api",
		ErrorHandler:   errorHandler,
		ReadBufferSize: 16 << 10,
		BodyLimit:      12 << 20,
	})
	fiberApp.Use(recover.New())
	fiberApp.Use(cors.New(cors.Config{
		// The public face brings its own CORS: it answers any origin and sends
		// no credentials, which this policy cannot express at the same time.
		Next:             isFacePath,
		AllowOrigins:     strings.Split(cfg.CORS.AllowOrigins, ","),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	fiberApp.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	// Auth middleware is attached per route, never with Group(prefix, mw).
	// fiber.Group registers its handlers as methodUse on the prefix, so they
	// run for every route under it regardless of which Router registered them:
	// an OptionalAuth group followed by a RequireAuth group made every write
	// resolve the session twice, and on an expired access token that meant two
	// refreshes -- the second one presenting a token the OP had already
	// rotated, which logged the user out mid-write.
	optionalAuth := middleware.OptionalAuth(authSvc, cfg.Server.Secure)
	requireAuth := middleware.RequireAuth(authSvc, cfg.Server.Secure)
	readLimit := publicLimiter()
	writeLimit := userLimiter(120)
	uploadLimit := userLimiter(60)
	cacheable := publicCache(60)

	// The public developer-platform face, mounted on its public prefixes before
	// the site's own group so the two are visibly separate things. See face.go.
	mountFace(fiberApp, h)

	api := fiberApp.Group("/api/v1")

	// Fully public, viewer-independent, and therefore cacheable.
	api.Get("/packs", readLimit, cacheable, etag.New(), h.ListPacks)
	api.Get("/tags", readLimit, cacheable, etag.New(), h.ListTags)

	// The ecosystem's default-avatar manifest. Its cache window is an order of
	// magnitude longer than the rest of this group because the pool changes by
	// hand, perhaps twice a year, while the packs list changes whenever anyone
	// publishes. stale-while-revalidate is the load-bearing part: if this
	// service is down, every consuming site keeps serving avatars from a week
	// old copy instead of falling back.
	api.Get("/avatar-pool", readLimit, avatarPoolCache, etag.New(), h.AvatarPool)

	// The sticker-picker payload other sites' editors render, in one response
	// rather than the face's list-then-fetch-each: a picker needs every
	// sticker of every pack at once, and 1+N round trips to build one panel is
	// what made every consumer hardcode the URLs in the first place.
	api.Get("/editor-packs", readLimit, avatarPoolCache, etag.New(), h.EditorPacks)

	// Public, but an author also sees their own drafts here.
	api.Get("/packs/:packId", readLimit, optionalAuth, h.GetPack)
	api.Get("/packs/:packId/download", readLimit, optionalAuth, h.DownloadPack)
	api.Get("/stickers/:stickerId", readLimit, optionalAuth, h.GetSticker)
	api.Get("/stickers/:stickerId/download", readLimit, optionalAuth, h.DownloadSticker)
	api.Get("/users/:uid/packs", readLimit, optionalAuth, h.ListUserPacks)
	api.Get("/characters/:characterId", readLimit, optionalAuth, h.GetCharacter)
	api.Get("/search", readLimit, h.Search)

	// Comments hang off the pack, because that is how they are addressed
	// upstream -- there is no thread id stored here to route by.
	api.Get("/packs/:packId/comments", readLimit, optionalAuth, h.ListComments)
	api.Post("/packs/:packId/comments", requireAuth, writeLimit, h.AddComment)
	api.Patch("/comments/:commentId", requireAuth, writeLimit, h.PatchComment)
	api.Delete("/comments/:commentId", requireAuth, writeLimit, h.DeleteComment)
	api.Post("/comments/:commentId/like", requireAuth, writeLimit, h.ToggleCommentLike)
	api.Post("/comments/:commentId/report", requireAuth, writeLimit, h.FlagComment)

	// Site-wide comment reads. They are not cacheable: the feed changes with
	// every comment anywhere, and the search lane is a round trip to community.
	api.Get("/comments/latest", readLimit, h.LatestComments)
	api.Get("/comments/search", readLimit, h.SearchComments)

	// Unread and subscription. A read receipt is a write on purpose -- a GET
	// that marks a thread read cannot be prefetched or retried safely -- so the
	// two receipts sit behind the write limiter. The badge itself is a read,
	// but a costly one: the header asks for it on every page, and each ask is
	// an upstream page walk.
	api.Get("/me/comments/unread", requireAuth, readLimit, h.UnreadComments)
	api.Post("/comments/threads/:threadId/read", requireAuth, writeLimit, h.MarkCommentsRead)
	api.Post("/comments/threads/:threadId/notification", requireAuth, writeLimit, h.SetCommentNotification)

	// The catalog pickers sit behind auth: the application key must never
	// reach a browser, and only an author composing a pack needs them. They
	// carry the upload limiter because each call is an upstream request.
	api.Get("/catalog/works", requireAuth, uploadLimit, h.SearchCatalogWorks)
	api.Get("/catalog/works/:workId/characters", requireAuth, uploadLimit, h.CatalogWorkRoster)

	api.Post("/auth/oauth/callback", writeLimit, authHandler.Callback)
	api.Post("/auth/logout", authHandler.Logout)
	api.Get("/auth/me", optionalAuth, authHandler.Me)

	api.Get("/me/packs", requireAuth, h.ListMyPacks)
	api.Post("/me/packs", requireAuth, writeLimit, h.CreatePack)
	api.Patch("/me/packs/:packId", requireAuth, writeLimit, h.PatchPack)
	api.Delete("/me/packs/:packId", requireAuth, writeLimit, h.DeletePack)
	api.Post("/me/packs/:packId/publish", requireAuth, writeLimit, h.Publish)
	api.Post("/me/packs/:packId/unpublish", requireAuth, writeLimit, h.Unpublish)
	api.Post("/me/packs/:packId/images", requireAuth, uploadLimit, h.UploadImage)
	api.Post("/me/packs/:packId/stickers", requireAuth, writeLimit, h.AddSticker)
	api.Patch("/me/packs/:packId/stickers/:stickerId", requireAuth, writeLimit, h.PatchSticker)
	api.Delete("/me/packs/:packId/stickers/:stickerId", requireAuth, writeLimit, h.DeleteSticker)
	api.Put("/me/packs/:packId/stickers/order", requireAuth, writeLimit, h.ReorderStickers)

	return &App{Fiber: fiberApp, stop: stop}
}

func publicLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:          300,
		Expiration:   time.Minute,
		LimitReached: tooManyRequests,
	})
}

// userLimiter counts per signed-in user, falling back to the client address for
// anonymous callers -- a whole office behind one NAT would otherwise share a
// single write budget.
func userLimiter(perMinute int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        perMinute,
		Expiration: time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			if user := middleware.CurrentUser(c); user != nil {
				return "u" + strconv.Itoa(user.ID)
			}
			return c.IP()
		},
		LimitReached: tooManyRequests,
	})
}

func tooManyRequests(c fiber.Ctx) error {
	return response.Error(c, errors.New(errors.CodeBiz, "too many requests", 429))
}

// avatarPoolCache is deliberately not publicCache(n): the numbers differ by
// three orders of magnitude and writing them apart keeps anyone from "tidying"
// the pool onto the 60s window that suits a content list.
func avatarPoolCache(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "public, max-age=3600, s-maxage=86400, stale-while-revalidate=604800")
	return c.Next()
}

func publicCache(seconds int) fiber.Handler {
	value := "public, max-age=" + strconv.Itoa(seconds)
	return func(c fiber.Ctx) error {
		c.Set(fiber.HeaderCacheControl, value)
		return c.Next()
	}
}

// errorHandler speaks whichever error language the path belongs to. Without
// the face branch an unrouted /v2/sticker path would answer with the site's
// house envelope, which is exactly the second error dialect the face exists to
// avoid.
func errorHandler(c fiber.Ctx, err error) error {
	var appErr *errors.AppError
	if stderrors.As(err, &appErr) {
		if isFacePath(c) {
			return faceProblem(c, appErr.StatusCode, appErr.Message)
		}
		return response.Error(c, appErr)
	}
	var fe *fiber.Error
	if stderrors.As(err, &fe) {
		if isFacePath(c) {
			return faceProblem(c, fe.Code, fe.Message)
		}
		return response.Error(c, errors.New(errors.CodeBiz, fe.Message, fe.Code))
	}
	slog.Error("unhandled", "error", err)
	if isFacePath(c) {
		return faceProblem(c, fiber.StatusInternalServerError, "internal error")
	}
	return response.Error(c, errors.ErrInternal("internal error"))
}

func faceProblem(c fiber.Ctx, status int, detail string) error {
	switch status {
	case fiber.StatusNotFound:
		return problem.Write(c, problem.CodeNotFound, detail)
	case fiber.StatusBadRequest:
		return problem.Write(c, problem.CodeInvalidParameter, detail)
	case fiber.StatusServiceUnavailable:
		return problem.Write(c, problem.CodeServiceUnavailable, detail)
	default:
		return problem.Write(c, problem.CodeInternalError, detail)
	}
}
