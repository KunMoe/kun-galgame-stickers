package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kun-galgame-sticker-api/internal/infrastructure/database"
	"kun-galgame-sticker-api/internal/platform/sticker/repository"
	"kun-galgame-sticker-api/pkg/config"
	"kun-galgame-sticker-api/pkg/imageclient"
	"kun-galgame-sticker-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")

	root := flag.String("root", defaultRoot(), "directory with KUN_Stickers{n} originals")
	dryRun := flag.Bool("dry-run", false, "print files without uploading")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	if cfg.Image.BaseURL == "" {
		slog.Error("KUN_IMAGE_CLIENT_BASE_URL is required")
		os.Exit(1)
	}

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	repo := repository.New(db)
	cli := imageclient.New(imageclient.Config{
		BaseURL:      cfg.Image.BaseURL,
		CDNBase:      cfg.Image.CDNBase,
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		Timeout:      2 * time.Minute,
	})

	rows, err := repo.ListWithoutHash()
	if err != nil {
		slog.Error("list stickers", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	var ok, skip, fail int
	var hashes []string

	for _, row := range rows {
		src, err := findSource(*root, row.Sid, row.Pid)
		if err != nil {
			fail++
			slog.Warn("missing original", "sid", row.Sid, "pid", row.Pid, "err", err)
			continue
		}
		if *dryRun {
			ok++
			slog.Info("would upload", "sid", row.Sid, "pid", row.Pid, "file", src)
			continue
		}
		f, err := os.Open(src)
		if err != nil {
			fail++
			slog.Warn("open", "file", src, "err", err)
			continue
		}
		result, err := cli.UploadWithSub(ctx, f, filepath.Base(src), "sticker", "")
		_ = f.Close()
		if err != nil {
			fail++
			slog.Warn("upload", "sid", row.Sid, "pid", row.Pid, "err", err)
			continue
		}
		if err := repo.SetImageHash(row.Sid, row.Pid, result.Hash); err != nil {
			fail++
			slog.Warn("update hash", "sid", row.Sid, "pid", row.Pid, "err", err)
			continue
		}
		ok++
		hashes = append(hashes, result.Hash)
		slog.Info("migrated", "sid", row.Sid, "pid", row.Pid, "hash", result.Hash, "dedup", result.Deduplicated)
	}

	if !*dryRun && len(hashes) > 0 {
		if _, err := cli.ReferencePing(ctx, hashes); err != nil {
			slog.Warn("reference ping", "err", err)
		}
	}

	slog.Info("done", "ok", ok, "skip", skip, "fail", fail, "pending", len(rows), "dry_run", *dryRun)
	if fail > 0 {
		os.Exit(1)
	}
}

func defaultRoot() string {
	candidates := []string{
		"../../apps/web/public/kun-galgame-stickers",
		"apps/web/public/kun-galgame-stickers",
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return "apps/web/public/kun-galgame-stickers"
}

func findSource(root string, sid, pid int) (string, error) {
	dir := filepath.Join(root, fmt.Sprintf("KUN_Stickers%d", sid))
	matches, _ := filepath.Glob(filepath.Join(dir, fmt.Sprintf("s%d-%d.*", sid, pid)))
	if path := preferPNG(matches); path != "" {
		return path, nil
	}
	tg := filepath.Join(root, "telegram", fmt.Sprintf("KUNgal%d", sid), fmt.Sprintf("%d.png", pid))
	if st, err := os.Stat(tg); err == nil && !st.IsDir() {
		return tg, nil
	}
	return "", fmt.Errorf("no original for %d-%d", sid, pid)
}

func preferPNG(paths []string) string {
	var fallback string
	for _, p := range paths {
		ext := strings.ToLower(filepath.Ext(p))
		switch ext {
		case ".png":
			return p
		case ".jpg", ".jpeg", ".webp":
			if fallback == "" {
				fallback = p
			}
		}
	}
	return fallback
}
