package media

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	cleanupTimeout      = 50 * time.Second
	cleanupPhaseTimeout = 10 * time.Second
	cleanupPageSize     = 128
)

var managedMediaPrefixes = [...]string{"posts/", "profiles/"}

// Cleaner retains an orphan-scan position for one worker. A new cleaner starts at the first prefix.
type Cleaner struct {
	q        *store.Queries
	objects  Objects
	pageSize int
	prefix   int
	after    string
}

func NewCleaner(q *store.Queries, objects Objects, pageSize int) *Cleaner {
	return &Cleaner{q: q, objects: objects, pageSize: pageSize}
}

// RunCycle retries tombstones and multipart cleanup, then scans from the current orphan checkpoint.
func (c *Cleaner) RunCycle(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	before := time.Now().Add(-time.Hour)
	phaseCtx, cancel := context.WithTimeout(ctx, cleanupPhaseTimeout)
	tombstoneErr := c.cleanupTombstones(phaseCtx)
	cancel()

	phaseCtx, cancel = context.WithTimeout(ctx, cleanupPhaseTimeout)
	multipartErr := c.objects.CleanupMultipart(phaseCtx, before)
	cancel()

	return errors.Join(tombstoneErr, multipartErr, c.cleanupOrphans(ctx, before))
}

// Keep a tombstone until both objects are removed so failures and restarts retry deletion.
func (c *Cleaner) cleanupTombstones(ctx context.Context) error {
	rows, err := c.q.ClaimStaleMedia(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := c.objects.Delete(ctx, row.ObjectKey); err != nil {
			return err
		}
		if row.ThumbnailKey != "" {
			if err := c.objects.Delete(ctx, row.ThumbnailKey); err != nil {
				return err
			}
		}
		if err := c.q.DeleteStaleMedia(ctx, row.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cleaner) cleanupOrphans(ctx context.Context, before time.Time) error {
	var deletionErr error
	for c.prefix < len(managedMediaPrefixes) {
		if err := ctx.Err(); err != nil {
			return errors.Join(deletionErr, err)
		}
		prefix := managedMediaPrefixes[c.prefix]
		page, err := c.objects.ListPage(ctx, prefix, c.after, c.pageSize)
		if err != nil {
			return errors.Join(deletionErr, err)
		}
		referenced, err := c.referencedMediaKeys(ctx, prefix, page, before)
		if err != nil {
			return errors.Join(deletionErr, err)
		}
		for _, obj := range page {
			if err := ctx.Err(); err != nil {
				return errors.Join(deletionErr, err)
			}
			if isMature(obj, before) && !referenced[obj.Key] {
				if err := c.objects.Delete(ctx, obj.Key); err != nil {
					if ctx.Err() != nil {
						return errors.Join(deletionErr, err, ctx.Err())
					}
					// A failed deletion leaves the object in storage for the next pass.
					deletionErr = err
				}
			}
			c.after = obj.Key
		}
		if len(page) < c.pageSize {
			c.prefix++
			c.after = ""
		}
	}
	c.prefix = 0
	return deletionErr
}

func isMature(obj Object, before time.Time) bool {
	return !obj.Modified.IsZero() && obj.Modified.Before(before)
}

func (c *Cleaner) referencedMediaKeys(ctx context.Context, prefix string, page []Object, before time.Time) (map[string]bool, error) {
	if len(page) > c.pageSize {
		return nil, errors.New("object listing exceeded page size")
	}
	keys := make([]string, 0, len(page))
	previous := c.after
	for _, obj := range page {
		if !strings.HasPrefix(obj.Key, prefix) || obj.Key <= previous {
			return nil, errors.New("invalid object listing order or prefix")
		}
		previous = obj.Key
		if isMature(obj, before) {
			keys = append(keys, obj.Key)
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	referenced, err := c.q.ReferencedMediaKeys(ctx, keys)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(referenced))
	for _, key := range referenced {
		result[key] = true
	}
	return result, nil
}

func RunCleanup(ctx context.Context, pool *pgxpool.Pool, objects Objects, log *slog.Logger) {
	if objects == nil {
		return
	}
	cleaner := NewCleaner(store.New(pool), objects, cleanupPageSize)
	for ctx.Err() == nil {
		cleanupCtx, cancel := context.WithTimeout(ctx, cleanupTimeout)
		err := cleaner.RunCycle(cleanupCtx)
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
