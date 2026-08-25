package handler

import (
	"strconv"

	"kun-galgame-sticker-api/internal/platform/sticker/service"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type StickerHandler struct {
	svc *service.StickerService
}

func New(svc *service.StickerService) *StickerHandler {
	return &StickerHandler{svc: svc}
}

func (h *StickerHandler) ListPacks(c fiber.Ctx) error {
	packs, err := h.svc.ListPacks()
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, packs)
}

func (h *StickerHandler) GetPack(c fiber.Ctx) error {
	sid, parseErr := atoiParam(c.Params("sid"))
	if parseErr != nil {
		return response.Error(c, parseErr)
	}
	pack, err := h.svc.GetPack(sid)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, pack)
}

func (h *StickerHandler) GetOne(c fiber.Ctx) error {
	sid, parseErr := atoiParam(c.Params("sid"))
	if parseErr != nil {
		return response.Error(c, parseErr)
	}
	pid, parseErr := atoiParam(c.Params("pid"))
	if parseErr != nil {
		return response.Error(c, parseErr)
	}
	sticker, err := h.svc.GetOne(sid, pid)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, sticker)
}

func atoiParam(raw string) (int, *errors.AppError) {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.ErrNotFound("invalid id")
	}
	return n, nil
}
