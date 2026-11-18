package media

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

func TestS3IncompleteUploads(t *testing.T) {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT required")
	}
	storage, err := New(endpoint, "etheyl_test", "etheyl_test_password", "etheyl-multipart-test", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	core := minio.Core{Client: storage.client}
	const key = "posts/interrupted"
	id, err := core.NewMultipartUpload(t.Context(), storage.bucket, key, minio.PutObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.PutObjectPart(t.Context(), storage.bucket, key, id, 1, bytes.NewReader([]byte("unfinished")), 10, minio.PutObjectPartOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := storage.CleanupMultipart(t.Context(), time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	found := 0
	for object := range storage.client.ListIncompleteUploads(t.Context(), storage.bucket, "", true) {
		if object.Err != nil {
			t.Fatal(object.Err)
		}
		found++
	}
	if found != 1 {
		t.Fatal("recent multipart upload removed", found)
	}
	if err := storage.CleanupMultipart(t.Context(), time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	for object := range storage.client.ListIncompleteUploads(t.Context(), storage.bucket, "", true) {
		if object.Err != nil {
			t.Fatal(object.Err)
		}
		t.Fatal("stale multipart upload retained")
	}
}

func TestS3ListPage(t *testing.T) {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT required")
	}
	storage, err := New(endpoint, "etheyl_test", "etheyl_test_password", "etheyl-list-page-test", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("posts/page-%d/", time.Now().UnixNano())
	for _, key := range []string{prefix + "d", prefix + "a", prefix + "c", prefix + "b", strings.TrimSuffix(prefix, "/") + "-unowned", "profiles/other"} {
		if err := storage.Upload(t.Context(), key, bytes.NewReader([]byte("x")), 1, "text/plain"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := storage.Delete(ctx, key); err != nil {
				t.Error(err)
			}
		})
	}
	after := ""
	for _, want := range [][]string{{prefix + "a", prefix + "b"}, {prefix + "c", prefix + "d"}, {}} {
		page, err := storage.ListPage(t.Context(), prefix, after, 2)
		if err != nil {
			t.Fatal(err)
		}
		keys := make([]string, 0, len(page))
		for _, obj := range page {
			if obj.Modified.IsZero() {
				t.Error("listing omitted modification time")
			}
			keys = append(keys, obj.Key)
			after = obj.Key
		}
		if !slices.Equal(keys, want) {
			t.Fatalf("listed keys %v, want %v", keys, want)
		}
	}

	// Cleanup can remove the last key in a page before resuming from it.
	if err := storage.Delete(t.Context(), prefix+"b"); err != nil {
		t.Fatal(err)
	}
	page, err := storage.ListPage(t.Context(), prefix, prefix+"b", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 || page[0].Key != prefix+"c" || page[1].Key != prefix+"d" {
		t.Fatal("resume from removed cursor did not yield the next two objects", page)
	}
}
