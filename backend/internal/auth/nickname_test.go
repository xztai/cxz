package auth

import (
	"strings"
	"testing"
)

func TestNickname(t *testing.T) {
	for _, input := range []string{"Maker-123", "user_name", "ABC", strings.Repeat("Z", 30)} {
		value, ok := NormalizeNickname(input)
		if !ok || value != strings.ToLower(input) {
			t.Fatalf("rejected nickname %q", input)
		}
	}
	for _, input := range []string{"", "ab", strings.Repeat("a", 31), "user.name", "user/name", "user?name", "user#name", "user+name", "user%20name", "пользователь", "Kate", " user", "user ", "user\nname", "user@name"} {
		if _, ok := NormalizeNickname(input); ok {
			t.Fatalf("accepted nickname %q", input)
		}
	}
}
