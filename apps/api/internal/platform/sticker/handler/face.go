package handler

import (
	"strconv"
	"strings"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/internal/platform/sticker/repository"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/problem"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// Handlers for the public developer-platform face. Two things separate them
// from the site's own handlers: the body is bare JSON rather than the house
// {code, message, data} envelope, and failures are RFC 9457 problem documents.
// Authentication is not among them -- the gateway terminates it, and by the
// time a request arrives here it has already been keyed, budgeted and metered.
//
// The parameter helpers below answer a *problem.Problem that is either nil or
// a refusal the caller still has to write. They must not write it themselves:
// the first version returned the result of problem.Write, which is nil on
// success, so `if err != nil` never fired, the handler ran the query anyway
// and the second write replaced the problem document with a success body
// under a 400 status line.

const (
	faceDefaultLimit = 20
	faceMaxLimit     = 100
)

// faceFail translates a service error into a problem document. The service
// speaks the site's error codes because it is shared with the site.
func faceFail(c fiber.Ctx, err *errors.AppError) error {
	return problem.Write(c, problem.OfStatus(err.StatusCode, err.Message))
}

func invalid(name, reason, detail string) *problem.Problem {
	return problem.Param(problem.CodeInvalidParameter, name, reason, detail)
}

// faceLimit rejects rather than clamps, unlike the site's own handlers: a
// third party writing a paging loop needs to be told its page size was
// refused, where a browser just wants the biggest page it may have.
// LIMIT_TOO_LARGE is catalog's code for exactly this.
func faceLimit(c fiber.Ctx) (int, *problem.Problem) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return faceDefaultLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, invalid("limit", problem.ReasonInvalidFormat, "limit must be an integer")
	}
	if n < 1 {
		return 0, invalid("limit", problem.ReasonOutOfRange, "limit must be at least 1")
	}
	if n > faceMaxLimit {
		return 0, problem.Param(problem.CodeLimitTooLarge, "limit", problem.ReasonOutOfRange,
			"limit must be at most "+strconv.Itoa(faceMaxLimit)+"; the value is not clamped")
	}
	return n, nil
}

func faceCursor(c fiber.Ctx) (int, *problem.Problem) {
	raw := strings.TrimSpace(c.Query("cursor"))
	if raw == "" {
		return 0, nil
	}
	offset, ok := dto.ParseFaceCursor(raw)
	if !ok {
		return 0, problem.Param(problem.CodeInvalidCursor, "cursor", problem.ReasonInvalidFormat,
			"pass the next_cursor from a previous page of this collection")
	}
	return offset, nil
}

// facePage reads the three parameters every collection on the face shares.
func facePage(c fiber.Ctx) (dto.FacePage, *problem.Problem) {
	limit, bad := faceLimit(c)
	if bad != nil {
		return dto.FacePage{}, bad
	}
	offset, bad := faceCursor(c)
	if bad != nil {
		return dto.FacePage{}, bad
	}
	includeTotal, _, bad := faceBool(c, "include_total")
	if bad != nil {
		return dto.FacePage{}, bad
	}
	return dto.FacePage{Offset: offset, Limit: limit, IncludeTotal: includeTotal}, nil
}

// faceBool is a closed vocabulary, as catalog's flags are: absent means the
// default, and anything other than true or false is a client error rather than
// a silent false.
func faceBool(c fiber.Ctx, name string) (value, set bool, bad *problem.Problem) {
	switch strings.TrimSpace(c.Query(name)) {
	case "":
		return false, false, nil
	case "true":
		return true, true, nil
	case "false":
		return false, true, nil
	}
	return false, false, invalid(name, problem.ReasonInvalidFormat, name+" must be true or false")
}

// faceCatalogID parses a catalog id from the path. Route parameters are named
// as the spec names them, so the name that fails is the name a caller reads.
func faceCatalogID(c fiber.Ctx, name string) (int64, *problem.Problem) {
	return parseCatalogID(name, c.Params(name))
}

// faceCatalogFilter is faceCatalogID for an optional query filter. It refuses
// a malformed value instead of dropping the filter, which would answer the
// unfiltered collection as though it were the filtered one.
func faceCatalogFilter(c fiber.Ctx, name string) (int64, *problem.Problem) {
	raw := c.Query(name)
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	return parseCatalogID(name, raw)
}

func parseCatalogID(name, raw string) (int64, *problem.Problem) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, invalid(name, problem.ReasonInvalidFormat, name+" must be a positive decimal id")
	}
	return id, nil
}

func faceUUID(c fiber.Ctx, name string) (uuid.UUID, *problem.Problem) {
	id, appErr := uuidParam(c, name)
	if appErr != nil {
		return uuid.Nil, invalid(name, problem.ReasonInvalidFormat, appErr.Message)
	}
	return id, nil
}

func faceSort(c fiber.Ctx) (string, *problem.Problem) {
	raw := strings.TrimSpace(c.Query("sort"))
	if raw != "" && raw != dto.SortNew && raw != dto.SortHot {
		return "", invalid("sort", problem.ReasonUnknownValue, "sort must be new or hot")
	}
	return sortOf(raw), nil
}

