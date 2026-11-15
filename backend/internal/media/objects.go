package media

import (
	"context"
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
	Walk(context.Context, func(Object) error) error
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

func (s *S3) Walk(ctx context.Context, fn func(Object) error) error {
	for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Recursive: true}) {
		if obj.Err != nil {
			return obj.Err
		}
		if err := fn(Object{Key: obj.Key, Modified: obj.LastModified}); err != nil {
			return err
		}
	}
	return nil
}

func (s *S3) CleanupMultipart(ctx context.Context, before time.Time) error {
	for obj := range s.client.ListIncompleteUploads(ctx, s.bucket, "", true) {
		if obj.Err != nil {
			return obj.Err
		}
		if obj.Initiated.Before(before) {
			if err := s.client.RemoveIncompleteUpload(ctx, s.bucket, obj.Key); err != nil {
				return fmt.Errorf("remove incomplete upload: %w", err)
			}
		}
	}
	return nil
}
