package media

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTempCleanupKeepsRecentAndUnrelatedFiles(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-2 * time.Hour)
	for _, name := range []string{"etheyl-upload-old", "etheyl-upload-recent", "unrelated"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("temporary"), 0600); err != nil {
			t.Fatal(err)
		}
		if name != "etheyl-upload-recent" {
			if err := os.Chtimes(path, old, old); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.Symlink(filepath.Join(dir, "unrelated"), filepath.Join(dir, "etheyl-upload-link")); err != nil {
		t.Fatal(err)
	}
	if err := cleanupTempUploads(t.Context(), dir, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "etheyl-upload-old")); !os.IsNotExist(err) {
		t.Fatal("abandoned temp upload retained", err)
	}
	for _, name := range []string{"etheyl-upload-recent", "unrelated", "etheyl-upload-link"} {
		if _, err := os.Lstat(filepath.Join(dir, name)); err != nil {
			t.Fatal("live/unrelated file removed", name, err)
		}
	}
}
