package media

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage interface {
	Put(context.Context, string, Image) error
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}
type S3 struct {
	client *minio.Client
	bucket string
}

func New(endpoint, access, secret, bucket string, secure bool) (*S3, error) {
	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(access, secret, ""), Secure: secure, Region: "us-east-1"})
	if err != nil {
		return nil, fmt.Errorf("S3 client: %w", err)
	}
	return &S3{client: client, bucket: bucket}, nil
}

// Initialize is run by the deployment job before API startup. An empty policy
// explicitly removes anonymous access on both fresh and existing buckets.
func (s *S3) Initialize(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: "us-east-1"}); err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
	}
	if err := s.client.SetBucketPolicy(ctx, s.bucket, ""); err != nil {
		return fmt.Errorf("private bucket: %w", err)
	}
	return nil
}
func (s *S3) Put(ctx context.Context, key string, img Image) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(img.Data), int64(len(img.Data)), minio.PutObjectOptions{ContentType: img.ContentType, DisableMultipart: true})
	if err != nil {
		return fmt.Errorf("store image: %w", err)
	}
	return nil
}
func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get image: %w", err)
	}
	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		return nil, fmt.Errorf("stat image: %w", err)
	}
	return obj, nil
}
func (s *S3) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove image: %w", err)
	}
	return nil
}
