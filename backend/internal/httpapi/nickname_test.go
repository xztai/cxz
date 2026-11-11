package httpapi

import (
	"strings"
	"sync"
	"testing"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/migrations"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestIntegrationNicknames(t *testing.T) {
	h := setup(t)
	h.expect("POST", "/auth/register", `{"email":"person@example.com","nickname":"Maker_1-Test","password":"original password 123"}`, 202)
	u, err := h.server.q.UserByEmail(t.Context(), "person@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if u.Nickname != "maker_1-test" {
		t.Fatal("nickname was not canonicalized")
	}
	body := h.expect("POST", "/auth/register", `{"email":"other@example.com","nickname":"MAKER_1-test","password":"original password 123"}`, 409)
	if !strings.Contains(body, `"code":"nickname_exists"`) {
		t.Fatal("wrong nickname conflict")
	}
	for _, value := range []string{"dot.name", "пользователь", "bad/name", "bad?name", "bad#name"} {
		h.expect("POST", "/auth/register", `{"email":"another@example.com","nickname":"`+value+`","password":"original password 123"}`, 422)
	}
	// Legacy names are no longer accepted by the public API.
	h.expect("POST", "/auth/register", `{"email":"another@example.com","name":"Legacy","password":"original password 123"}`, 422)
}

func TestIntegrationConcurrentNicknames(t *testing.T) {
	h := setup(t)
	results := make(chan int, 2)
	var wg sync.WaitGroup
	for _, body := range []string{
		`{"email":"one@example.com","nickname":"same_handle","password":"original password 123"}`,
		`{"email":"two@example.com","nickname":"SAME_HANDLE","password":"original password 123"}`,
	} {
		wg.Go(func() { status, _, _ := h.call("POST", "/auth/register", body, true); results <- status })
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for status := range results {
		counts[status]++
	}
	if counts[202] != 1 || counts[409] != 1 {
		t.Fatalf("concurrent registration results: %v", counts)
	}
	var users, mails int
	if err := h.pool.QueryRow(t.Context(), "SELECT count(*) FROM users").Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := h.pool.QueryRow(t.Context(), "SELECT count(*) FROM mail_outbox").Scan(&mails); err != nil {
		t.Fatal(err)
	}
	if users != 1 || mails != 1 {
		t.Fatal("duplicate nickname created a user or mail")
	}
}

func TestIntegrationNicknameMigration(t *testing.T) {
	h := setup(t)
	db := stdlib.OpenDBFromPool(h.pool)
	defer func() { _ = db.Close() }()
	provider, err := migrations.Provider(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	hash, err := auth.HashPassword("original password 123")
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := h.pool.QueryRow(t.Context(), "INSERT INTO users (email,name,password_hash,email_verified) VALUES ('legacy@example.com','Прежнее имя',$1,true) RETURNING id", hash).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := h.pool.Exec(t.Context(), "INSERT INTO sessions (token_hash,user_id,expires_at) VALUES ('legacy-session',$1,now()+interval '1 day')", id); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(t.Context()); err != nil {
		t.Fatal(err)
	}
	u, err := h.server.q.SessionUser(t.Context(), "legacy-session")
	if err != nil {
		t.Fatal(err)
	}
	if u.Nickname != "user-1" || u.Name != "Прежнее имя" || u.PasswordHash != hash || !u.EmailVerified {
		t.Fatal("migration changed existing account data or session")
	}
}
