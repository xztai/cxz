package httpapi

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type catalogQueryCounter struct {
	store.DBTX
	calls int
}

func (c *catalogQueryCounter) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	c.calls++
	return c.DBTX.Query(ctx, sql, args...)
}

func (c *catalogQueryCounter) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	c.calls++
	return c.DBTX.QueryRow(ctx, sql, args...)
}

func (c *catalogQueryCounter) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	c.calls++
	return c.DBTX.Exec(ctx, sql, args...)
}

func TestIntegrationCatalogQueryCount(t *testing.T) {
	h := setup(t)
	ctx := t.Context()

	const eligible = 26
	var profiles [eligible + 2]Profile
	for i := range profiles {
		nickname := fmt.Sprintf("creator-%02d", i)
		displayName := "Directory author"
		switch i {
		case 0:
			displayName = "Mixed Case 100%_ Match"
		case 1:
			displayName = "Mixed Case 100px Match"
		}
		const description = "Public creator description"
		var id int64
		err := h.pool.QueryRow(ctx, `
INSERT INTO users (email, nickname, name, password_hash, email_verified, blocked)
VALUES ($1, $2, $3, 'unused', $4, $5)
RETURNING id`, nickname+"@catalog.example", nickname, displayName, i != eligible+1, i == eligible).Scan(&id)
		if err != nil {
			t.Fatalf("seed creator %d: %v", i, err)
		}
		createdAt := "2025-01-01T00:00:00Z"
		if i == 0 {
			createdAt = "2025-01-02T00:00:00Z"
		}
		_, err = h.pool.Exec(ctx, `
INSERT INTO creator_profiles (user_id, display_name, description, created_at)
VALUES ($1, $2, $3, $4::text::timestamptz)`, id, displayName, description, createdAt)
		if err != nil {
			t.Fatalf("seed creator profile %d: %v", i, err)
		}
		profiles[i] = Profile{ID: strconv.FormatInt(id, 10), Nickname: nickname, DisplayName: displayName, Description: description}
	}

	addImage := func(index int, kind, mimeType, hash string, size int64, width, height int32) *ImageMetadata {
		t.Helper()
		p := profiles[index]
		id, err := strconv.ParseInt(p.ID, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		err = h.server.q.SaveProfileImage(ctx, store.SaveProfileImageParams{
			UserID: id, Kind: kind, ObjectKey: "catalog-private/" + p.Nickname + "/" + kind,
			ContentType: mimeType, SizeBytes: size, Width: width, Height: height, Sha256: hash,
		})
		if err != nil {
			t.Fatalf("seed creator %d %s: %v", index, kind, err)
		}
		return &ImageMetadata{
			URL:         "/api/v1/authors/" + p.Nickname + "/images/" + kind + "?v=" + hash,
			ContentType: mimeType,
			SizeBytes:   size,
			Width:       width,
			Height:      height,
		}
	}
	profiles[0].Avatar = addImage(0, "avatar", "image/png", strings.Repeat("a", 64), 1001, 32, 48)
	profiles[0].Cover = addImage(0, "cover", "image/jpeg", strings.Repeat("b", 64), 2002, 640, 480)
	profiles[25].Avatar = addImage(25, "avatar", "image/jpeg", strings.Repeat("c", 64), 3003, 96, 64)
	profiles[1].Cover = addImage(1, "cover", "image/png", strings.Repeat("d", 64), 4004, 800, 400)

	order := []int{0}
	for i := eligible - 1; i > 0; i-- {
		order = append(order, i)
	}
	for _, tc := range []struct {
		name    string
		query   string
		order   []int
		hasMore bool
	}{
		{name: "full page", order: order[:24], hasMore: true},
		{name: "next page", query: "?offset=24", order: order[24:]},
		{name: "literal case-insensitive search", query: "?q=" + url.QueryEscape("mixed CASE 100%_ mATCH"), order: []int{0}},
		{name: "empty wildcard search", query: "?q=" + url.QueryEscape("%_missing")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			counter := &catalogQueryCounter{DBTX: h.pool}
			server := *h.server
			server.q = store.New(counter)
			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/authors"+tc.query, nil)
			server.Handler().ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("catalog status = %d, want 200", rec.Code)
			}
			if counter.calls != 1 {
				t.Errorf("database calls = %d, want 1", counter.calls)
			}

			body := rec.Body.Bytes()
			var page struct {
				Authors []Profile `json:"authors"`
				HasMore *bool     `json:"hasMore"`
			}
			if err := json.Unmarshal(body, &page); err != nil {
				t.Fatal(err)
			}
			if page.Authors == nil || len(page.Authors) != len(tc.order) || page.HasMore == nil || *page.HasMore != tc.hasMore {
				t.Fatalf("unexpected pagination, authors = %d, hasMore = %v", len(page.Authors), page.HasMore)
			}
			if bytes.Contains(body, []byte("@catalog.example")) || bytes.Contains(body, []byte("catalog-private/")) {
				t.Error("catalog exposes private fixture values")
			}

			for _, key := range []string{`"avatar"`, `"cover"`} {
				if count := bytes.Count(body, []byte(key)); count != len(tc.order) {
					t.Errorf("%s field count = %d, want %d", key, count, len(tc.order))
				}
			}
			for i, index := range tc.order {
				got, want := page.Authors[i], profiles[index]
				for _, image := range []struct {
					kind string
					got  *ImageMetadata
					want *ImageMetadata
				}{
					{kind: "avatar", got: got.Avatar, want: want.Avatar},
					{kind: "cover", got: got.Cover, want: want.Cover},
				} {
					if image.got == nil || image.want == nil {
						if image.got != image.want {
							t.Errorf("author %d %s has incorrect nullability", i, image.kind)
						}
					} else if *image.got != *image.want {
						t.Errorf("author %d %s has incorrect metadata", i, image.kind)
					}
				}
				got.Avatar, got.Cover, want.Avatar, want.Cover = nil, nil, nil, nil
				if got != want {
					t.Errorf("author %d = %s, want profile for %s", i, got.Nickname, want.Nickname)
				}
			}
		})
	}
}