// faceRatingFilter turns the nsfw flag into this site's rating filter. Absent
// or false hides r18 packs, matching catalog's nsfw parameter exactly.
func faceRatingFilter(c fiber.Ctx) (string, *problem.Problem) {
	nsfw, _, bad := faceBool(c, "nsfw")
	if bad != nil {
		return "", bad
	}
	if nsfw {
		return dto.RatingFilterAll, nil
	}
	return dto.RatingFilterSFW, nil
}

func (h *Handler) FaceListPacks(c fiber.Ctx) error {
	work, bad := faceCatalogFilter(c, "work")
	if bad != nil {
		return problem.Write(c, bad)
	}
	return h.faceListPacks(c, work)
}

// faceListPacks is shared by /packs and /works/{work_id}/packs, which differ
// only in where the work filter comes from.
func (h *Handler) faceListPacks(c fiber.Ctx, work int64) error {
	page, bad := facePage(c)
	if bad != nil {
		return problem.Write(c, bad)
	}
	sort, bad := faceSort(c)
	if bad != nil {
		return problem.Write(c, bad)
	}
	rating, bad := faceRatingFilter(c)
	if bad != nil {
		return problem.Write(c, bad)
	}
	official, officialSet, bad := faceBool(c, "official")
	if bad != nil {
		return problem.Write(c, bad)
	}
	linked, linkedSet, bad := faceBool(c, "linked")
	if bad != nil {
		return problem.Write(c, bad)
	}

	query := dto.ListQuery{
		Sort:             sort,
		Search:           faceSearch(c),
		Tag:              repository.Slugify(c.Query("tag")),
		Rating:           rating,
		OfficialOnly:     officialSet && official,
		LinkedOnly:       linkedSet && linked,
		AnyCatalogWorkID: work,
	}
	out, appErr := h.svc.FaceListPacks(c.Context(), query, page)
	if appErr != nil {
		return faceFail(c, appErr)
	}
	return c.JSON(out)
}

func (h *Handler) FaceGetPack(c fiber.Ctx) error {
	id, bad := faceUUID(c, "pack_id")
	if bad != nil {
		return problem.Write(c, bad)
	}
	out, appErr := h.svc.FaceGetPack(c.Context(), id)
	if appErr != nil {
		return faceFail(c, appErr)
	}
	return c.JSON(out)
}

func (h *Handler) FaceGetSticker(c fiber.Ctx) error {
	id, bad := faceUUID(c, "sticker_id")
	if bad != nil {
		return problem.Write(c, bad)
	}
	out, appErr := h.svc.FaceGetSticker(id)
	if appErr != nil {
		return faceFail(c, appErr)
	}
	return c.JSON(out)
}

func (h *Handler) FaceListCharacters(c fiber.Ctx) error {
	page, bad := facePage(c)
	if bad != nil {
		return problem.Write(c, bad)
	}
	work, bad := faceCatalogFilter(c, "work")
	if bad != nil {
		return problem.Write(c, bad)
	}
	out, appErr := h.svc.FaceCharacters(faceSearch(c), work, page)
	if appErr != nil {
		return faceFail(c, appErr)
	}
	return c.JSON(out)
}

func (h *Handler) FaceGetCharacter(c fiber.Ctx) error {
	id, bad := faceCatalogID(c, "character_id")
	if bad != nil {
		return problem.Write(c, bad)
	}
	out, appErr := h.svc.FaceCharacter(id)
	if appErr != nil {
		return faceFail(c, appErr)
	}
	return c.JSON(out)
}

func (h *Handler) FaceCharacterStickers(c fiber.Ctx) error {
	id, bad := faceCatalogID(c, "character_id")
	if bad != nil {
		return problem.Write(c, bad)
	}
	page, bad := facePage(c)
	if bad != nil {
		return problem.Write(c, bad)
	}
	out, appErr := h.svc.FaceCharacterStickers(id, page)
	if appErr != nil {
		return faceFail(c, appErr)
	}
	return c.JSON(out)
}

func (h *Handler) FaceListWorks(c fiber.Ctx) error {
	page, bad := facePage(c)
	if bad != nil {
		return problem.Write(c, bad)
	}
	out, appErr := h.svc.FaceWorks(faceSearch(c), page)
	if appErr != nil {
		return faceFail(c, appErr)
	}
	return c.JSON(out)
}

// FaceWorkPacks is /packs?work= under the path a caller holding a catalog work
// id reaches for. "About this work" means the pack declares it or holds a
// sticker of it -- the same relation the work index counts.
func (h *Handler) FaceWorkPacks(c fiber.Ctx) error {
	id, bad := faceCatalogID(c, "work_id")
	if bad != nil {
		return problem.Write(c, bad)
	}
	return h.faceListPacks(c, id)
}

func (h *Handler) FaceListTags(c fiber.Ctx) error {
	page, bad := facePage(c)
	if bad != nil {
		return problem.Write(c, bad)
	}
	out, appErr := h.svc.FaceTags(page)
	if appErr != nil {
		return faceFail(c, appErr)
	}
	return c.JSON(out)
}

func faceSearch(c fiber.Ctx) string {
	return truncate(strings.TrimSpace(c.Query("q")), maxSearchLen)
}
