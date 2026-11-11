package httpapi

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"etheyl.local/backend/internal/mailqueue"
	"etheyl.local/backend/migrations"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestIntegrationLoginOwnerAfterAttack(t *testing.T) {
	h := setup(t)
	h.registerAndVerify()
	h.server.cfg.TrustProxy = true
	h.ip = "198.51.100.1"
	for range 10 {
		h.expect("POST", "/auth/login", `{"email":"person@example.com","password":"wrong password"}`, 401)
	}
	h.expect("POST", "/auth/login", loginBody, 429)
	h.ip = "198.51.100.2"
	h.expect("POST", "/auth/login", loginBody, 200)
	for range 11 {
		h.expect("POST", "/auth/login", loginBody, 200)
	}
}

func TestIntegrationLockedLoginDeadline(t *testing.T) {
	h := setup(t)
	h.registerAndVerify()
	h.server.cfg.RequestTimeout = 250 * time.Millisecond
	tx, err := h.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(t.Context(), "SELECT id FROM users FOR UPDATE"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	started := time.Now()
	for range 4 {
		wg.Go(func() {
			status, body, _ := h.call("POST", "/auth/login", loginBody, true)
			if status != 503 || strings.Contains(body, `"busy"`) {
				t.Errorf("locked login: %d %s", status, body)
			}
		})
	}
	h.expect("POST", "/auth/logout", "", 200)
	wg.Wait()
	if time.Since(started) > 2*time.Second {
		t.Fatal("SQL outlived request deadline")
	}
	if _, err := tx.Exec(t.Context(), "SELECT 1"); err != nil {
		t.Fatal("blocking transaction unexpectedly canceled", err)
	}
}

func TestIntegrationMailFailureRecovery(t *testing.T) {
	h := setup(t)
	h.registerAndVerify()
	h.expect("POST", "/auth/forgot-password", `{"email":"person@example.com"}`, 202)
	if err := h.server.q.CleanupMail(t.Context()); err != nil {
		t.Fatal(err)
	}
	sender := &testSender{fail: true}
	for range 8 {
		if _, err := h.pool.Exec(t.Context(), "UPDATE mail_outbox SET available_at=now()"); err != nil {
			t.Fatal(err)
		}
		if _, err := mailqueue.DeliverOne(t.Context(), h.pool, sender, testCipher(t), h.server.log); err != nil {
			t.Fatal(err)
		}
	}
	var id int64
	var attempts int
	var failed bool
	if err := h.pool.QueryRow(t.Context(), "SELECT id,attempts,failed_at IS NOT NULL FROM mail_outbox").Scan(&id, &attempts, &failed); err != nil {
		t.Fatal(err)
	}
	if attempts != 8 || !failed {
		t.Fatal("missing terminal state")
	}
	if found, err := mailqueue.DeliverOne(t.Context(), h.pool, sender, testCipher(t), h.server.log); err != nil || found {
		t.Fatal("dead letter selected again", err)
	}
	monitor := mailqueue.NewMonitor()
	monitor.Beat()
	monitor.Refresh(t.Context(), h.pool, h.server.log)
	w := httptest.NewRecorder()
	mailqueue.Status(w, httptest.NewRequest("GET", "/api/v1/mail/status", nil), h.pool, monitor)
	if w.Code != 503 || !strings.Contains(w.Body.String(), `"failed":1`) {
		t.Fatal("delivery failure invisible", w.Body.String())
	}
	sender.fail = false
	if n, err := h.server.q.RequeueMail(t.Context(), id); err != nil || n != 1 {
		t.Fatal("cannot requeue valid action", err)
	}
	if _, err := mailqueue.DeliverOne(t.Context(), h.pool, sender, testCipher(t), h.server.log); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 9 {
		t.Fatal("SMTP recovery failed")
	}
	h.expect("POST", "/auth/forgot-password", `{"email":"person@example.com"}`, 202)
	if _, err := h.pool.Exec(t.Context(), "UPDATE action_tokens SET expires_at=now()-interval '1 second'; UPDATE mail_outbox SET failed_at=now()"); err != nil {
		t.Fatal(err)
	}
	if err := h.pool.QueryRow(t.Context(), "SELECT id FROM mail_outbox LIMIT 1").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if n, err := h.server.q.RequeueMail(t.Context(), id); err != nil || n != 0 {
		t.Fatal("expired mail requeued", err)
	}
	if _, err := h.pool.Exec(t.Context(), "UPDATE mail_outbox SET failed_at=NULL"); err != nil {
		t.Fatal(err)
	}
	if _, err := mailqueue.DeliverOne(t.Context(), h.pool, sender, testCipher(t), h.server.log); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 9 {
		t.Fatal("expired link sent")
	}
}

func TestIntegrationConcurrentMigrators(t *testing.T) {
	h := setup(t)
	db := stdlib.OpenDBFromPool(h.pool)
	defer func() { _ = db.Close() }()
	p, err := migrations.Provider(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.DownTo(t.Context(), 0); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			separate := stdlib.OpenDBFromPool(h.pool)
			defer func() { _ = separate.Close() }()
			provider, err := migrations.Provider(separate)
			if err != nil {
				t.Error(err)
				return
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			if _, err := provider.Up(ctx); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	var count int
	if err := h.pool.QueryRow(t.Context(), "SELECT count(*) FROM goose_db_version WHERE is_applied").Scan(&count); err != nil || count != 4 {
		t.Fatal("unexpected migration history", count, err)
	}
}
