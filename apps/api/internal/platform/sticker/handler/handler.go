package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"kun-galgame-sticker-api/internal/middleware"
	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/internal/platform/sticker/repository"
	"kun-galgame-sticker-api/internal/platform/sticker/service"
	"kun-galgame-sticker-api/pkg/errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	defaultLimit = 24
	maxLimit     = 50
	maxSearchLen = 100
	maxTagsShown = 40
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func viewer(c fiber.Ctx) service.Viewer {
	user := middleware.CurrentUser(c)
	if user == nil {
		return service.Viewer{}
	}
	return service.Viewer{UID: user.ID, Sub: user.Sub, Roles: user.Roles}
}

func uuidParam(c fiber.Ctx, name string) (uuid.UUID, *errors.AppError) {
	id, err := uuid.Parse(strings.TrimSpace(c.Params(name)))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, errors.ErrInvalidParams(name + " must be a uuid")
	}
	return id, nil
}

func parseBody[T any](c fiber.Ctx) (T, *errors.AppError) {
	var out T
	if len(c.Body()) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(c.Body(), &out); err != nil {
		return out, errors.ErrInvalidParams("malformed json body")
	}
	return out, nil
}

// listQuery clamps instead of rejecting: a browser that asks for limit=5000
// gets the biggest page it is allowed, not a 400 it cannot act on.
func listQuery(c fiber.Ctx) dto.ListQuery {
	page := clamp(c.Query("page"), 1, 1, 10000)
	limit := clamp(c.Query("limit"), defaultLimit, 1, maxLimit)
	return dto.ListQuery{
		Offset:        (page - 1) * limit,
		Limit:         limit,
		Sort:          sortOf(c.Query("sort")),
		Search:        truncate(strings.TrimSpace(c.Query("q")), maxSearchLen),
		Tag:           repository.Slugify(c.Query("tag")),
		Rating:        ratingOf(c.Query("rating")),
		OfficialOnly:  isTrue(c.Query("official")),
		LinkedOnly:    isTrue(c.Query("linked")),
		CatalogWorkID: catalogID(c.Query("work")),
	}
}

func sortOf(raw string) string {
	if raw == dto.SortHot {
		return dto.SortHot
	}
	return dto.SortNew
}

func ratingOf(raw string) string {
	if raw == dto.RatingFilterAll {
		return dto.RatingFilterAll
	}
	return dto.RatingFilterSFW
}

func isTrue(raw string) bool { return raw == "1" || raw == "true" }

// catalogID parses a catalog identifier. They are decimal and positive; zero
// is the documented "no link", so it never reaches a query as a filter.
func catalogID(raw string) int64 {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

func clamp(raw string, fallback, low, high int) int {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return min(max(n, low), high)
}

func truncate(value string, maxRunes int) string {
	if runes := []rune(value); len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}
	return value
}
