package handler

import (
	"strings"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) ListNotifications(c fiber.Ctx) error {
	page, appErr := h.svc.Notifications(c.Context(), viewer(c), strings.TrimSpace(c.Query("cursor")))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *Handler) UnreadNotificationCount(c fiber.Ctx) error {
	count, appErr := h.svc.UnreadNotificationCount(c.Context(), viewer(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, count)
}

func (h *Handler) MarkNotificationsRead(c fiber.Ctx) error {
	req, appErr := parseBody[dto.NotificationReadRequest](c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	count, appErr := h.svc.MarkNotificationsRead(c.Context(), viewer(c), req.IDs, req.All)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, count)
}
