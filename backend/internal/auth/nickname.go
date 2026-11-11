package auth

import "strings"

// NormalizeNickname accepts only URL-safe ASCII characters, without Unicode folding.
func NormalizeNickname(value string) (string, bool) {
	if len(value) < 3 || len(value) > 30 {
		return "", false
	}
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			continue
		}
		return "", false
	}
	return strings.ToLower(value), true
}
