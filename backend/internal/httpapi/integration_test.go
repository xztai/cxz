package httpapi

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/config"
	"etheyl.local/backend/internal/mailqueue"
	"etheyl.local/backend/internal/store"
	"etheyl.local/backend/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

type harness struct {
	t      *testing.T
	pool   *pgxpool.Pool
	url    string
	client *http.Client
	csrf   string
	ip     string
	server *Server
}

func setup(t *testing.T) *harness {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL integration tests")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(cfg.ConnConfig.Database, "_test") {
		t.Fatal("integration tests require a database ending in _test")
	}
	cfg.MaxConns = 16
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	db := stdlib.OpenDBFromPool(pool)
	provider, err := migrations.Provider(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(t.Context()); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	if _, err := pool.Exec(t.Context(), "TRUNCATE users, sessions, action_tokens, mail_outbox, rate_limits RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	s, err := New(pool, config.Config{Origin: "http://example.test", OutboxKeys: testOutboxKey}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	h := &harness{t: t, pool: pool, url: ts.URL, client: &http.Client{Jar: jar}, server: s}
	_, data, _ := h.call("GET", "/auth/csrf", "", false)
	var token struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		t.Fatal(err)
	}
	h.csrf = token.Token
	return h
}
func (h *harness) call(method, path, body string, csrf bool) (int, string, []*http.Cookie) {
	h.t.Helper()
	r, err := http.NewRequestWithContext(h.t.Context(), method, h.url+"/api/v1"+path, strings.NewReader(body))
	if err != nil {
		h.t.Fatal(err)
	}
	if h.ip != "" {
		r.Header.Set("X-Real-IP", h.ip)
	}
	r.Header.Set("Content-Type", "application/json")
	if csrf {
		r.Header.Set("Origin", "http://example.test")
		r.Header.Set("X-CSRF-Token", h.csrf)
	}
	resp, err := h.client.Do(r)
	if err != nil {
		h.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	value, err := io.ReadAll(resp.Body)
	if err != nil {
		h.t.Fatal(err)
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		h.t.Fatal("missing no-store")
	}
	return resp.StatusCode, string(value), resp.Cookies()
}
func (h *harness) expect(method, path, body string, status int) string {
	h.t.Helper()
	got, data, _ := h.call(method, path, body, method != "GET")
	if got != status {
		h.t.Fatalf("%s %s: status %d, expected %d; %s", method, path, got, status, data)
	}
	return data
}
func (h *harness) mailToken(purpose string) string {
	h.t.Helper()
	var body, recipient, tokenHash string
	subject := "Подтвердите email — etheyl"
	if purpose == "reset" {
		subject = "Восстановление пароля — etheyl"
	}
	if err := h.pool.QueryRow(h.t.Context(), "SELECT body,recipient,token_hash FROM mail_outbox WHERE subject=$1 ORDER BY id DESC LIMIT 1", subject).Scan(&body, &recipient, &tokenHash); err != nil {
		h.t.Fatal(err)
	}
	box, err := mailqueue.NewCipher(testOutboxKey)
	if err != nil {
		h.t.Fatal(err)
	}
	plain, err := box.Open(body, mailqueue.Metadata(recipient, subject, tokenHash))
	if err != nil {
		h.t.Fatal(err)
	}
	if strings.Contains(body, "#token=") {
		h.t.Fatal("plaintext persisted")
	}
	body = plain
	_, rest, ok := strings.Cut(body, "#token=")
	if !ok {
		h.t.Fatal("mail lacks token")
	}
	token, _, _ := strings.Cut(rest, "\n")
	return token
}
func (h *harness) registerAndVerify() {
	h.t.Helper()
	h.expect("POST", "/auth/register", `{"email":"person@example.com","nickname":"author","password":"original password 123"}`, 202)
	token := h.mailToken("verify")
	h.expect("POST", "/auth/verify-email", `{"token":"`+token+`"}`, 200)
}

const loginBody = `{"email":"person@example.com","password":"original password 123"}`

func TestIntegrationAuthLifecycle(t *testing.T) {
	h := setup(t)
	h.expect("POST", "/auth/register", `{"email":"person@example.com","nickname":"author","password":"original password 123","isAdmin":true}`, 422)
	h.expect("POST", "/auth/register", `{"email":" Person@Example.COM ","nickname":"author","password":"original password 123"}`, 202)
	u, err := h.server.q.UserByEmail(t.Context(), "person@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if u.EmailVerified || u.IsAdmin || u.PasswordHash == "original password 123" {
		t.Fatal("registration persisted unsafe user")
	}
	originalToken := h.mailToken("verify")
	h.expect("POST", "/auth/register", `{"email":"PERSON@example.com","nickname":"another_author","password":"different password 456"}`, 409)
	unchanged, err := h.server.q.UserByEmail(t.Context(), u.Email)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Nickname != u.Nickname || unchanged.PasswordHash != u.PasswordHash || h.mailToken("verify") != originalToken {
		t.Fatal("duplicate registration changed the pending account or its token")
	}
	var mails int
	if err := h.pool.QueryRow(t.Context(), "SELECT count(*) FROM mail_outbox").Scan(&mails); err != nil {
		t.Fatal(err)
	}
	if mails != 1 {
		t.Fatal("duplicate registration queued another email")
	}
	preview := h.expect("POST", "/auth/verify-email/preview", `{"token":"`+originalToken+`"}`, 200)
	if !strings.Contains(preview, `"email":"person@example.com"`) {
		t.Fatal("preview omitted the email")
	}
	if !strings.Contains(h.expect("GET", "/me", "", 200), `"user":null`) {
		t.Fatal("preview signed in before confirmation")
	}
	h.expect("POST", "/auth/login", loginBody, 403)
	old := h.mailToken("verify")
	h.expect("POST", "/auth/resend-verification", `{"email":"person@example.com"}`, 202)
	h.expect("POST", "/auth/verify-email", `{"token":"`+old+`"}`, 400)
	token := h.mailToken("verify")
	h.expect("POST", "/auth/verify-email", `{"token":"`+token+`"}`, 200)
	if !strings.Contains(h.expect("GET", "/me", "", 200), `"email":"person@example.com"`) {
		t.Fatal("email confirmation did not sign in")
	}
	h.expect("POST", "/auth/verify-email", `{"token":"`+token+`"}`, 400)
	h.expect("POST", "/auth/verify-email/preview", `{"token":"`+token+`"}`, 400)
	h.expect("POST", "/auth/register", `{"email":"person@example.com","nickname":"author","password":"original password 123"}`, 409)
	status, body, cookies := h.call("POST", "/auth/login", loginBody, true)
	if status != 200 || strings.Contains(body, "password") || !strings.Contains(body, `"isAdmin":false`) {
		t.Fatalf("login response: %d %s", status, body)
	}
	var session string
	for _, cookie := range cookies {
		if cookie.Name == "etheyl_session" {
			session = cookie.Value
			if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.Domain != "" {
				t.Fatal("unsafe session cookie")
			}
		}
	}
	if session == "" {
		t.Fatal("no session cookie")
	}
	var hash string
	if err := h.pool.QueryRow(t.Context(), "SELECT token_hash FROM sessions LIMIT 1").Scan(&hash); err != nil || hash == session {
		t.Fatal("raw session persisted")
	}
	if !strings.Contains(h.expect("GET", "/me", "", 200), "person@example.com") {
		t.Fatal("missing current user")
	}
	h.expect("GET", "/admin/status", "", 403)
	h.expect("POST", "/auth/forgot-password", `{"email":"unknown@example.com"}`, 202)
	h.expect("POST", "/auth/forgot-password", `{"email":"person@example.com"}`, 202)
	reset := h.mailToken("reset")
	h.expect("POST", "/auth/reset-password", `{"token":"`+reset+`","password":"original password 123"}`, 422)
	if !strings.Contains(h.expect("GET", "/me", "", 200), `"email":"person@example.com"`) {
		t.Fatal("unchanged password revoked the current session")
	}
	h.expect("POST", "/auth/reset-password", `{"token":"`+reset+`","password":"replacement password 456"}`, 200)
	if !strings.Contains(h.expect("GET", "/me", "", 200), `"user":null`) {
		t.Fatal("password reset did not clear session")
	}
	r, err := http.NewRequestWithContext(t.Context(), "GET", h.url+"/api/v1/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.AddCookie(&http.Cookie{Name: "etheyl_session", Value: session})
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	value, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(value), `"user":null`) {
		t.Fatal("stolen old session survived reset")
	}
	h.expect("POST", "/auth/reset-password", `{"token":"`+reset+`","password":"another password 789"}`, 400)
	h.expect("POST", "/auth/login", loginBody, 401)
	h.expect("POST", "/auth/login", `{"email":"person@example.com","password":"replacement password 456"}`, 200)
	h.expect("POST", "/auth/logout", "", 200)
	if !strings.Contains(h.expect("GET", "/me", "", 200), `"user":null`) {
		t.Fatal("logout did not revoke session")
	}
}

func TestIntegrationPasswordResetClearsFailedLoginLimits(t *testing.T) {
	h := setup(t)
	h.registerAndVerify()
	keys := []string{
		"login:127.0.0.1:person@example.com",
		"risk:person@example.com",
		"login:203.0.113.10:person@example.com",
		"login:127.0.0.1:other@example.com",
	}
	for _, key := range keys {
		for range 10 {
			if _, err := h.server.q.RateLimit(t.Context(), auth.Digest(key)); err != nil {
				t.Fatal(err)
			}
		}
	}
	h.expect("POST", "/auth/login", loginBody, 429)
	h.expect("POST", "/auth/forgot-password", `{"email":"person@example.com"}`, 202)
	token := h.mailToken("reset")
	h.expect("POST", "/auth/reset-password", `{"token":"`+strings.Repeat("a", 43)+`","password":"replacement password 456"}`, 400)
	h.expect("POST", "/auth/reset-password", `{"token":"`+token+`","password":"original password 123"}`, 422)
	for _, key := range keys {
		if hits, err := h.server.q.FailedLoginHits(t.Context(), auth.Digest(key)); err != nil || hits != 10 {
			t.Fatal("rejected reset changed the failed-login limit")
		}
	}
	// Forwarded IPs are ignored unless proxy trust is explicitly enabled.
	h.ip = "203.0.113.10"
	h.expect("POST", "/auth/reset-password", `{"token":"`+token+`","password":"replacement password 456"}`, 200)
	for i, key := range keys {
		want := int32(10)
		if i < 2 {
			want = 0
		}
		if hits, err := h.server.q.FailedLoginHits(t.Context(), auth.Digest(key)); err != nil || hits != want {
			t.Fatal("successful reset did not clear only the current login limit and account risk")
		}
	}
	h.expect("POST", "/auth/login", `{"email":"person@example.com","password":"replacement password 456"}`, 200)
}

func TestIntegrationCSRFExpiryAndLimits(t *testing.T) {
	h := setup(t)
	if status, _, _ := h.call("POST", "/auth/login", loginBody, false); status != 403 {
		t.Fatal("accepted missing csrf")
	}
	if err := h.server.checkCSRF(httptest.NewRequest("POST", "/", nil)); err == nil {
		t.Fatal("accepted missing origin")
	}
	h.registerAndVerify()
	h.expect("POST", "/auth/login", loginBody, 200)
	if _, err := h.pool.Exec(t.Context(), "UPDATE sessions SET expires_at=now()-interval '1 second'"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.expect("GET", "/me", "", 200), `"user":null`) {
		t.Fatal("accepted expired session")
	}
	h.expect("POST", "/auth/forgot-password", `{"email":"person@example.com"}`, 202)
	token := h.mailToken("reset")
	if _, err := h.pool.Exec(t.Context(), "UPDATE action_tokens SET expires_at=now()-interval '1 second'"); err != nil {
		t.Fatal(err)
	}
	h.expect("POST", "/auth/reset-password", `{"token":"`+token+`","password":"replacement password"}`, 400)
	for range 10 {
		if _, err := h.server.q.RateLimit(t.Context(), auth.Digest("login:127.0.0.1:person@example.com")); err != nil {
			t.Fatal(err)
		}
	}
	// A new server instance still observes the same shared rate limit.
	second, err := New(h.pool, h.server.cfg, h.server.log)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.limit(t.Context(), "login:127.0.0.1:person@example.com", 10); err == nil {
		t.Fatal("rate limit did not survive server restart")
	}
	h.expect("POST", "/auth/login", loginBody, 429)
	if _, err := h.pool.Exec(t.Context(), "UPDATE rate_limits SET expires_at=now()-interval '1 second'"); err != nil {
		t.Fatal(err)
	}
	h.expect("POST", "/auth/login", loginBody, 200)
	h.expect("POST", "/auth/register", `{"email":"bad","nickname":"n","password":"short"}`, 422)
	h.expect("POST", "/auth/login", `{"email":"person@example.com","password":"x","password":"y"}`, 422)
}

func TestIntegrationConcurrentTokenConsumption(t *testing.T) {
	h := setup(t)
	h.registerAndVerify()
	h.expect("POST", "/auth/forgot-password", `{"email":"person@example.com"}`, 202)
	token := h.mailToken("reset")
	results := make(chan int, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			status, _, _ := h.call("POST", "/auth/reset-password", `{"token":"`+token+`","password":"replacement password"}`, true)
			results <- status
		})
	}
	wg.Wait()
	close(results)
	successes := 0
	failures := 0
	for status := range results {
		switch status {
		case 200:
			successes++
		case 400:
			failures++
		default:
			t.Fatalf("unexpected concurrent result %d", status)
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatal("token was not consumed exactly once")
	}
}

