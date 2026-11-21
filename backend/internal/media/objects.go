package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
)

const MaxFileBytes int64 = 100 << 20
const TransferTimeout = 10 * time.Minute

type Object struct {
	Key      string
	Modified time.Time
}

// Objects never exposes presigned or public URLs. Open supports seeking so the
// API can serve HTTP byte ranges without downloading entire originals.
type Objects interface {
	Upload(context.Context, string, io.Reader, int64, string) error
	Open(context.Context, string) (io.ReadSeekCloser, error)
	Delete(context.Context, string) error
	ListPage(ctx context.Context, prefix, after string, limit int) ([]Object, error)
	CleanupMultipart(context.Context, time.Time) error
}

func (s *S3) Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *S3) Open(ctx context.Context, key string) (io.ReadSeekCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		return nil, err
	}
	return obj, nil
}

// ListPage returns up to limit objects in key order, starting strictly after after.
func (s *S3) ListPage(ctx context.Context, prefix, after string, limit int) ([]Object, error) {
	if limit <= 0 {
		return nil, errors.New("list page limit must be positive")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	page := make([]Object, 0, limit)
	opts := minio.ListObjectsOptions{Recursive: true, Prefix: prefix, StartAfter: after, MaxKeys: limit}
	for obj := range s.client.ListObjectsIter(ctx, s.bucket, opts) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		page = append(page, Object{Key: obj.Key, Modified: obj.LastModified})
		if len(page) == limit {
			break
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return page, nil
}

func (s *S3) CleanupMultipart(ctx context.Context, before time.Time) error {
	ctx, cancel := context.WithCancel(ctx)
	uploads := s.client.ListIncompleteUploads(ctx, s.bucket, "", true)
	defer func() {
		cancel()
		// The client can send a final error after cancellation.
		for range uploads {
		}
	}()
	for obj := range uploads {
		if obj.Err != nil {
			return obj.Err
		}
		if obj.Initiated.Before(before) {
			if err := s.client.RemoveIncompleteUpload(ctx, s.bucket, obj.Key); err != nil {
				return fmt.Errorf("remove incomplete upload: %w", err)
			}
		}
	}
	return ctx.Err()
}
