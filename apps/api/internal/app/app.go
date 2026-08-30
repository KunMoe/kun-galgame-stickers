package app

import (
	"context"
	stderrors "errors"
	"log/slog"
	"strings"

	"kun-galgame-sticker-api/internal/infrastructure/database"
	"kun-galgame-sticker-api/internal/middleware"
	identityhandler "kun-galgame-sticker-api/internal/platform/identity/handler"
	"kun-galgame-sticker-api/internal/platform/identity/oauth"
	identityservice "kun-galgame-sticker-api/internal/platform/identity/service"
	stickerhandler "kun-galgame-sticker-api/internal/platform/sticker/handler"
	stickerrepo "kun-galgame-sticker-api/internal/platform/sticker/repository"
	stickerservice "kun-galgame-sticker-api/internal/platform/sticker/service"
	"kun-galgame-sticker-api/pkg/config"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/imageclient"
	"kun-galgame-sticker-api/pkg/response"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

type App struct {
	Fiber *fiber.App
}

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

	stickerSvc := stickerservice.New(stickerrepo.New(db), imgCli)
	stickerSvc.StartRefPing(context.Background())
	stickerH := stickerhandler.New(stickerSvc)

	fiberApp := fiber.New(fiber.Config{
		AppName:        "kun-galgame-sticker-api",
		ErrorHandler:   errorHandler,
		ReadBufferSize: 16 << 10,
		BodyLimit:      12 << 20,
	})
	fiberApp.Use(recover.New())
	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(cfg.CORS.AllowOrigins, ","),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	fiberApp.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	api := fiberApp.Group("/api/v1")
	api.Get("/sticker/packs", stickerH.ListPacks)

	api.Post("/auth/oauth/callback", authHandler.Callback)
	api.Post("/auth/logout", authHandler.Logout)

	opt := api.Group("", middleware.OptionalAuth(authSvc, cfg.Server.Secure))
	opt.Get("/auth/me", authHandler.Me)
	opt.Get("/sticker/packs/:sid", stickerH.GetPack)
	opt.Get("/sticker/packs/:sid/:pid", stickerH.GetOne)

	req := api.Group("", middleware.RequireAuth(authSvc, cfg.Server.Secure))
	req.Get("/sticker/me/packs", stickerH.ListMine)
	req.Post("/sticker/packs", stickerH.CreatePack)
	req.Patch("/sticker/packs/:sid", stickerH.PatchPack)
	req.Post("/sticker/packs/:sid/publish", stickerH.Publish)
	req.Post("/sticker/packs/:sid/unpublish", stickerH.Unpublish)
	req.Post("/sticker/packs/:sid/images", stickerH.UploadImage)
	req.Post("/sticker/packs/:sid/stickers", stickerH.AddSticker)
	req.Patch("/sticker/packs/:sid/stickers/:pid", stickerH.PatchSticker)
	req.Delete("/sticker/packs/:sid/stickers/:pid", stickerH.DeleteSticker)

	return &App{Fiber: fiberApp}
}

func errorHandler(c fiber.Ctx, err error) error {
	var appErr *errors.AppError
	if stderrors.As(err, &appErr) {
		return response.Error(c, appErr)
	}
	var fe *fiber.Error
	if stderrors.As(err, &fe) {
		return c.Status(fe.Code).JSON(fiber.Map{
			"code":    errors.CodeBiz,
			"message": fe.Message,
		})
	}
	slog.Error("unhandled", "error", err)
	return response.Error(c, errors.ErrInternal("internal error"))
}