type testSender struct {
	mu    sync.Mutex
	calls int
	fail  bool
}

func (s *testSender) Send(_ context.Context, _ store.MailOutbox) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.fail {
		return errors.New("SMTP unavailable")
	}
	time.Sleep(20 * time.Millisecond)
	return nil
}
func TestIntegrationMailQueue(t *testing.T) {
	h := setup(t)
	h.registerAndVerify()
	h.expect("POST", "/auth/forgot-password", `{"email":"person@example.com"}`, 202)
	if err := h.server.q.CleanupMail(t.Context()); err != nil {
		t.Fatal(err)
	}
	sender := &testSender{}
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			if _, err := mailqueue.DeliverOne(t.Context(), h.pool, sender, testCipher(t), h.server.log); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if sender.calls != 1 {
		t.Fatal("concurrent workers sent the same row")
	}
	h.expect("POST", "/auth/forgot-password", `{"email":"person@example.com"}`, 202)
	sender.fail = true
	if _, err := mailqueue.DeliverOne(t.Context(), h.pool, sender, testCipher(t), h.server.log); err != nil {
		t.Fatal(err)
	}
	var attempts int
	var deferred bool
	if err := h.pool.QueryRow(t.Context(), "SELECT attempts, available_at>now() FROM mail_outbox LIMIT 1").Scan(&attempts, &deferred); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 || !deferred {
		t.Fatal("mail failure was not deferred")
	}
}

