package app

import (
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

	stickerSvc := stickerservice.New(stickerrepo.New(db))
	stickerH := stickerhandler.New(stickerSvc)

	fiberApp := fiber.New(fiber.Config{
		AppName:        "kun-galgame-sticker-api",
		ErrorHandler:   errorHandler,
		ReadBufferSize: 16 << 10,
	})
	fiberApp.Use(recover.New())
	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(cfg.CORS.AllowOrigins, ","),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	fiberApp.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	api := fiberApp.Group("/api/v1")
	api.Get("/sticker/packs", stickerH.ListPacks)
	api.Get("/sticker/packs/:sid", stickerH.GetPack)
	api.Get("/sticker/packs/:sid/:pid", stickerH.GetOne)

	api.Post("/auth/oauth/callback", authHandler.Callback)
	api.Post("/auth/logout", authHandler.Logout)

	authed := api.Group("", middleware.OptionalAuth(authSvc, cfg.Server.Secure))
	authed.Get("/auth/me", authHandler.Me)

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


