package httpapi

import (
	"strings"
	"sync"
	"testing"
)

func TestIntegrationVerificationExpiry(t *testing.T) {
	h := setup(t)
	h.expect("POST", "/auth/register", `{"email":"person@example.com","nickname":"author","password":"original password 123"}`, 202)
	token := h.mailToken("verify")
	if status, _, _ := h.call("POST", "/auth/verify-email/preview", `{"token":"`+token+`"}`, false); status != 403 {
		t.Fatal("preview bypassed CSRF validation")
	}
	h.expect("POST", "/auth/verify-email/preview", `{"token":"bad"}`, 400)
	if _, err := h.pool.Exec(t.Context(), "UPDATE action_tokens SET expires_at=now()-interval '1 second'"); err != nil {
		t.Fatal(err)
	}
	h.expect("POST", "/auth/verify-email/preview", `{"token":"`+token+`"}`, 400)
	h.expect("POST", "/auth/verify-email", `{"token":"`+token+`"}`, 400)
	if !strings.Contains(h.expect("GET", "/me", "", 200), `"user":null`) {
		t.Fatal("expired token created a session")
	}
}

func TestIntegrationConcurrentVerification(t *testing.T) {
	h := setup(t)
	h.expect("POST", "/auth/register", `{"email":"person@example.com","nickname":"author","password":"original password 123"}`, 202)
	token := h.mailToken("verify")
	results := make(chan int, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			status, _, _ := h.call("POST", "/auth/verify-email", `{"token":"`+token+`"}`, true)
			results <- status
		})
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for status := range results {
		counts[status]++
	}
	if counts[200] != 1 || counts[400] != 1 {
		t.Fatalf("verification results: %v", counts)
	}
	var sessions int
	if err := h.pool.QueryRow(t.Context(), "SELECT count(*) FROM sessions").Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if sessions != 1 {
		t.Fatal("verification created duplicate sessions")
	}
}
