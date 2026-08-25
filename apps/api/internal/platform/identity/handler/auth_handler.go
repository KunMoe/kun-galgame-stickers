package handler

import (
	"encoding/json"

	"kun-galgame-sticker-api/internal/middleware"
	"kun-galgame-sticker-api/internal/platform/identity/dto"
	"kun-galgame-sticker-api/internal/platform/identity/service"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type AuthHandler struct {
	svc    *service.AuthService
	secure bool
}

func New(svc *service.AuthService, secure bool) *AuthHandler {
	return &AuthHandler{svc: svc, secure: secure}
}

func (h *AuthHandler) Callback(c fiber.Ctx) error {
	var req dto.CallbackRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return response.Error(c, errors.ErrBadRequest("invalid body"))
	}
	tokens, user, appErr := h.svc.Callback(req.Code, req.CodeVerifier)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	middleware.PersistSession(c, tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresIn, user, h.secure)
	return response.OK(c, user)
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	refresh := c.Cookies(middleware.CookieRefresh)
	h.svc.Revoke(refresh)
	middleware.ClearSession(c, h.secure)
	return response.OKMessage(c, "已登出")
}

func (h *AuthHandler) Me(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return response.Error(c, errors.ErrUnauthorized("not signed in"))
	}
	return response.OK(c, user)
}
