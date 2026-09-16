package handler

import (
	"strconv"
	"strings"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) ListComments(c fiber.Ctx) error {
	packID, appErr := uuidParam(c, "packId")
	if appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.svc.PackComments(c.Context(), packID, strings.TrimSpace(c.Query("after")), viewer(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *Handler) AddComment(c fiber.Ctx) error {
	packID, appErr := uuidParam(c, "packId")
	if appErr != nil {
		return response.Error(c, appErr)
	}
	req, appErr := parseBody[dto.CommentRequest](c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	comment, appErr := h.svc.AddComment(c.Context(), packID, viewer(c), req.Body, req.ReplyTo)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, comment)
}

func (h *Handler) PatchComment(c fiber.Ctx) error {
	postID, appErr := commentIDParam(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	req, appErr := parseBody[dto.CommentRequest](c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	comment, appErr := h.svc.EditComment(c.Context(), postID, viewer(c), req.Body)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, comment)
}

func (h *Handler) DeleteComment(c fiber.Ctx) error {
	postID, appErr := commentIDParam(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.svc.DeleteComment(c.Context(), postID, viewer(c)); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, fiber.Map{"deleted": true})
}

func (h *Handler) ToggleCommentLike(c fiber.Ctx) error {
	postID, appErr := commentIDParam(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	result, appErr := h.svc.ToggleCommentLike(c.Context(), postID, viewer(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

func (h *Handler) FlagComment(c fiber.Ctx) error {
	postID, appErr := commentIDParam(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	req, appErr := parseBody[dto.CommentFlagRequest](c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.svc.FlagComment(c.Context(), postID, viewer(c), req.Reason, req.Note); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, fiber.Map{"reported": true})
}

// Comment ids are community's, not this site's: plain positive integers.
func commentIDParam(c fiber.Ctx) (int64, *errors.AppError) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Params("commentId")), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.ErrInvalidParams("commentId must be a positive integer")
	}
	return id, nil
}

// The unread, feed and search lanes are addressed by community's thread id
// rather than by a pack, because that is what the reader already holds: the
// comment page hands out thread_id, and turning a pack back into a thread
// would cost an upstream round trip to learn something the client knows.
func (h *Handler) MarkCommentsRead(c fiber.Ctx) error {
	threadID, appErr := threadIDParam(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	req, appErr := parseBody[dto.CommentReadRequest](c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	state, appErr := h.svc.MarkCommentsRead(c.Context(), threadID, req.PostNumber, viewer(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, state)
}

func (h *Handler) SetCommentNotification(c fiber.Ctx) error {
	threadID, appErr := threadIDParam(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	req, appErr := parseBody[dto.CommentNotificationRequest](c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	state, appErr := h.svc.SetCommentNotification(c.Context(), threadID, req.Level, viewer(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, state)
}

func (h *Handler) UnreadComments(c fiber.Ctx) error {
	unread, appErr := h.svc.UnreadComments(c.Context(), viewer(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, unread)
}

func (h *Handler) LatestComments(c fiber.Ctx) error {
	feed, appErr := h.svc.LatestComments(c.Context(), strings.TrimSpace(c.Query("cursor")))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, feed)
}

func (h *Handler) SearchComments(c fiber.Ctx) error {
	feed, appErr := h.svc.SearchComments(
		c.Context(), c.Query("q"), strings.TrimSpace(c.Query("cursor")),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, feed)
}

func threadIDParam(c fiber.Ctx) (int64, *errors.AppError) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Params("threadId")), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.ErrInvalidParams("threadId must be a positive integer")
	}
	return id, nil
}
