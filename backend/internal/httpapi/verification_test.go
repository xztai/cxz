package httpapi

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

const changeEmailBody = `{"email":"person@example.com","newEmail":" corrected@Example.COM ","password":"original password 123"}`

func TestIntegrationChangeUnverifiedEmail(t *testing.T) {
	h := setup(t)
	h.expect("POST", "/auth/register", `{"email":"person@example.com","nickname":"author","password":"original password 123"}`, 202)
	oldToken := h.mailToken("verify")
	before, err := h.server.q.UserByEmail(t.Context(), "person@example.com")
	if err != nil {
		t.Fatal(err)
	}
	h.expect("POST", "/auth/login", `{"email":"person@example.com","password":"wrong password"}`, 401)
	if body := h.expect("POST", "/auth/login", loginBody, 403); !strings.Contains(body, `"email_unverified"`) {
		t.Fatal("correct credentials did not report unverified email")
	}
	if status, _, _ := h.call("POST", "/auth/change-unverified-email", changeEmailBody, false); status != 403 {
		t.Fatal("email change bypassed Origin and CSRF validation")
	}
	status, _, cookies := h.call("POST", "/auth/change-unverified-email", changeEmailBody, true)
	if status != 202 || len(cookies) != 0 {
		t.Fatal("email change must be accepted without a session cookie")
	}
	after, err := h.server.q.UserByEmail(t.Context(), "corrected@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if after.ID != before.ID || after.Nickname != before.Nickname || after.PasswordHash != before.PasswordHash || after.EmailVerified {
		t.Fatal("email change altered account identity, password or verification")
	}
	if _, err := h.server.q.UserByEmail(t.Context(), "person@example.com"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("old email still identifies the account", err)
	}
	if !strings.Contains(h.expect("GET", "/me", "", 200), `"user":null`) {
		t.Fatal("email change created a session")
	}
	var count int
	if err := h.pool.QueryRow(t.Context(), "SELECT count(*) FROM sessions").Scan(&count); err != nil || count != 0 {
		t.Fatal("unverified account has a session", err)
	}
	h.expect("POST", "/auth/verify-email/preview", `{"token":"`+oldToken+`"}`, 400)
	h.expect("POST", "/auth/verify-email", `{"token":"`+oldToken+`"}`, 400)
	newToken := h.mailToken("verify")
	if body := h.expect("POST", "/auth/verify-email/preview", `{"token":"`+newToken+`"}`, 200); !strings.Contains(body, "corrected@example.com") {
		t.Fatal("new verification link points to the wrong email")
	}
	if body := h.expect("POST", "/auth/verify-email", `{"token":"`+newToken+`"}`, 200); !strings.Contains(body, "corrected@example.com") {
		t.Fatal("new address was not confirmed")
	}
	h.expect("POST", "/auth/logout", "{}", 200)
	h.expect("POST", "/auth/login", loginBody, 401)
	h.expect("POST", "/auth/login", `{"email":"corrected@example.com","password":"original password 123"}`, 200)
}

func TestIntegrationChangeUnverifiedEmailRejections(t *testing.T) {
	for _, tc := range []struct {
		name, setupSQL, body, code string
		status                     int
	}{
		{"password", "", `{"email":"person@example.com","newEmail":"corrected@example.com","password":"wrong"}`, "credentials", 401},
		{"unknown", "", `{"email":"unknown@example.com","newEmail":"corrected@example.com","password":"original password 123"}`, "credentials", 401},
		{"blocked", "UPDATE users SET blocked=true", changeEmailBody, "credentials", 401},
		{"verified", "UPDATE users SET email_verified=true", changeEmailBody, "email_already_verified", 409},
		{"same", "", `{"email":"person@example.com","newEmail":"PERSON@example.com","password":"original password 123"}`, "validation", 422},
		{"malformed", "", `{"email":"person@example.com","newEmail":"not an email","password":"original password 123"}`, "validation", 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := setup(t)
			h.expect("POST", "/auth/register", `{"email":"person@example.com","nickname":"author","password":"original password 123"}`, 202)
			oldToken := h.mailToken("verify")
			if tc.setupSQL != "" {
				if _, err := h.pool.Exec(t.Context(), tc.setupSQL); err != nil {
					t.Fatal(err)
				}
			}
			body := h.expect("POST", "/auth/change-unverified-email", tc.body, tc.status)
			if !strings.Contains(body, `"code":"`+tc.code+`"`) {
				t.Fatal("unexpected rejection code")
			}
			if _, err := h.server.q.UserByEmail(t.Context(), "person@example.com"); err != nil {
				t.Fatal("rejected change altered the address", err)
			}
			if h.mailToken("verify") != oldToken {
				t.Fatal("rejected change queued a new verification link")
			}
		})
	}
}

