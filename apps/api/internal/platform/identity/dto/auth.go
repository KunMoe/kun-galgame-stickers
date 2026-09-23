package dto

import "kun-galgame-sticker-api/pkg/userclient"

type CallbackRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
}

type User struct {
	Sub     string   `json:"sub"`
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Picture string   `json:"picture"`
	Roles   []string `json:"roles"`
}

type Me struct {
	User
	Cosmetics *userclient.Cosmetics `json:"cosmetics,omitempty"`
}
