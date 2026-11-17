package auth

import (
	"strings"
	"testing"
)

func TestPasswordHash(t *testing.T) {
	password := "Очень длинный пароль 2026"
	a, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	b, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("password salts must differ")
	}
	if !CheckPassword(a, password) || CheckPassword(a, "wrong password") || CheckPassword(a, password+" ") {
		t.Fatal("password verification failed")
	}
	for _, value := range []string{"", "invalid", "$argon2id$v=19$m=999999999,t=3,p=2$a$b", "$argon2id$v=19$m=65536,t=3,p=2$AA$AA"} {
		if CheckPassword(value, password) {
			t.Fatalf("accepted malformed hash %q", value)
		}
	}
	for _, value := range []string{"short", strings.Repeat("a", 257), string([]byte{0xff})} {
		if ValidPassword(value) {
			t.Fatal("accepted invalid password")
		}
	}
}
func TestEmailAndTokens(t *testing.T) {
	if value, ok := NormalizeEmail(" Person@Example.COM "); !ok || value != "person@example.com" {
		t.Fatal("email normalization")
	}
	for _, value := range []string{"Name <person@example.com>", "a\nb@example.com", "bad"} {
		if _, ok := NormalizeEmail(value); ok {
			t.Fatal("accepted invalid email")
		}
	}
	a, err := Token()
	if err != nil {
		t.Fatal(err)
	}
	b, err := Token()
	if err != nil {
		t.Fatal(err)
	}
	if !ValidToken(a) || a == b || len(Digest(a)) != 64 || Digest(a) == a || ValidToken("bad") {
		t.Fatal("invalid random token")
	}
}
