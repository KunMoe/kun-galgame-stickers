package service

import (
	"context"

	"kun-galgame-sticker-api/internal/platform/identity/dto"
	"kun-galgame-sticker-api/internal/platform/identity/oauth"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/perm"
	"kun-galgame-sticker-api/pkg/userclient"
)

type AuthService struct {
	client *oauth.Client
	users  *userclient.Client
}

func New(client *oauth.Client, users *userclient.Client) *AuthService {
	return &AuthService{client: client, users: users}
}

func (s *AuthService) Callback(code, codeVerifier string) (*oauth.Tokens, *dto.User, *errors.AppError) {
	if code == "" || codeVerifier == "" {
		return nil, nil, errors.ErrBadRequest("missing OAuth callback parameters")
	}
	tokens, err := s.client.ExchangeCode(code, codeVerifier)
	if err != nil {
		return nil, nil, errors.ErrBadRequest("OAuth exchange failed: " + err.Error())
	}
	user, err := s.client.FetchUser(tokens.AccessToken)
	if err != nil {
		return nil, nil, errors.ErrBadRequest("OAuth userinfo failed: " + err.Error())
	}
	return tokens, toDTO(user), nil
}

func (s *AuthService) Refresh(refreshToken string) (*oauth.Tokens, *dto.User, *errors.AppError) {
	tokens, err := s.client.Refresh(refreshToken)
	if err != nil {
		return nil, nil, errors.ErrUnauthorized("session expired")
	}
	user, err := s.client.FetchUser(tokens.AccessToken)
	if err != nil {
		return nil, nil, errors.ErrUnauthorized("session expired")
	}
	return tokens, toDTO(user), nil
}

func (s *AuthService) FetchUser(accessToken string) (*dto.User, *errors.AppError) {
	user, err := s.client.FetchUser(accessToken)
	if err != nil {
		return nil, errors.ErrUnauthorized("session expired")
	}
	return toDTO(user), nil
}

func (s *AuthService) Me(ctx context.Context, user *dto.User) dto.Me {
	return dto.Me{User: *user, Cosmetics: s.users.User(ctx, user.ID).Cosmetics}
}

func (s *AuthService) Revoke(refreshToken string) {
	if refreshToken != "" {
		s.client.Revoke(refreshToken)
	}
}

func toDTO(u *oauth.User) *dto.User {
	roles := perm.Union(u.Roles, u.SiteRoles)
	if roles == nil {
		roles = []string{}
	}
	return &dto.User{
		Sub:     u.Sub,
		ID:      u.ID,
		Name:    u.Name,
		Picture: u.Picture,
		Roles:   roles,
	}
}
