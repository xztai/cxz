package media

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The API's local temp directory is separate from the worker container. Clean
// files left by API crashes here; the one-hour grace exceeds transfer deadlines.
func cleanupTempUploads(ctx context.Context, dir string, before time.Time) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !strings.HasPrefix(entry.Name(), "etheyl-upload-") || !entry.Type().IsRegular() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if info.ModTime().Before(before) {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func RunTempCleanup(ctx context.Context, log *slog.Logger) {
	for {
		if err := cleanupTempUploads(ctx, os.TempDir(), time.Now().Add(-time.Hour)); err != nil && ctx.Err() == nil {
			log.Error("temporary upload cleanup failed")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}
	}
}