func TestCookieAndProxyBoundaries(t *testing.T) {
	s := &Server{cfg: config.Config{Origin: "https://etheyl.test", SecureCookies: true}}
	w := httptest.NewRecorder()
	s.cookie(w, s.sessionCookieName(), "test", 3600)
	c := w.Result().Cookies()[0]
	if !c.Secure || !c.HttpOnly || c.Name != "__Host-etheyl_session" || c.Domain != "" {
		t.Fatal("unsafe HTTPS cookie")
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.0.2.1:2345"
	r.Header.Set("X-Real-IP", "198.51.100.1")
	if s.clientIP(r) != "192.0.2.1" {
		t.Fatal("trusted spoofed proxy header")
	}
	token, err := auth.Token()
	if err != nil {
		t.Fatal(err)
	}
	r = httptest.NewRequest("POST", "/", nil)
	r.AddCookie(&http.Cookie{Name: s.csrfCookieName(), Value: token})
	r.Header.Set("X-CSRF-Token", token)
	r.Header.Set("Origin", "https://evil.test")
	if s.checkCSRF(r) == nil {
		t.Fatal("accepted foreign origin")
	}
	r.Header.Set("Origin", s.cfg.Origin)
	if err := s.checkCSRF(r); err != nil {
		t.Fatal(err)
	}
	r.Header.Set("X-CSRF-Token", "bad")
	if s.checkCSRF(r) == nil {
		t.Fatal("accepted invalid header")
	}
}

const testOutboxKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func testCipher(t *testing.T) *mailqueue.Cipher {
	t.Helper()
	box, err := mailqueue.NewCipher(testOutboxKey)
	if err != nil {
		t.Fatal(err)
	}
	return box
}
