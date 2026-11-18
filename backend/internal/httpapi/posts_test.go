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
	"maps"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"etheyl.local/backend/internal/media"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
)

type seekCloser struct{ *bytes.Reader }

func (s seekCloser) Close() error { return nil }

type memoryFiles struct {
	mu            sync.Mutex
	data          map[string][]byte
	modified      map[string]time.Time
	failDelete    bool
	failDeleteKey string
	beforeList    func(prefix, after string) error
}

func (m *memoryFiles) Upload(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = data
	return nil
}
func (m *memoryFiles) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.data[key]
	if !ok {
		return nil, errors.New("missing object")
	}
	return seekCloser{bytes.NewReader(data)}, nil
}
func (m *memoryFiles) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failDelete || key == m.failDeleteKey {
		return errors.New("storage unavailable")
	}
	delete(m.data, key)
	delete(m.modified, key)
	return nil
}
func (m *memoryFiles) ListPage(ctx context.Context, prefix, after string, limit int) ([]media.Object, error) {
	if m.beforeList != nil {
		if err := m.beforeList(prefix, after); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	page := make([]media.Object, 0, limit)
	for _, key := range slices.Sorted(maps.Keys(m.data)) {
		if key <= after || !strings.HasPrefix(key, prefix) {
			continue
		}
		modified := m.modified[key]
		if modified.IsZero() {
			modified = time.Now().Add(-2 * time.Hour)
		}
		page = append(page, media.Object{Key: key, Modified: modified})
		if len(page) == limit {
			break
		}
	}
	return page, nil
}
func (m *memoryFiles) CleanupMultipart(context.Context, time.Time) error { return nil }

func publicationHarness(t *testing.T) *harness {
	t.Helper()
	h := setup(t)
	h.server.files = &memoryFiles{data: make(map[string][]byte)}
	if endpoint := os.Getenv("TEST_S3_ENDPOINT"); endpoint != "" {
		objects, err := media.New(endpoint, "etheyl_test", "etheyl_test_password", "etheyl-publications-test", false)
		if err != nil {
			t.Fatal(err)
		}
		if err := objects.Initialize(t.Context()); err != nil {
			t.Fatal(err)
		}
		h.server.files = objects
	}
	newTestSession(t, h, "writer")
	h.expect("PUT", "/me/profile", `{"nickname":"writer","displayName":"Автор","description":""}`, 200)
	return h
}

func postResult(t *testing.T, body string) Post {
	t.Helper()
	var result struct {
		Post Post `json:"post"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	return result.Post
}
func postBody(tier string) string {
	return fmt.Sprintf(`{"title":"История","teaser":"Публичный анонс","bodyMarkdown":"## Секретный текст\n\n<script>alert(1)</script>\n\n[bad](javascript:alert(1))","minimumTierId":%s}`, tier)
}
func uploadFile(t *testing.T, h *harness, id, kind, name string, data []byte, csrf bool, status int) PostFile {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), "POST", h.url+"/api/v1/me/posts/"+id+"/media?"+url.Values{"kind": {kind}, "filename": {name}}.Encode(), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if csrf {
		req.Header.Set("Origin", "http://example.test")
		req.Header.Set("X-CSRF-Token", h.csrf)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != status {
		t.Fatalf("upload: %d, want %d: %s", resp.StatusCode, status, body)
	}
	var result struct {
		Media PostFile `json:"media"`
	}
	if status == 201 {
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal(err)
		}
	}
	return result.Media
}
func mediaRequest(t *testing.T, h *harness, method, path, byteRange string, status int) ([]byte, http.Header) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, h.url+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if byteRange != "" {
		req.Header.Set("Range", byteRange)
	}
	if status == 403 || status == 404 {
		req.Header.Set("If-None-Match", "*")
	} else {
		req.Header.Set("If-None-Match", `"old-etag"`)
	}
	req.Header.Set("If-Modified-Since", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != status {
		t.Fatalf("%s %s: %d, want %d: %s", method, path, resp.StatusCode, status, body)
	}
	if resp.Header.Get("Cache-Control") != "no-store" || resp.Header.Get("Vary") != "Cookie" {
		t.Fatal("unsafe caching", resp.Header)
	}
	if resp.Header.Get("Location") != "" {
		t.Fatal("media redirected")
	}
	return body, resp.Header
}

func fixtureSubscription(t *testing.T, h *harness, tier string) {
	t.Helper()
	if _, err := h.pool.Exec(t.Context(), `INSERT INTO subscriptions(subscriber_id,author_id,tier_id,expires_at)
 SELECT u.id,t.author_id,t.id,now()+interval '1 day' FROM users u,subscription_tiers t WHERE u.nickname='reader' AND t.id=$1
 ON CONFLICT(subscriber_id,author_id) DO UPDATE SET tier_id=EXCLUDED.tier_id,starts_at=now(),expires_at=EXCLUDED.expires_at,revoked_at=NULL`, mustID(t, tier)); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationPublicationsAccessAndMedia(t *testing.T) {
	h := publicationHarness(t)
	low := testTier(t, h.expect("POST", "/me/tiers", `{"name":"Начальный","description":"","priceKopecks":99900}`, 201))
	mid := testTier(t, h.expect("POST", "/me/tiers", `{"name":"Средний","description":"","priceKopecks":100}`, 201))
	high := testTier(t, h.expect("POST", "/me/tiers", `{"name":"Старший","description":"","priceKopecks":0}`, 201))
	p := postResult(t, h.expect("POST", "/me/posts", postBody(fmt.Sprintf("%q", mid.ID)), 201))
	h.expect("PUT", "/me/posts/"+p.ID, `{"title":"Не открывать случайно","teaser":"","bodyMarkdown":"private"}`, 422)
	h.expect("PUT", "/me/posts/"+p.ID, `{"title":"Не открывать случайно","teaser":"","bodyMarkdown":"private","minimumTierId":false}`, 422)
	ownerClient, ownerCSRF := h.client, h.csrf
	if p.BodyHTML == nil || !strings.Contains(*p.BodyHTML, "<h2>") || strings.Contains(*p.BodyHTML, "<script>") || strings.Contains(*p.BodyHTML, `href="javascript:`) {
		t.Fatal("unsafe Markdown", p.BodyHTML)
	}
	var img bytes.Buffer
	if err := png.Encode(&img, image.NewNRGBA(image.Rect(0, 0, 1200, 600))); err != nil {
		t.Fatal(err)
	}
	// Preserve original trailing data; thumbnails are separate, fully decoded PNGs.
	original := append(img.Bytes(), []byte("ORIGINAL-METADATA")...)
	uploadFile(t, h, p.ID, "image", "photo.png", original, false, 403)
	uploadFile(t, h, p.ID, "image", "fake.png", []byte("<script>bad()</script>"), true, 422)
	uploadFile(t, h, p.ID, "image", "../photo.png", original, true, 422)
	picture := uploadFile(t, h, p.ID, "image", "photo.png", original, true, 201)
	audioData := append([]byte("RIFF\x24\x00\x00\x00WAVEfmt "), bytes.Repeat([]byte{0}, 100)...)
	audio := uploadFile(t, h, p.ID, "audio", "voice.wav", audioData, true, 201)
	videoData := append([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'm', 'p', '4', '2', 0, 0, 0, 0, 'm', 'p', '4', '2', 'i', 's', 'o', 'm'}, bytes.Repeat([]byte{1}, 100)...)
	video := uploadFile(t, h, p.ID, "video", "film.mp4", videoData, true, 201)
	attachment := uploadFile(t, h, p.ID, "attachment", "документ.html", []byte("<script>download only</script>"), true, 201)
	files := []PostFile{picture, audio, video, attachment}
	rows, err := h.pool.Query(t.Context(), "SELECT object_key,thumbnail_key FROM post_media")
	if err != nil {
		t.Fatal(err)
	}
	var storedKeys []string
	for rows.Next() {
		var originalKey, thumbnailKey string
		if err := rows.Scan(&originalKey, &thumbnailKey); err != nil {
			t.Fatal(err)
		}
		storedKeys = append(storedKeys, originalKey)
		if thumbnailKey != "" {
			storedKeys = append(storedKeys, thumbnailKey)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if endpoint := os.Getenv("TEST_S3_ENDPOINT"); endpoint != "" {
		for _, key := range storedKeys {
			resp, err := http.Get("http://" + endpoint + "/etheyl-publications-test/" + key)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != 403 {
				t.Fatal("anonymous S3 access", resp.StatusCode)
			}
		}
	}
	bytesOut, _ := mediaRequest(t, h, "GET", picture.URL, "", 200)
	if !bytes.Equal(bytesOut, original) {
		t.Fatal("image original changed")
	}
	thumb, _ := mediaRequest(t, h, "GET", *picture.ThumbnailURL, "", 200)
	cfg, err := png.DecodeConfig(bytes.NewReader(thumb))
	if err != nil || cfg.Width != 640 || cfg.Height != 320 {
		t.Fatal("invalid thumbnail", cfg, err)
	}
	_, headers := mediaRequest(t, h, "GET", attachment.URL, "", 200)
	if !strings.HasPrefix(headers.Get("Content-Disposition"), "attachment;") || headers.Get("Content-Type") != "application/octet-stream" {
		t.Fatal("attachment rendered inline")
	}
	// Anonymous requests cannot discover a draft, even when they know its media IDs.
	h.client = &http.Client{}
	h.expect("GET", "/posts/"+p.ID, "", 404)
	for _, f := range files {
		mediaRequest(t, h, "GET", f.URL, "bytes=0-4", 404)
	}
	h.client, h.csrf = ownerClient, ownerCSRF
	h.expect("POST", "/me/posts/"+p.ID+"/publish", "", 200)
	newTestSession(t, h, "reader")
	readerClient, readerCSRF := h.client, h.csrf
	locked := func() {
		t.Helper()
		body := h.expect("GET", "/posts/"+p.ID, "", 200)
		for _, secret := range []string{"Секретный текст", "bodyMarkdown", "bodyHtml", `"media"`, "photo.png", "posts/"} {
			if strings.Contains(body, secret) {
				t.Fatalf("locked content leaked: %s", body)
			}
		}
		if !strings.Contains(body, "Публичный анонс") {
			t.Fatal("missing explicit teaser")
		}
		for _, f := range files {
			mediaRequest(t, h, "GET", f.URL, "bytes=0-4", 403)
			mediaRequest(t, h, "HEAD", f.URL, "", 403)
		}
		mediaRequest(t, h, "GET", *picture.ThumbnailURL, "", 403)
	}
	locked()
	fixtureSubscription(t, h, low.ID)
	locked()
	for _, tier := range []string{mid.ID, high.ID} {
		fixtureSubscription(t, h, tier)
		readable := postResult(t, h.expect("GET", "/posts/"+p.ID, "", 200))
		if !readable.CanRead || readable.BodyMarkdown == nil || len(readable.Media) != 4 {
			t.Fatal("subscriber denied")
		}
		for _, sample := range []struct {
			file PostFile
			data []byte
		}{{audio, audioData}, {video, videoData}} {
			part, hdr := mediaRequest(t, h, "GET", sample.file.URL, "bytes=3-8", 206)
			if !bytes.Equal(part, sample.data[3:9]) || hdr.Get("Content-Range") != fmt.Sprintf("bytes 3-8/%d", len(sample.data)) {
				t.Fatal("incorrect range", hdr)
			}
			part, _ = mediaRequest(t, h, "GET", sample.file.URL, "bytes=-4", 206)
			if !bytes.Equal(part, sample.data[len(sample.data)-4:]) {
				t.Fatal("incorrect suffix")
			}
			failure, hdr := mediaRequest(t, h, "GET", sample.file.URL, "bytes=999999-", 416)
			if !strings.Contains(string(failure), `"code":"invalid_range"`) || !strings.HasPrefix(hdr.Get("Content-Type"), "application/json") {
				t.Fatal("unstable range error", string(failure))
			}
			mediaRequest(t, h, "HEAD", sample.file.URL, "", 200)
		}
	}
	// Revoked, expired and not-yet-active subscriptions all lose access immediately.
	for _, change := range []string{"revoked_at=now()", "expires_at=now()-interval '1 second',starts_at=now()-interval '1 day'", "starts_at=now()+interval '1 hour'"} {
		fixtureSubscription(t, h, high.ID)
		if _, err := h.pool.Exec(t.Context(), "UPDATE subscriptions SET "+change); err != nil {
			t.Fatal(err)
		}
		locked()
	}
	h.expect("GET", "/me/posts/"+p.ID, "", 404)
	h.expect("PUT", "/me/posts/"+p.ID, postBody("null"), 409)
	h.expect("PUT", "/me/profile", `{"nickname":"reader","displayName":"Другой автор","description":""}`, 200)
	h.expect("PUT", "/me/posts/"+p.ID, postBody("null"), 404)
	h.expect("DELETE", "/me/posts/"+p.ID, "", 404)
	h.expect("POST", "/me/posts/"+p.ID+"/publish", "", 404)
	uploadFile(t, h, p.ID, "attachment", "intrusion.txt", []byte("bad"), true, 404)
	h.expect("DELETE", "/me/posts/"+p.ID+"/media/"+picture.ID, "", 404)
	other := testTier(t, h.expect("POST", "/me/tiers", `{"name":"Чужой","description":"","priceKopecks":0}`, 201))
	fixtureSubscription(t, h, other.ID)
	locked()
	h.client, h.csrf = ownerClient, ownerCSRF
	h.expect("PUT", "/me/posts/"+p.ID, postBody(fmt.Sprintf("%q", other.ID)), 422)
	// Public lists never include full bodies or filenames, even for the owner.
	listing := h.expect("GET", "/authors/writer/posts", "", 200)
	if strings.Contains(listing, "Секретный текст") || strings.Contains(listing, "photo.png") {
		t.Fatal("list leaked content")
	}
	h.expect("POST", "/me/tiers/"+mid.ID+"/archive", "", 200)
	h.expect("PUT", "/me/posts/"+p.ID, postBody(fmt.Sprintf("%q", mid.ID)), 200)
	fixtureSubscription(t, h, mid.ID)
	h.client, h.csrf = readerClient, readerCSRF
	if !postResult(t, h.expect("GET", "/posts/"+p.ID, "", 200)).CanRead {
		t.Fatal("archived tier lost existing access")
	}
	h.client = &http.Client{}
	locked()
	h.client, h.csrf = ownerClient, ownerCSRF
	h.expect("PUT", "/me/posts/"+p.ID, postBody("null"), 200)
	h.client = &http.Client{}
	if !postResult(t, h.expect("GET", "/posts/"+p.ID, "", 200)).CanRead {
		t.Fatal("public post locked")
	}
	mediaRequest(t, h, "GET", audio.URL, "bytes=0-4", 206)
	if _, err := h.pool.Exec(t.Context(), "UPDATE users SET blocked=true WHERE nickname='writer'"); err != nil {
		t.Fatal(err)
	}
	h.expect("GET", "/posts/"+p.ID, "", 404)
	mediaRequest(t, h, "GET", picture.URL, "", 404)
	if _, err := h.pool.Exec(t.Context(), "UPDATE users SET blocked=false WHERE nickname='writer'"); err != nil {
		t.Fatal(err)
	}
	h.client, h.csrf = ownerClient, ownerCSRF
	h.expect("PUT", "/me/posts/"+p.ID, postBody(fmt.Sprintf("%q", high.ID)), 200)
	h.client = &http.Client{}
	locked()
	h.client, h.csrf = ownerClient, ownerCSRF
	h.expect("POST", "/me/posts/"+p.ID+"/unpublish", "", 200)
	h.client = &http.Client{}
	h.expect("GET", "/posts/"+p.ID, "", 404)
	mediaRequest(t, h, "GET", audio.URL, "", 404)
	h.client, h.csrf = ownerClient, ownerCSRF
	h.expect("DELETE", "/me/posts/"+p.ID+"/media/"+audio.ID, "", 200)
	mediaRequest(t, h, "GET", audio.URL, "", 404)
	h.expect("DELETE", "/me/posts/"+p.ID, "", 200)
	mediaRequest(t, h, "GET", picture.URL, "", 404)
	mediaRequest(t, h, "GET", *picture.ThumbnailURL, "", 404)
	if _, err := h.pool.Exec(t.Context(), "UPDATE post_media SET created_at=now()-interval '2 hours'"); err != nil {
		t.Fatal(err)
	}
	if err := media.NewCleaner(h.server.q, h.server.files, 128).RunCycle(t.Context()); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := h.pool.QueryRow(t.Context(), "SELECT count(*) FROM post_media").Scan(&remaining); err != nil || remaining != 0 {
		t.Fatal("unused uploads retained", remaining, err)
	}
	if _, err := h.server.files.Open(t.Context(), "does-not-exist"); err == nil {
		t.Fatal("missing object accessible")
	}
	for _, key := range storedKeys {
		if r, err := h.server.files.Open(t.Context(), key); err == nil {
			_ = r.Close()
			t.Fatal("unused original or thumbnail retained")
		}
	}
}

type zeroStream struct{}

func (zeroStream) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestIntegrationMediaSizeBoundary(t *testing.T) {
	if os.Getenv("TEST_S3_ENDPOINT") == "" {
		t.Skip("TEST_S3_ENDPOINT required for 100 MiB streaming test")
	}
	h := publicationHarness(t)
	p := postResult(t, h.expect("POST", "/me/posts", postBody("null"), 201))
	for _, sample := range []struct {
		size    int64
		chunked bool
		status  int
	}{{media.MaxFileBytes, false, 201}, {media.MaxFileBytes + 1, false, 413}, {media.MaxFileBytes + 1, true, 413}} {
		req, err := http.NewRequestWithContext(t.Context(), "POST", h.url+"/api/v1/me/posts/"+p.ID+"/media?kind=attachment&filename=large.bin", io.LimitReader(zeroStream{}, sample.size))
		if err != nil {
			t.Fatal(err)
		}
		if !sample.chunked {
			req.ContentLength = sample.size
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Origin", "http://example.test")
		req.Header.Set("X-CSRF-Token", h.csrf)
		resp, err := h.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != sample.status {
			t.Fatalf("size %d, chunked %v: %d %s", sample.size, sample.chunked, resp.StatusCode, body)
		}
	}
	var count int
	if err := h.pool.QueryRow(t.Context(), "SELECT count(*) FROM post_media WHERE state='ready'").Scan(&count); err != nil || count != 1 {
		t.Fatal("failed uploads were finalized", count, err)
	}
	h.expect("DELETE", "/me/posts/"+p.ID, "", 200)
	if _, err := h.pool.Exec(t.Context(), "UPDATE post_media SET created_at=now()-interval '2 hours'"); err != nil {
		t.Fatal(err)
	}
	if err := media.NewCleaner(h.server.q, h.server.files, 128).RunCycle(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationMediaCleanupRetriesAndReservations(t *testing.T) {
	h := publicationHarness(t)
	objects := &memoryFiles{data: make(map[string][]byte)}
	h.server.files = objects
	p := postResult(t, h.expect("POST", "/me/posts", postBody("null"), 201))
	f := uploadFile(t, h, p.ID, "attachment", "keep.txt", []byte("keep"), true, 201)
	var key string
	if err := h.pool.QueryRow(t.Context(), "SELECT object_key FROM post_media WHERE id=$1", mustID(t, f.ID)).Scan(&key); err != nil {
		t.Fatal(err)
	}
	objects.data["posts/orphan"] = []byte("orphan")
	if _, err := h.pool.Exec(t.Context(), `INSERT INTO post_media(author_id,post_id,kind,filename,content_type,size_bytes,object_key,created_at)
 VALUES(1,$1,'attachment','incomplete','application/octet-stream',1,'posts/incomplete',now()-interval '2 hours'),
 (1,$1,'attachment','recent','application/octet-stream',1,'posts/recent',now())`, mustID(t, p.ID)); err != nil {
		t.Fatal(err)
	}
	objects.data["posts/incomplete"] = []byte("partial")
	objects.data["posts/recent"] = []byte("in flight")
	cleaner := media.NewCleaner(h.server.q, objects, 2)
	objects.failDelete = true
	if err := cleaner.RunCycle(t.Context()); err == nil {
		t.Fatal("cleanup ignored storage failure")
	}
	objects.failDelete = false
	if err := cleaner.RunCycle(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(objects.data) != 2 || objects.data[key] == nil || objects.data["posts/recent"] == nil {
		t.Fatal("cleanup removed live objects or retained orphan", objects.data)
	}
	var count int
	if err := h.pool.QueryRow(t.Context(), "SELECT count(*) FROM post_media WHERE state='deleting'").Scan(&count); err != nil || count != 0 {
		t.Fatal("cleanup tombstones retained", err)
	}
	mediaRequest(t, h, "GET", f.URL, "", 200)
	// Requests without both exact Origin and cookie/header CSRF cannot mutate posts.
	for _, origin := range []string{"", "http://example.test.evil", "https://example.test", "http://example.test:80"} {
		req, err := http.NewRequestWithContext(t.Context(), "DELETE", h.url+"/api/v1/me/posts/"+p.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", origin)
		req.Header.Set("X-CSRF-Token", h.csrf)
		resp, err := h.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != 403 {
			t.Fatal("inexact Origin accepted", origin, resp.StatusCode)
		}
	}
	jar, _ := cookiejar.New(nil)
	h.client = &http.Client{Jar: jar}
	h.expect("GET", "/me/posts", "", 401)
}

func TestIntegrationMediaCleanupResumesSweep(t *testing.T) {
	h := setup(t)
	newTestSession(t, h, "writer")
	h.expect("PUT", "/me/profile", `{"nickname":"writer","displayName":"Автор","description":""}`, 200)
	post := postResult(t, h.expect("POST", "/me/posts", postBody("null"), 201))
	postID := mustID(t, post.ID)
	if _, err := h.pool.Exec(t.Context(), `
INSERT INTO post_media(author_id,post_id,kind,filename,content_type,size_bytes,object_key,thumbnail_key,state)
VALUES (1,$1,'attachment','a','text/plain',1,'posts/a','','ready'),
       (1,$1,'attachment','b','text/plain',1,'posts/b','','ready'),
       (1,$1,'attachment','c','text/plain',1,'posts/c','','ready'),
       (1,$1,'attachment','d','text/plain',1,'posts/d','','ready'),
       (1,$1,'image','e','image/png',1,'posts/e','posts/f','ready')`, postID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.pool.Exec(t.Context(), `
INSERT INTO profile_images(user_id,kind,object_key,content_type,size_bytes,width,height,sha256)
VALUES (1,'avatar','profiles/a','image/png',1,1,1,repeat('a',64))`); err != nil {
		t.Fatal(err)
	}
	protected := []string{"posts/a", "posts/b", "posts/c", "posts/d", "posts/e", "posts/f",
		"posts/g-fresh", "posts/h-fresh", "profiles/a", "unmanaged/orphan"}
	objects := &memoryFiles{
		data: map[string][]byte{
			"posts/i-retry":     {},
			"posts/j-orphan":    {},
			"profiles/b-orphan": {},
		},
		modified: map[string]time.Time{"posts/g-fresh": time.Now(), "posts/h-fresh": time.Now()},
	}
	for _, key := range protected {
		objects.data[key] = []byte{}
	}
	assertKeys := func(extra ...string) {
		t.Helper()
		want := append(slices.Clone(protected), extra...)
		slices.Sort(want)
		if got := slices.Sorted(maps.Keys(objects.data)); !slices.Equal(got, want) {
			t.Fatalf("retained keys %v, want %v", got, want)
		}
	}

	cleaner := media.NewCleaner(h.server.q, objects, 2)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var canceledAt string
	objects.beforeList = func(_ string, after string) error {
		if after == "posts/b" {
			canceledAt = after
			cancel()
		}
		return nil
	}
	if err := cleaner.RunCycle(ctx); !errors.Is(err, context.Canceled) || canceledAt == "" {
		t.Fatalf("expected cancellation after the first page, got %v at %q", err, canceledAt)
	}
	assertKeys("posts/i-retry", "posts/j-orphan", "profiles/b-orphan")

	// This orphan is inserted behind the checkpoint and must wait for the next pass.
	objects.data["posts/0-earlier"] = []byte{}
	objects.failDeleteKey = "posts/i-retry"
	type cursor struct{ prefix, after string }
	var pages []cursor
	objects.beforeList = func(prefix, after string) error {
		pages = append(pages, cursor{prefix, after})
		return nil
	}
	if err := cleaner.RunCycle(t.Context()); err == nil {
		t.Fatal("expected a deletion failure")
	}
	wantPages := []cursor{
		{"posts/", "posts/b"}, {"posts/", "posts/d"}, {"posts/", "posts/f"}, {"posts/", "posts/h-fresh"},
		{"posts/", "posts/j-orphan"}, {"profiles/", ""}, {"profiles/", "profiles/b-orphan"},
	}
	if !slices.Equal(pages, wantPages) {
		t.Fatalf("listed pages %v, want %v", pages, wantPages)
	}
	assertKeys("posts/0-earlier", "posts/i-retry")

	objects.failDeleteKey = ""
	if err := cleaner.RunCycle(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertKeys()
}

type mediaReferenceFailure struct {
	store.DBTX
}

func (db mediaReferenceFailure) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	if strings.HasPrefix(query, "-- name: ReferencedMediaKeys ") {
		return nil, errMediaReferenceQuery
	}
	return db.DBTX.Query(ctx, query, args...)
}

var errMediaReferenceQuery = errors.New("reference query failed")

func TestIntegrationMediaCleanupUnclassifiedBatch(t *testing.T) {
	h := setup(t)
	objects := &memoryFiles{data: map[string][]byte{"posts/orphan": {}}}
	listErr := errors.New("listing interrupted")
	objects.beforeList = func(_, _ string) error { return listErr }
	cleaner := media.NewCleaner(h.server.q, objects, 2)
	if err := cleaner.RunCycle(t.Context()); !errors.Is(err, listErr) {
		t.Fatalf("expected listing error, got %v", err)
	}
	if _, ok := objects.data["posts/orphan"]; !ok {
		t.Fatal("removed an object after listing failed")
	}

	objects.beforeList = nil
	cleaner = media.NewCleaner(store.New(mediaReferenceFailure{DBTX: h.pool}), objects, 2)
	if err := cleaner.RunCycle(t.Context()); !errors.Is(err, errMediaReferenceQuery) {
		t.Fatalf("expected reference query error, got %v", err)
	}
	if _, ok := objects.data["posts/orphan"]; !ok {
		t.Fatal("removed an object after the reference query failed")
	}
}

func TestIntegrationMediaCleanupEmptyBucket(t *testing.T) {
	h := setup(t)
	objects := &memoryFiles{}
	var prefixes []string
	objects.beforeList = func(prefix, after string) error {
		if after != "" || len(prefixes) >= 2 {
			t.Fatalf("unexpected read at prefix %q after %q", prefix, after)
		}
		prefixes = append(prefixes, prefix)
		return nil
	}
	cleaner := media.NewCleaner(h.server.q, objects, 2)
	for range 2 {
		prefixes = nil
		if err := cleaner.RunCycle(t.Context()); err != nil {
			t.Fatal(err)
		}
		if want := []string{"posts/", "profiles/"}; !slices.Equal(prefixes, want) {
			t.Fatalf("listed prefixes %v, want %v", prefixes, want)
		}
	}
}