func TestIntegrationChangeUnverifiedEmailConflictAndRateLimit(t *testing.T) {
	h := setup(t)
	h.expect("POST", "/auth/register", `{"email":"person@example.com","nickname":"author","password":"original password 123"}`, 202)
	oldToken := h.mailToken("verify")
	h.expect("POST", "/auth/register", `{"email":"corrected@example.com","nickname":"other","password":"different password 456"}`, 202)
	for range 5 {
		if body := h.expect("POST", "/auth/change-unverified-email", changeEmailBody, 409); !strings.Contains(body, `"email_exists"`) {
			t.Fatal("occupied address did not report a conflict")
		}
	}
	h.expect("POST", "/auth/change-unverified-email", changeEmailBody, 429)
	h.expect("POST", "/auth/verify-email/preview", `{"token":"`+oldToken+`"}`, 200)
	h.expect("POST", "/auth/verify-email", `{"token":"`+oldToken+`"}`, 200)
}

func TestIntegrationChangeUnverifiedEmailMailFailureRollsBack(t *testing.T) {
	h := setup(t)
	h.expect("POST", "/auth/register", `{"email":"person@example.com","nickname":"author","password":"original password 123"}`, 202)
	oldToken := h.mailToken("verify")
	if _, err := h.pool.Exec(t.Context(), `
		CREATE FUNCTION reject_changed_email_mail() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'mail failure for test'; END $$;
		CREATE TRIGGER reject_changed_email_mail BEFORE INSERT ON mail_outbox
		FOR EACH ROW WHEN (NEW.recipient = 'corrected@example.com') EXECUTE FUNCTION reject_changed_email_mail()`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := h.pool.Exec(t.Context(), "DROP TRIGGER reject_changed_email_mail ON mail_outbox; DROP FUNCTION reject_changed_email_mail()"); err != nil {
			t.Error(err)
		}
	}()
	h.expect("POST", "/auth/change-unverified-email", changeEmailBody, 500)
	if _, err := h.server.q.UserByEmail(t.Context(), "person@example.com"); err != nil {
		t.Fatal("mail failure changed the address", err)
	}
	if _, err := h.server.q.UserByEmail(t.Context(), "corrected@example.com"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("mail failure left the new address assigned", err)
	}
	h.expect("POST", "/auth/verify-email/preview", `{"token":"`+oldToken+`"}`, 200)
	h.expect("POST", "/auth/verify-email", `{"token":"`+oldToken+`"}`, 200)
}

func TestIntegrationConcurrentChangeAndVerification(t *testing.T) {
	h := setup(t)
	h.expect("POST", "/auth/register", `{"email":"person@example.com","nickname":"author","password":"original password 123"}`, 202)
	token := h.mailToken("verify")
	var changeStatus, verifyStatus int
	var wg sync.WaitGroup
	wg.Go(func() {
		changeStatus, _, _ = h.call("POST", "/auth/change-unverified-email", changeEmailBody, true)
	})
	wg.Go(func() {
		verifyStatus, _, _ = h.call("POST", "/auth/verify-email", `{"token":"`+token+`"}`, true)
	})
	wg.Wait()
	switch {
	case changeStatus == 202 && verifyStatus == 400:
		u, err := h.server.q.UserByEmail(t.Context(), "corrected@example.com")
		if err != nil || u.EmailVerified {
			t.Fatal("old link verified the corrected address", err)
		}
	case changeStatus == 409 && verifyStatus == 200:
		u, err := h.server.q.UserByEmail(t.Context(), "person@example.com")
		if err != nil || !u.EmailVerified {
			t.Fatal("confirmed account changed address", err)
		}
	default:
		t.Fatalf("unexpected concurrent statuses: change=%d verify=%d", changeStatus, verifyStatus)
	}
}

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
