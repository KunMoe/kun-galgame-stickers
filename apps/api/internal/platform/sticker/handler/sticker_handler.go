package handler

import (
	"encoding/json"
	"strconv"

	"kun-galgame-sticker-api/internal/middleware"
	"kun-galgame-sticker-api/internal/platform/sticker/dto"
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

func (h *StickerHandler) ListMine(c fiber.Ctx) error {
	user, appErr := middleware.MustUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	packs, err := h.svc.ListMine(user.ID)
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
	pack, err := h.svc.GetPack(sid, viewerID(c))
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
	sticker, err := h.svc.GetOne(sid, pid, viewerID(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, sticker)
}

func (h *StickerHandler) CreatePack(c fiber.Ctx) error {
	user, appErr := middleware.MustUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.CreatePackRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return response.Error(c, errors.ErrBadRequest("invalid body"))
	}
	pack, err := h.svc.CreatePack(user.ID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, pack)
}

func (h *StickerHandler) PatchPack(c fiber.Ctx) error {
	user, appErr := middleware.MustUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	sid, parseErr := atoiParam(c.Params("sid"))
	if parseErr != nil {
		return response.Error(c, parseErr)
	}
	var req dto.PatchPackRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return response.Error(c, errors.ErrBadRequest("invalid body"))
	}
	pack, err := h.svc.PatchPack(sid, user.ID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, pack)
}

func (h *StickerHandler) Publish(c fiber.Ctx) error {
	return h.mutatePack(c, h.svc.Publish)
}

func (h *StickerHandler) Unpublish(c fiber.Ctx) error {
	return h.mutatePack(c, h.svc.Unpublish)
}

func (h *StickerHandler) mutatePack(c fiber.Ctx, fn func(int, int) (*dto.Pack, *errors.AppError)) error {
	user, appErr := middleware.MustUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	sid, parseErr := atoiParam(c.Params("sid"))
	if parseErr != nil {
		return response.Error(c, parseErr)
	}
	pack, err := fn(sid, user.ID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, pack)
}

func (h *StickerHandler) UploadImage(c fiber.Ctx) error {
	user, appErr := middleware.MustUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	sid, parseErr := atoiParam(c.Params("sid"))
	if parseErr != nil {
		return response.Error(c, parseErr)
	}
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		return response.Error(c, errors.ErrBadRequest("file is required"))
	}
	src, err := file.Open()
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("cannot read file"))
	}
	defer src.Close()
	result, appErr := h.svc.UploadImage(c.Context(), sid, user.ID, user.Sub, file.Filename, src, file.Size)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

func (h *StickerHandler) AddSticker(c fiber.Ctx) error {
	user, appErr := middleware.MustUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	sid, parseErr := atoiParam(c.Params("sid"))
	if parseErr != nil {
		return response.Error(c, parseErr)
	}
	var req dto.CreateStickerRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return response.Error(c, errors.ErrBadRequest("invalid body"))
	}
	sticker, err := h.svc.AddSticker(sid, user.ID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, sticker)
}

func (h *StickerHandler) PatchSticker(c fiber.Ctx) error {
	user, appErr := middleware.MustUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	sid, parseErr := atoiParam(c.Params("sid"))
	if parseErr != nil {
		return response.Error(c, parseErr)
	}
	pid, parseErr := atoiParam(c.Params("pid"))
	if parseErr != nil {
		return response.Error(c, parseErr)
	}
	var req dto.PatchStickerRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return response.Error(c, errors.ErrBadRequest("invalid body"))
	}
	sticker, err := h.svc.PatchSticker(sid, pid, user.ID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, sticker)
}

func (h *StickerHandler) DeleteSticker(c fiber.Ctx) error {
	user, appErr := middleware.MustUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	sid, parseErr := atoiParam(c.Params("sid"))
	if parseErr != nil {
		return response.Error(c, parseErr)
	}
	pid, parseErr := atoiParam(c.Params("pid"))
	if parseErr != nil {
		return response.Error(c, parseErr)
	}
	if err := h.svc.DeleteSticker(sid, pid, user.ID); err != nil {
		return response.Error(c, err)
	}
	return response.OKMessage(c, "deleted")
}

func viewerID(c fiber.Ctx) int {
	user := middleware.CurrentUser(c)
	if user == nil {
		return 0
	}
	return user.ID
}

func atoiParam(raw string) (int, *errors.AppError) {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.ErrNotFound("invalid id")
	}
	return n, nil
}
