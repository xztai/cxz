package mailqueue

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestCipherAuthenticationAndRotation(t *testing.T) {
	first := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	second := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	old, err := NewCipher(first)
	if err != nil {
		t.Fatal(err)
	}
	body := old.Seal("private reset link", "recipient+digest")
	rotated, err := NewCipher(second + "," + first)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := rotated.Open(body, "recipient+digest")
	if err != nil || plain != "private reset link" {
		t.Fatal("rotation lost old mail", err)
	}
	if _, err := rotated.Open(body, "other recipient"); err == nil {
		t.Fatal("metadata swap accepted")
	}
	if _, err := rotated.Open(body[:len(body)-4]+"AAAA", "recipient+digest"); err == nil {
		t.Fatal("tampering accepted")
	}
	current, _ := NewCipher(second)
	if _, err := current.Open(body, "recipient+digest"); err == nil {
		t.Fatal("removed key accepted")
	}
	if strings.Contains(body, "private reset link") || body == old.Seal("private reset link", "recipient+digest") {
		t.Fatal("plaintext or reused nonce")
	}
	for _, bad := range []string{"", "short", "AAAA"} {
		if _, err := NewCipher(bad); err == nil {
			t.Fatal("bad key accepted")
		}
	}
}
