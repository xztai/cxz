package httpapi

import (
	"encoding/json/v2"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/store"
)

const quotaCommentBody = `{"body":"Комментарий"}`

func commentQuotaFixture(t *testing.T) (*harness, string, int64, string) {
	t.Helper()
	h := publicationHarness(t)
	post := postResult(t, h.expect("POST", "/me/posts", postBody("null"), 201))
	h.expect("POST", "/me/posts/"+post.ID+"/publish", "", 200)
	newTestSession(t, h, "visitor")
	var visitorID int64
	if err := h.pool.QueryRow(t.Context(), "SELECT id FROM users WHERE nickname=$1", "visitor").Scan(&visitorID); err != nil {
		t.Fatal(err)
	}
	return h, "/posts/" + post.ID + "/comments", mustID(t, post.ID), auth.Digest("comment:" + strconv.FormatInt(visitorID, 10))
}

func submitQuotaComment(t *testing.T, h *harness, path string, wantStatus int) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, h.url+"/api/v1"+path, strings.NewReader(quotaCommentBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.test")
	req.Header.Set("X-CSRF-Token", h.csrf)
	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != wantStatus {
		t.Fatalf("create comment status = %d, want %d", resp.StatusCode, wantStatus)
	}
	if wantStatus == http.StatusTooManyRequests {
		if got := resp.Header.Get("Retry-After"); got != "900" {
			t.Fatalf("Retry-After = %q, want 900", got)
		}
		var result struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.UnmarshalRead(resp.Body, &result); err != nil {
			t.Fatal("could not decode the rate-limit response")
		}
		if result.Error.Code != "rate_limit" {
			t.Fatal("expected rate_limit error code")
		}
	}
}

func assertCommentQuotaCounts(t *testing.T, h *harness, postID int64, key string, wantComments, wantHits int) {
	t.Helper()
	var comments, hits int
	if err := h.pool.QueryRow(t.Context(), "SELECT count(*) FROM post_comments WHERE post_id=$1", postID).Scan(&comments); err != nil {
		t.Fatal(err)
	}
	if comments != wantComments {
		t.Fatalf("saved comments = %d, want %d", comments, wantComments)
	}
	if err := h.pool.QueryRow(t.Context(), "SELECT hits FROM rate_limits WHERE key=$1", key).Scan(&hits); err != nil {
		t.Fatal(err)
	}
	if hits != wantHits {
		t.Fatalf("committed quota hits = %d, want %d", hits, wantHits)
	}
}

func TestIntegrationCommentQuotaSingleConnection(t *testing.T) {
	h, path, postID, key := commentQuotaFixture(t)
	t.Run("create with one available connection", func(t *testing.T) {
		for range h.pool.Config().MaxConns - 1 {
			conn, err := h.pool.Acquire(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(conn.Release)
		}
		submitQuotaComment(t, h, path, http.StatusCreated)
	})
	assertCommentQuotaCounts(t, h, postID, key, 1, 1)
}

func TestIntegrationCommentQuotaExpiryAndRollback(t *testing.T) {
	h, path, postID, key := commentQuotaFixture(t)
	for range 20 {
		submitQuotaComment(t, h, path, http.StatusCreated)
	}
	for range 2 {
		submitQuotaComment(t, h, path, http.StatusTooManyRequests)
	}
	assertCommentQuotaCounts(t, h, postID, key, 20, 20)

	if _, err := h.pool.Exec(t.Context(), "UPDATE rate_limits SET expires_at=now()-interval '1 second' WHERE key=$1", key); err != nil {
		t.Fatal(err)
	}
	submitQuotaComment(t, h, path, http.StatusCreated)
	assertCommentQuotaCounts(t, h, postID, key, 21, 1)

	const rollbackKey = "test:comment-quota-rollback"
	tx, err := h.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer socialRollback(tx)
	if err := limit(t.Context(), store.New(tx), rollbackKey, 20); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(t.Context()); err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err := h.pool.QueryRow(t.Context(), "SELECT EXISTS (SELECT 1 FROM rate_limits WHERE key=$1)", auth.Digest(rollbackKey)).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("rolled back quota increment left a counter row")
	}
}
