package media

import (
	"bytes"
	"os"
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
