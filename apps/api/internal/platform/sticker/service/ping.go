package service

import (
	"context"
	"log/slog"
	"time"
)

func (s *StickerService) StartRefPing(ctx context.Context) {
	if s.images == nil {
		return
	}
	go func() {
		timer := time.NewTimer(time.Minute)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.PingHashes(ctx)
		}
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.PingHashes(ctx)
				slog.Info("image reference ping finished")
			}
		}
	}()
}
