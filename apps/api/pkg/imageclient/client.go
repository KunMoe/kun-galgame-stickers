package imageclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	BaseURL      string
	CDNBase      string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
	Timeout      time.Duration
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	cfg.CDNBase = strings.TrimRight(cfg.CDNBase, "/")

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	return &Client{cfg: cfg, http: httpClient}
}

func MainURL(cdnBase, hash, ext string) string {
	if len(hash) < 4 {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s/%s.%s",
		strings.TrimRight(cdnBase, "/"), hash[:2], hash[2:4], hash, ext)
}

func VariantURL(cdnBase, hash, variant, ext string) string {
	if len(hash) < 4 {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s/%s_%s.%s",
		strings.TrimRight(cdnBase, "/"), hash[:2], hash[2:4], hash, variant, ext)
}

func (c *Client) MainURL(hash string) string { return MainURL(c.cfg.CDNBase, hash, "webp") }

func (c *Client) VariantURL(hash, variant string) string {
	return VariantURL(c.cfg.CDNBase, hash, variant, "webp")
}

type UploadResult struct {
	Hash         string            `json:"hash"`
	URL          string            `json:"url"`
	VariantURLs  map[string]string `json:"variant_urls"`
	Width        int               `json:"width"`
	Height       int               `json:"height"`
	Thumbhash    string            `json:"thumbhash,omitempty"`
	SizeBytes    int64             `json:"size_bytes"`
	Deduplicated bool              `json:"deduplicated"`
}

type Error struct {
	StatusCode int
	Code       int             `json:"code"`
	Message    string          `json:"message"`
	Details    json.RawMessage `json:"details,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("image service error: status=%d code=%d: %s", e.StatusCode, e.Code, e.Message)
}

var (
	ErrQuotaExceeded      = errors.New("imageclient: quota exceeded")
	ErrModerationRejected = errors.New("imageclient: rejected by moderation")
	// ErrMIMEDenied is the preset refusing the file's format (code 80009). Like
	// moderation it is a PERMANENT verdict on this file, not a transient
	// failure: retrying re-uploads the same bytes to the same preset for the
	// same answer. A backfill that treats it as retryable burns its whole
	// backoff ladder per file (~90s) to learn nothing.
	ErrMIMEDenied   = errors.New("imageclient: preset does not accept this format")
	ErrUnauthorized = errors.New("imageclient: unauthorized")
)

func classifyError(e *Error) error {
	if e == nil {
		return nil
	}
	switch e.Code {
	case 80008:
		return fmt.Errorf("%w: %s", ErrQuotaExceeded, e.Message)
	case 80001, 80002, 80003, 80004, 80005:
		return fmt.Errorf("%w: %s", ErrUnauthorized, e.Message)
	case 60002:
		return fmt.Errorf("%w: %s", ErrModerationRejected, e.Message)
	case 80009:
		return fmt.Errorf("%w: %s", ErrMIMEDenied, e.Message)
	default:
		return e
	}
}

func (c *Client) Upload(ctx context.Context, r io.Reader, filename, presetName string) (*UploadResult, error) {
	return c.UploadWithSub(ctx, r, filename, presetName, "")
}

func (c *Client) UploadWithSub(ctx context.Context, r io.Reader, filename, presetName, uploaderSub string) (*UploadResult, error) {
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)

	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("build multipart: %w", err)
	}
	if _, err := io.Copy(part, r); err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}
	if err := mw.WriteField("preset", presetName); err != nil {
		return nil, err
	}
	if uploaderSub != "" {
		if err := mw.WriteField("uploader_sub", uploaderSub); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/image/upload", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload http: %w", err)
	}
	defer resp.Body.Close()

	return parseUploadResponse(resp)
}

func parseUploadResponse(resp *http.Response) (*UploadResult, error) {
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var e Error
		_ = json.Unmarshal(raw, &e)
		e.StatusCode = resp.StatusCode
		return nil, classifyError(&e)
	}
	var env struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    UploadResult `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("parse upload response: %w", err)
	}
	return &env.Data, nil
}

type ImageMeta struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Thumbhash string `json:"thumbhash,omitempty"`
	Sexual    *int16 `json:"sexual,omitempty"`
}

func (c *Client) MetaBatch(ctx context.Context, hashes []string) (map[string]ImageMeta, error) {
	if len(hashes) == 0 {
		return map[string]ImageMeta{}, nil
	}
	if len(hashes) > 1000 {
		return nil, fmt.Errorf("imageclient: batch size %d exceeds limit 1000", len(hashes))
	}

	body, _ := json.Marshal(struct {
		Hashes []string `json:"hashes"`
	}{Hashes: hashes})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/image/meta-batch", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("meta-batch http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var e Error
		_ = json.Unmarshal(raw, &e)
		e.StatusCode = resp.StatusCode
		return nil, classifyError(&e)
	}
	var env struct {
		Data struct {
			Metas map[string]ImageMeta `json:"metas"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("parse meta-batch response: %w", err)
	}
	if env.Data.Metas == nil {
		return map[string]ImageMeta{}, nil
	}
	return env.Data.Metas, nil
}

type ReferencePingResult struct {
	Updated  int64    `json:"updated"`
	NotFound []string `json:"not_found"`
}

func (c *Client) ReferencePing(ctx context.Context, hashes []string) (*ReferencePingResult, error) {
	if len(hashes) == 0 {
		return &ReferencePingResult{}, nil
	}
	if len(hashes) > 1000 {
		return nil, fmt.Errorf("imageclient: batch size %d exceeds limit 1000", len(hashes))
	}

	body, _ := json.Marshal(struct {
		Hashes []string `json:"hashes"`
	}{Hashes: hashes})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/image/reference-ping", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ping http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var e Error
		_ = json.Unmarshal(raw, &e)
		e.StatusCode = resp.StatusCode
		return nil, classifyError(&e)
	}
	var env struct {
		Data ReferencePingResult `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("parse ping response: %w", err)
	}
	return &env.Data, nil
}

func (c *Client) Delete(ctx context.Context, hash string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.cfg.BaseURL+"/image/"+hash, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("delete http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		var e Error
		_ = json.Unmarshal(raw, &e)
		e.StatusCode = resp.StatusCode
		return classifyError(&e)
	}
	return nil
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("health: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) basicAuthHeader() string {
	creds := c.cfg.ClientID + ":" + c.cfg.ClientSecret
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}
