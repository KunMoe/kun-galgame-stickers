package oauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"kun-galgame-sticker-api/pkg/config"
)

const httpTimeout = 10 * time.Second

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type User struct {
	Sub     string   `json:"sub"`
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Picture string   `json:"picture"`
	Roles   []string `json:"roles"`
}

type Error struct {
	Message    string
	HTTPStatus int
}

func (e *Error) Error() string {
	return fmt.Sprintf("oauth: http=%d msg=%q", e.HTTPStatus, e.Message)
}

type Client struct {
	cfg        config.OAuthConfig
	httpClient *http.Client
}

func NewClient(cfg config.OAuthConfig) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: httpTimeout},
	}
}

func (c *Client) ExchangeCode(code, codeVerifier string) (*Tokens, error) {
	return c.postProtocol("/oauth/token", map[string]any{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  c.cfg.RedirectURI,
		"client_id":     c.cfg.ClientID,
		"client_secret": c.cfg.ClientSecret,
		"code_verifier": codeVerifier,
	})
}

func (c *Client) Refresh(refreshToken string) (*Tokens, error) {
	return c.postProtocol("/oauth/token", map[string]any{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     c.cfg.ClientID,
		"client_secret": c.cfg.ClientSecret,
	})
}

func (c *Client) Revoke(token string) {
	_, _ = c.postProtocol("/oauth/revoke", map[string]any{"token": token})
}

func (c *Client) FetchUser(accessToken string) (*User, error) {
	req, err := http.NewRequest(http.MethodGet, c.cfg.ServerURL+"/oauth/userinfo", nil)
	if err != nil {
		return nil, &Error{Message: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{Message: err.Error()}
	}
	defer resp.Body.Close()

	body, err := decodeProtocol(resp)
	if err != nil {
		return nil, err
	}
	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, &Error{HTTPStatus: resp.StatusCode, Message: "decode userinfo: " + err.Error()}
	}
	return &user, nil
}

func (c *Client) postProtocol(path string, payload map[string]any) (*Tokens, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, &Error{Message: err.Error()}
	}
	req, err := http.NewRequest(http.MethodPost, c.cfg.ServerURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, &Error{Message: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{Message: err.Error()}
	}
	defer resp.Body.Close()

	body, err := decodeProtocol(resp)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return &Tokens{}, nil
	}
	var tokens Tokens
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, &Error{HTTPStatus: resp.StatusCode, Message: "decode token: " + err.Error()}
	}
	return &tokens, nil
}

func decodeProtocol(resp *http.Response) (json.RawMessage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &Error{HTTPStatus: resp.StatusCode, Message: "read body: " + err.Error()}
	}
	if len(body) == 0 {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return json.RawMessage{}, nil
		}
		return nil, &Error{HTTPStatus: resp.StatusCode, Message: "empty body"}
	}
	var wire map[string]any
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, &Error{HTTPStatus: resp.StatusCode, Message: "non-json body"}
	}
	if errStr, ok := wire["error"].(string); ok && errStr != "" {
		detail, _ := wire["error_description"].(string)
		msg := errStr
		if detail != "" {
			msg = errStr + ": " + detail
		}
		return nil, &Error{HTTPStatus: resp.StatusCode, Message: msg}
	}
	if resp.StatusCode >= 400 {
		return nil, &Error{HTTPStatus: resp.StatusCode, Message: fmt.Sprintf("http %d", resp.StatusCode)}
	}
	return json.RawMessage(body), nil
}
