// Package problem writes RFC 9457 problem details.
//
// This is the error language of the public developer-platform faces, not of
// this site's own BFF: /api/v1 keeps the house {code, message, data} envelope,
// while /v2/sticker answers like catalog's /v2 does. A third party holding one
// nmk_ key across both faces should not have to decode two error dialects, and
// the gateway's own refusals (401/429 from ForwardAuth) are the only ones this
// service never gets to shape.
package problem

import (
	"crypto/rand"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Codes are reused verbatim from infra's closed registry
// (apps/api/internal/platform/apiv2/problem/registry.go) so a client can share
// one decoder. Only the subset a read-only face can actually produce is here.
const (
	CodeInvalidParameter   = "INVALID_PARAMETER"
	CodeLimitTooLarge      = "LIMIT_TOO_LARGE"
	CodeInvalidCursor      = "INVALID_CURSOR"
	CodeNotFound           = "NOT_FOUND"
	CodeMethodNotAllowed   = "METHOD_NOT_ALLOWED"
	CodeInternalError      = "INTERNAL_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// Field-level reasons, from the same registry.
const (
	ReasonInvalidFormat = "INVALID_FORMAT"
	ReasonOutOfRange    = "OUT_OF_RANGE"
	ReasonUnknownValue  = "UNKNOWN_VALUE"
)

// TypeBase is infra's platform domain. Every code above is registered there,
// so the type URI resolves to the portal's own page for it; this face has no
// problem types of its own to justify a sticker namespace.
const TypeBase = "https://developer.nextmoe.dev/problems/platform/"

// ContentType is RFC 9457's media type. Exported because a caller that builds
// its own document (a test, most often) should not spell it again.
const ContentType = "application/problem+json"

// HeaderRequestID carries the same value as the document's request_id.
const HeaderRequestID = "X-Request-ID"

type definition struct {
	title  string
	status int
	slug   string
}

var registry = map[string]definition{
	CodeInvalidParameter:   {"Invalid parameter", fiber.StatusBadRequest, "invalid-parameter"},
	CodeLimitTooLarge:      {"Limit too large", fiber.StatusBadRequest, "limit-too-large"},
	CodeInvalidCursor:      {"Invalid cursor", fiber.StatusBadRequest, "invalid-cursor"},
	CodeNotFound:           {"Not found", fiber.StatusNotFound, "not-found"},
	CodeMethodNotAllowed:   {"Method not allowed", fiber.StatusMethodNotAllowed, "method-not-allowed"},
	CodeInternalError:      {"Internal error", fiber.StatusInternalServerError, "internal-error"},
	CodeServiceUnavailable: {"Service unavailable", fiber.StatusServiceUnavailable, "service-unavailable"},
}

// FieldError names the parameter that failed. Infra's shape also allows a
// body pointer or a header in its place; a read face takes neither.
type FieldError struct {
	Parameter string `json:"parameter"`
	Reason    string `json:"reason"`
	Detail    string `json:"detail"`
}

// Problem is the wire shape. Field names and their meanings match infra's
// Problem so the two faces decode the same way; the members that only make
// sense for a write face (object, current_id, suspects) are left out rather
// than emitted permanently empty. Detail, request_id and errors are always
// present, errors as [] when the failure is not about one parameter.
type Problem struct {
	Type      string       `json:"type"`
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Detail    string       `json:"detail"`
	Instance  string       `json:"instance"`
	Code      string       `json:"code"`
	RequestID string       `json:"request_id"`
	Errors    []FieldError `json:"errors"`
}

// New builds a problem without writing it. An unregistered code becomes an
// internal error rather than a half-filled document.
func New(code, detail string) *Problem {
	def, ok := registry[code]
	if !ok {
		code, def = CodeInternalError, registry[CodeInternalError]
	}
	return &Problem{
		Type:   TypeBase + def.slug,
		Title:  def.title,
		Status: def.status,
		Detail: detail,
		Code:   code,
		Errors: []FieldError{},
	}
}

// Param is how every 400 on the face is built: errors[0] names the parameter
// the caller has to fix.
func Param(code, name, reason, detail string) *Problem {
	p := New(code, detail)
	p.Errors = []FieldError{{Parameter: name, Reason: reason, Detail: detail}}
	return p
}

// OfStatus maps a status the site's own error vocabulary produced onto the
// registry, for failures that surface from the service layer or the router.
func OfStatus(status int, detail string) *Problem {
	switch status {
	case fiber.StatusBadRequest:
		return New(CodeInvalidParameter, detail)
	case fiber.StatusNotFound:
		return New(CodeNotFound, detail)
	case fiber.StatusMethodNotAllowed:
		return New(CodeMethodNotAllowed, detail)
	case fiber.StatusServiceUnavailable:
		return New(CodeServiceUnavailable, detail)
	default:
		return New(CodeInternalError, detail)
	}
}

// Write answers with a problem document, filling in the members only the edge
// knows.
func Write(c fiber.Ctx, p *Problem) error {
	p.Instance = instance(c)
	p.RequestID = newRequestID()
	c.Set(HeaderRequestID, p.RequestID)
	// The face's shared-cache policy is for answers that are the same for
	// every caller. A problem carries this request's id, and a 5xx must not be
	// replayed from an edge cache after the fault has cleared.
	c.Set(fiber.HeaderCacheControl, "no-store")
	if p.Status >= fiber.StatusInternalServerError {
		// The id is only worth quoting in a support request if it resolves
		// somewhere; the gateway never sees a response body.
		slog.Error("face problem", "request_id", p.RequestID, "code", p.Code,
			"instance", p.Instance, "detail", p.Detail)
	}
	// The content type is JSON's second argument, not a header set beforehand:
	// c.JSON overwrites Content-Type, which quietly turned every problem
	// document back into application/json.
	return c.Status(p.Status).JSON(p, ContentType)
}

// instance carries the query string: for a read face, which filter combination
// failed is most of the diagnostic value.
func instance(c fiber.Ctx) string {
	path := c.Path()
	if raw := string(c.Request().URI().QueryString()); raw != "" {
		return path + "?" + raw
	}
	return path
}

// crockford is the ULID alphabet: base32 without I, L, O and U.
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// newRequestID mints infra's shape, `req_` plus a 26-character ULID, so a
// caller reporting a failure quotes one kind of identifier whichever face
// produced it. The layout is a real ULID: 10 characters of 48-bit millisecond
// timestamp (the top two bits padding), then 16 of 80-bit randomness.
func newRequestID() string {
	out := make([]byte, 0, 30)
	out = append(out, "req_"...)

	ms := uint64(time.Now().UnixMilli())
	for shift := 45; shift >= 0; shift -= 5 {
		out = append(out, crockford[(ms>>shift)&0x1f])
	}

	var rnd [10]byte
	_, _ = rand.Read(rnd[:]) // never fails since Go 1.24
	var acc, bits uint16
	for _, b := range rnd {
		acc = acc<<8 | uint16(b)
		for bits += 8; bits >= 5; bits -= 5 {
			out = append(out, crockford[(acc>>(bits-5))&0x1f])
		}
	}
	return string(out)
}
