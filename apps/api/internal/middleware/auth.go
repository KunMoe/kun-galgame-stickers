package middleware

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"kun-galgame-sticker-api/internal/platform/identity/dto"
	"kun-galgame-sticker-api/internal/platform/identity/service"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

const (
	CookieAccess  = "kun_oauth_access"
	CookieRefresh = "kun_oauth_refresh"
	CookieUser    = "kun_oauth_user"

	refreshTTLSec      = 60 * 60 * 24 * 7
	accessSafetyWindow = 30
	userLocalsKey      = "sticker_user"
)

type jwtClaims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
}

func OptionalAuth(auth *service.AuthService, secure bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := resolveUser(c, auth, secure)
		if user != nil {
			c.Locals(userLocalsKey, user)
		}
		return c.Next()
	}
}

func CurrentUser(c fiber.Ctx) *dto.User {
	user, _ := c.Locals(userLocalsKey).(*dto.User)
	return user
}

func MustUser(c fiber.Ctx) (*dto.User, *errors.AppError) {
	user := CurrentUser(c)
	if user == nil {
		return nil, errors.ErrUnauthorized("not signed in")
	}
	return user, nil
}

func RequireAuth(auth *service.AuthService, secure bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := resolveUser(c, auth, secure)
		if user == nil {
			return response.Error(c, errors.ErrUnauthorized("not signed in"))
		}
		c.Locals(userLocalsKey, user)
		return c.Next()
	}
}

func resolveUser(c fiber.Ctx, auth *service.AuthService, secure bool) *dto.User {
	access := c.Cookies(CookieAccess)
	now := time.Now().Unix()

	if access != "" {
		claims := decodeJWT(access)
		if claims != nil && claims.Exp > now+accessSafetyWindow {
			if cached := c.Cookies(CookieUser); cached != "" {
				var user dto.User
				if json.Unmarshal([]byte(cached), &user) == nil {
					return &user
				}
			}
			user, err := auth.FetchUser(access)
			if err != nil {
				return nil
			}
			persistUser(c, user, int(claims.Exp-now), secure)
			return user
		}
	}

	refresh := c.Cookies(CookieRefresh)
	if refresh == "" {
		if c.Cookies(CookieUser) != "" {
			clearSession(c, secure)
		}
		return nil
	}

	tokens, user, err := auth.Refresh(refresh)
	if err != nil {
		clearSession(c, secure)
		return nil
	}
	persistTokens(c, tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresIn, secure)
	persistUser(c, user, tokens.ExpiresIn, secure)
	return user
}

func persistTokens(c fiber.Ctx, access, refresh string, expiresIn int, secure bool) {
	setCookie(c, CookieAccess, access, expiresIn, secure)
	setCookie(c, CookieRefresh, refresh, refreshTTLSec, secure)
}

func persistUser(c fiber.Ctx, user *dto.User, maxAge int, secure bool) {
	raw, err := json.Marshal(user)
	if err != nil {
		return
	}
	setCookie(c, CookieUser, string(raw), maxAge, secure)
}

func PersistSession(c fiber.Ctx, access, refresh string, expiresIn int, user *dto.User, secure bool) {
	persistTokens(c, access, refresh, expiresIn, secure)
	persistUser(c, user, expiresIn, secure)
}

func ClearSession(c fiber.Ctx, secure bool) {
	clearSession(c, secure)
}

func setCookie(c fiber.Ctx, name, value string, maxAge int, secure bool) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Lax",
	})
}

func clearSession(c fiber.Ctx, secure bool) {
	for _, name := range []string{CookieAccess, CookieRefresh, CookieUser} {
		c.Cookie(&fiber.Cookie{
			Name:     name,
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			HTTPOnly: true,
			Secure:   secure,
			SameSite: "Lax",
		})
	}
}

func decodeJWT(token string) *jwtClaims {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	payload := parts[1]
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}
	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		raw, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil
		}
	}
	var claims jwtClaims
	if json.Unmarshal(raw, &claims) != nil {
		return nil
	}
	return &claims
}
