package media

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Cleanup keeps a tombstone until both objects are removed, so failures and
// process restarts retry deletion. Ready attached files are never claimed.
func Cleanup(ctx context.Context, q *store.Queries, objects Objects) error {
	rows, err := q.ClaimStaleMedia(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := objects.Delete(ctx, row.ObjectKey); err != nil {
			return err
		}
		if row.ThumbnailKey != "" {
			if err := objects.Delete(ctx, row.ThumbnailKey); err != nil {
				return err
			}
		}
		if err := q.DeleteStaleMedia(ctx, row.ID); err != nil {
			return err
		}
	}
	before := time.Now().Add(-time.Hour)
	if err := objects.CleanupMultipart(ctx, before); err != nil {
		return err
	}
	return objects.Walk(ctx, func(obj Object) error {
		if !obj.Modified.Before(before) || (!strings.HasPrefix(obj.Key, "posts/") && !strings.HasPrefix(obj.Key, "profiles/")) {
			return nil
		}
		referenced, err := q.MediaObjectReferenced(ctx, obj.Key)
		if err != nil {
			return err
		}
		if !referenced {
			return objects.Delete(ctx, obj.Key)
		}
		return nil
	})
}

func RunCleanup(ctx context.Context, pool *pgxpool.Pool, objects Objects, log *slog.Logger) {
	if objects == nil {
		return
	}
	for {
		cleanupCtx, cancel := context.WithTimeout(ctx, 50*time.Second)
		err := Cleanup(cleanupCtx, store.New(pool), objects)
		cancel()
		if err != nil && ctx.Err() == nil {
			log.Error("media cleanup failed")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}
	}
}
