package billing

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCalendarMonth(t *testing.T) {
	for _, tc := range []struct {
		start, want string
		anchor      int
	}{
		{"2026-01-31T12:30:00Z", "2026-02-28T12:30:00Z", 31},
		{"2026-02-28T12:30:00Z", "2026-03-31T12:30:00Z", 31},
		{"2028-01-31T12:30:00Z", "2028-02-29T12:30:00Z", 31},
		{"2026-12-30T12:30:00Z", "2027-01-30T12:30:00Z", 30},
	} {
		start, _ := time.Parse(time.RFC3339, tc.start)
		if got := NextMonth(start, tc.anchor).Format(time.RFC3339); got != tc.want {
			t.Errorf("%s -> %s, want %s", tc.start, got, tc.want)
		}
	}
}
func TestClientCredentialsIdempotenceAndErrors(t *testing.T) {
	status := 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shop, key, ok := r.BasicAuth()
		if !ok || shop != "shop" || key != "secret" {
			t.Error("missing shop authentication")
		}
		if r.Method == "POST" && r.Header.Get("Idempotence-Key") != "stable-key" {
			t.Error("missing stable key")
		}
		w.WriteHeader(status)
		if status == 200 {
			_ = json.MarshalWrite(w, RemotePayment{ID: "remote-id", Status: "pending"})
		} else {
			_, _ = w.Write([]byte(`{"description":"secret provider body"}`))
		}
	}))
	defer server.Close()
	c := NewClient("shop", "secret")
	c.base = server.URL
	if _, err := c.Create(t.Context(), "stable-key", "{}"); err != nil {
		t.Fatal(err)
	}
	for _, code := range []int{400, 401, 429, 500} {
		status = code
		_, err := c.Get(t.Context(), "remote-id")
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatal("unsafe or missing error")
		}
	}
	if _, err := c.Get(t.Context(), "../me"); err == nil {
		t.Fatal("accepted arbitrary provider path")
	}
}
func TestConfirmationAllowlist(t *testing.T) {
	for _, raw := range []string{"http://yoomoney.ru/pay", "https://yoomoney.ru.evil.test/pay", "https://evil.test", "https://user@yoomoney.ru/pay", "https://yoomoney.ru:444/pay"} {
		if safeConfirmation(raw) != "" {
			t.Error("unsafe redirect", raw)
		}
	}
	if safeConfirmation("https://yoomoney.ru/checkout?order=123") == "" {
		t.Fatal("valid provider URL rejected")
	}
	if money(100) != "1.00" || money(19950) != "199.50" {
		t.Fatal("inexact kopecks conversion")
	}
}
