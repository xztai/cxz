package httpapi

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"sync"
	"testing"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/media"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type memoryImages struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func (m *memoryImages) Put(_ context.Context, key string, img media.Image) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = bytes.Clone(img.Data)
	return nil
}
func (m *memoryImages) Get(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.objects[key]
	if !ok {
		return nil, fmt.Errorf("missing object")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}
func (m *memoryImages) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	return nil
}
func testTier(t *testing.T, body string) Tier {
	t.Helper()
	var result struct {
		Tier Tier `json:"tier"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	return result.Tier
}
func newTestSession(t *testing.T, h *harness, nickname string) {
	t.Helper()
	u, err := h.server.q.CreateUser(t.Context(), store.CreateUserParams{Email: nickname + "@example.com", Nickname: nickname, PasswordHash: "unused", EmailVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	token, err := auth.Token()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.pool.Exec(t.Context(), "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '1 day')", auth.Digest(token), u.ID); err != nil {
		t.Fatal(err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	h.client = &http.Client{Jar: jar}
	req, err := http.NewRequestWithContext(t.Context(), "GET", h.url, nil)
	if err != nil {
		t.Fatal(err)
	}
	jar.SetCookies(req.URL, []*http.Cookie{{Name: "etheyl_session", Value: token, Path: "/"}})
	body := h.expect("GET", "/auth/csrf", "", 200)
	var csrf struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(body), &csrf); err != nil {
		t.Fatal(err)
	}
	h.csrf = csrf.Token
}
func TestIntegrationCreatorsAndTiers(t *testing.T) {
	h := setup(t)
	h.expect("GET", "/me/profile", "", 401)
	h.expect("PUT", "/me/profile", `{"nickname":"author","displayName":"Автор","description":""}`, 401)
	newTestSession(t, h, "author")
	h.expect("GET", "/me/profile", "", 200)
	h.expect("POST", "/me/tiers", `{"name":"Ранний","description":"","priceKopecks":100}`, 409)
	h.expect("PUT", "/me/profile", `{"nickname":"bad.name","displayName":"Автор","description":""}`, 422)
	h.expect("PUT", "/me/profile", `{"nickname":"AUTHOR","displayName":"Анна — керамика","description":"Дневник мастерской"}`, 200)
	body := h.expect("GET", "/authors?q=КЕРАМИКА", "", 200)
	if !strings.Contains(body, "Анна") || strings.Contains(body, "email") || strings.Contains(body, "object_key") {
		t.Fatal(body)
	}
	h.expect("GET", "/authors?q=%25", "", 200)
	h.expect("GET", "/authors?offset=-1", "", 422)
	h.expect("PUT", "/me/profile", `{"nickname":"author","displayName":"Взлом","description":"","userId":"2"}`, 422)
	low := testTier(t, h.expect("POST", "/me/tiers", `{"name":"Читатель","description":"Новости","priceKopecks":9900}`, 201))
	high := testTier(t, h.expect("POST", "/me/tiers", `{"name":"Друг","description":"Всё младшее","priceKopecks":29950}`, 201))
	if high.Position <= low.Position {
		t.Fatal("new tier was not placed above old tier")
	}
	h.expect("POST", "/me/tiers", `{"name":"Дробь","description":"","priceKopecks":0.5}`, 422)
	h.expect("POST", "/me/tiers", `{"name":"Нет цены","description":""}`, 422)
	includes, err := h.server.q.TierIncludes(t.Context(), store.TierIncludesParams{GrantedID: mustID(t, high.ID), RequiredID: mustID(t, low.ID)})
	if err != nil || !includes {
		t.Fatal("higher tier must include lower", err)
	}
	includes, err = h.server.q.TierIncludes(t.Context(), store.TierIncludesParams{GrantedID: mustID(t, low.ID), RequiredID: mustID(t, high.ID)})
	if err != nil || includes {
		t.Fatal("lower tier included higher", err)
	}
	order := fmt.Sprintf(`{"ids":[%q,%q]}`, low.ID, high.ID)
	h.expect("PUT", "/me/tiers/order", order, 200)
	h.expect("PUT", "/me/tiers/order", fmt.Sprintf(`{"ids":[%q,%q]}`, high.ID, low.ID), 200)
	if _, err := h.server.q.MarkTierUsed(t.Context(), store.MarkTierUsedParams{ID: mustID(t, low.ID), AuthorID: 1}); err != nil {
		t.Fatal(err)
	}
	h.expect("PUT", "/me/tiers/order", order, 409)
	h.expect("PUT", "/me/tiers/"+low.ID, `{"name":"Читатель+","description":"Новое описание","priceKopecks":10000}`, 200)
	if _, err := h.pool.Exec(t.Context(), "DELETE FROM subscription_tiers WHERE id=$1", mustID(t, low.ID)); err == nil {
		t.Fatal("database deleted a used tier")
	}
	if _, err := h.pool.Exec(t.Context(), "UPDATE subscription_tiers SET used_at=NULL WHERE id=$1", mustID(t, low.ID)); err == nil {
		t.Fatal("database erased usage")
	}
	h.expect("DELETE", "/me/tiers/"+low.ID, "", 200)
	var archived bool
	if err := h.pool.QueryRow(t.Context(), "SELECT archived_at IS NOT NULL FROM subscription_tiers WHERE id=$1", mustID(t, low.ID)).Scan(&archived); err != nil || !archived {
		t.Fatal("used tier not archived", err)
	}
	h.expect("PUT", "/me/tiers/"+low.ID, `{"name":"Недоступно","description":"","priceKopecks":0}`, 404)
	public := h.expect("GET", "/authors/author", "", 200)
	if strings.Contains(public, "Читатель+") {
		t.Fatal("archive is public")
	}
	h.expect("DELETE", "/me/tiers/"+high.ID, "", 200)
	newTestSession(t, h, "visitor")
	h.expect("PUT", "/me/profile", `{"nickname":"author","displayName":"Другой","description":""}`, 409)
	h.expect("PUT", "/me/profile", `{"nickname":"visitor","displayName":"Другой","description":""}`, 200)
	h.expect("PUT", "/me/tiers/"+low.ID, `{"name":"Взлом","description":"","priceKopecks":0}`, 404)
	h.expect("POST", "/me/tiers/"+low.ID+"/archive", "", 404)
	h.expect("DELETE", "/me/tiers/"+low.ID, "", 404)
	h.expect("PUT", "/me/tiers/order", fmt.Sprintf(`{"ids":[%q]}`, low.ID), 422)
	own := testTier(t, h.expect("POST", "/me/tiers", `{"name":"Другой автор","description":"","priceKopecks":0}`, 201))
	includes, err = h.server.q.TierIncludes(t.Context(), store.TierIncludesParams{GrantedID: mustID(t, own.ID), RequiredID: mustID(t, low.ID)})
	if err != nil || includes {
		t.Fatal("cross-author access", err)
	}
	h.expect("PUT", "/me/profile", `{"nickname":"new_visitor","displayName":"Другой","description":""}`, 200)
	h.expect("GET", "/authors/visitor", "", 404)
	h.expect("GET", "/authors/new_visitor", "", 200)
}
func mustID(t *testing.T, id string) int64 {
	t.Helper()
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func uploadTestProfileImage(t *testing.T, h *harness, body []byte, kind, mime string, csrf bool) int {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), "PUT", h.url+"/api/v1/me/profile/images/"+kind, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", mime)
	if csrf {
		r.Header.Set("Origin", "http://example.test")
		r.Header.Set("X-CSRF-Token", h.csrf)
	}
	resp, err := h.client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestIntegrationProfileImages(t *testing.T) {
	h := setup(t)
	newTestSession(t, h, "painter")
	h.expect("PUT", "/me/profile", `{"nickname":"painter","displayName":"Художник","description":""}`, 200)
	objects := &memoryImages{objects: make(map[string][]byte)}
	h.server.images = objects
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewNRGBA(image.Rect(0, 0, 12, 6))); err != nil {
		t.Fatal(err)
	}

	if got := uploadTestProfileImage(t, h, data.Bytes(), "avatar", "image/png", false); got != 403 {
		t.Fatal(got)
	}
	if got := uploadTestProfileImage(t, h, []byte("bad"), "avatar", "image/png", true); got != 422 {
		t.Fatal(got)
	}
	if got := uploadTestProfileImage(t, h, data.Bytes(), "avatar", "image/jpeg", true); got != 422 {
		t.Fatal(got)
	}
	for range 2 {
		if got := uploadTestProfileImage(t, h, data.Bytes(), "avatar", "image/png", true); got != 200 {
			t.Fatal(got)
		}
	}
	if len(objects.objects) != 1 {
		t.Fatal("replaced image was not cleaned up")
	}
	body := h.expect("GET", "/authors/painter", "", 200)
	if !strings.Contains(body, `"width":12`) || strings.Contains(body, "profiles/") {
		t.Fatal(body)
	}
	h.expect("GET", "/authors/painter/images/avatar", "", 200)
	newTestSession(t, h, "intruder")
	if got := uploadTestProfileImage(t, h, data.Bytes(), "cover", "image/png", true); got != 409 {
		t.Fatal(got)
	}
	h.expect("PUT", "/me/profile", `{"nickname":"intruder","displayName":"Другой","description":""}`, 200)
	h.expect("DELETE", "/me/profile/images/avatar", "", 200)
	h.expect("GET", "/authors/painter/images/avatar", "", 200)
	if _, err := h.pool.Exec(t.Context(), "UPDATE users SET blocked=true WHERE nickname='painter'"); err != nil {
		t.Fatal(err)
	}
	h.expect("GET", "/authors/painter", "", 404)
	h.expect("GET", "/authors/painter/images/avatar", "", 404)
}

type lostCommitAcknowledgment struct {
	pgx.Tx
	err error
}

func (tx lostCommitAcknowledgment) Commit(ctx context.Context) error {
	if err := tx.Tx.Commit(ctx); err != nil {
		return err
	}
	return tx.err
}

func TestIntegrationProfileImagePersistence(t *testing.T) {
	for _, tc := range []struct {
		name         string
		invalidWidth bool
	}{
		{name: "commit acknowledgment lost"},
		{name: "metadata rejected", invalidWidth: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			h := setup(t)
			newTestSession(t, h, "painter")
			h.expect("PUT", "/me/profile", `{"nickname":"painter","displayName":"Художник","description":""}`, 200)
			objects := &memoryImages{objects: make(map[string][]byte)}
			h.server.images = objects

			pngBytes := func(width int) []byte {
				t.Helper()
				var data bytes.Buffer
				if err := png.Encode(&data, image.NewNRGBA(image.Rect(0, 0, width, 6))); err != nil {
					t.Fatal(err)
				}
				return data.Bytes()
			}
			if got := uploadTestProfileImage(t, h, pngBytes(12), "avatar", "image/png", true); got != http.StatusOK {
				t.Fatalf("seed avatar: status %d", got)
			}
			publicAvatar := "/authors/painter/images/avatar"
			original := h.expect("GET", publicAvatar, "", 200)

			req, err := http.NewRequestWithContext(ctx, "PUT", h.url+"/api/v1/me/profile/images/avatar", nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, cookie := range h.client.Jar.Cookies(req.URL) {
				req.AddCookie(cookie)
			}
			tx, q, id, err := h.server.creatorTx(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			old, err := q.ProfileImages(ctx, id)
			if err != nil || len(old) != 1 {
				t.Fatalf("look up seeded avatar: got %d images, err %v", len(old), err)
			}

			replacement, err := media.Validate(bytes.NewReader(pngBytes(16)), "image/png")
			if err != nil {
				t.Fatal(err)
			}
			key := fmt.Sprintf("profiles/%d/avatar/test-replacement", id)
			if err := objects.Put(ctx, key, replacement); err != nil {
				t.Fatal(err)
			}
			params := store.SaveProfileImageParams{
				UserID: id, Kind: "avatar", ObjectKey: key, ContentType: replacement.ContentType,
				SizeBytes: int64(len(replacement.Data)), Width: int32(replacement.Width),
				Height: int32(replacement.Height), Sha256: replacement.SHA256,
			}
			if tc.invalidWidth {
				params.Width = 0
			}
			lostAck := errors.New("commit acknowledgment lost")
			err = h.server.saveUploadedProfileImage(ctx, lostCommitAcknowledgment{Tx: tx, err: lostAck}, params, old)

			expectedKey, expectedData, expectedCount := key, string(replacement.Data), 2
			if tc.invalidWidth {
				pgErr, ok := errors.AsType[*pgconn.PgError](err)
				if !ok || pgErr.ConstraintName != "profile_images_width_check" {
					t.Fatalf("expected width constraint violation, got %v", err)
				}
				if err := tx.Rollback(ctx); err != nil {
					t.Fatal(err)
				}
				expectedKey, expectedData, expectedCount = old[0].ObjectKey, original, 1
			} else if !errors.Is(err, lostAck) {
				t.Fatalf("expected lost commit acknowledgment, got %v", err)
			}

			profileImage, err := store.New(h.pool).ProfileImage(ctx, store.ProfileImageParams{Nickname: "painter", Kind: "avatar"})
			if err != nil {
				t.Fatalf("look up persisted avatar: %v", err)
			}
			if profileImage.ObjectKey != expectedKey {
				t.Fatal("persisted avatar has the wrong object key")
			}
			objects.mu.Lock()
			_, oldExists := objects.objects[old[0].ObjectKey]
			count := len(objects.objects)
			objects.mu.Unlock()
			if !oldExists || count != expectedCount {
				t.Fatalf("unexpected stored avatars: old exists %t, object count %d (want %d)", oldExists, count, expectedCount)
			}
			if got := h.expect("GET", publicAvatar, "", 200); got != expectedData {
				t.Fatal("public avatar did not return the expected bytes")
			}
		})
	}
}

func TestIntegrationCreatorUnicodeLimits(t *testing.T) {
	h := setup(t)
	newTestSession(t, h, "boundary")
	// A supplementary character is two \u escapes (12 bytes) in legal JSON.
	description := strings.Repeat(`\ud83e\uddea`, 5000)
	body := `{"nickname":"boundary","displayName":"Граница","description":"` + description + `"}`
	h.expect("PUT", "/me/profile", body, 200)
	tooLong := `{"nickname":"boundary","displayName":"Граница","description":"` + description + `x"}`
	h.expect("PUT", "/me/profile", tooLong, 422)
	public := h.expect("GET", "/authors/boundary", "", 200)
	if strings.Count(public, "🧪") != 5000 {
		t.Fatal("Unicode description did not survive persistence")
	}
}
