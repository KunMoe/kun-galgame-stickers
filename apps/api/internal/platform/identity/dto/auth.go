package dto

type CallbackRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
}

type User struct {
	Sub     string   `json:"sub"`
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Picture string   `json:"picture"`
	Roles   []string `json:"roles"`
}
