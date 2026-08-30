package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	OAuth    OAuthConfig
	CORS     CORSConfig
	Image    ImageConfig
}

type ImageConfig struct {
	BaseURL string
	CDNBase string
}

type ServerConfig struct {
	Port   string
	Mode   string
	Secure bool
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
}

type OAuthConfig struct {
	ServerURL    string
	WebURL       string
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type CORSConfig struct {
	AllowOrigins string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:   env("SERVER_PORT", "9421"),
			Mode:   env("SERVER_MODE", "dev"),
			Secure: env("SERVER_MODE", "dev") == "prod",
		},
		Database: DatabaseConfig{
			URL:             os.Getenv("KUN_DATABASE_URL"),
			MaxOpenConns:    envInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    envInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: envInt("DB_CONN_MAX_LIFETIME", 300),
		},
		OAuth: OAuthConfig{
			ServerURL:    strings.TrimRight(os.Getenv("KUN_OAUTH_SERVER_URL"), "/"),
			WebURL:       strings.TrimRight(env("KUN_OAUTH_WEB_URL", "https://oauth.kungal.com"), "/"),
			ClientID:     os.Getenv("KUN_OAUTH_CLIENT_ID"),
			ClientSecret: os.Getenv("KUN_OAUTH_CLIENT_SECRET"),
			RedirectURI:  os.Getenv("KUN_OAUTH_REDIRECT_URI"),
		},
		CORS: CORSConfig{
			AllowOrigins: env("CORS_ALLOW_ORIGINS", "http://127.0.0.1:5173"),
		},
		Image: ImageConfig{
			BaseURL: strings.TrimRight(os.Getenv("KUN_IMAGE_CLIENT_BASE_URL"), "/"),
			CDNBase: strings.TrimRight(env("KUN_IMAGE_CDN_BASE", "https://image.kungal.iloveren.link"), "/"),
		},
	}
	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("KUN_DATABASE_URL is required")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
