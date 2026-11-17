package httpapi

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type feedResponse struct {
	Posts []struct {
		PostSummary
		CanRead bool `json:"canRead"`
	} `json:"posts"`
	Next *string `json:"nextCursor"`
}
type discussionResponse struct {
	Summary  InteractionSummary `json:"summary"`
	Comments []Comment          `json:"comments"`
	Next     *string            `json:"nextCursor"`
}

func discussionResult(t *testing.T, raw string) discussionResponse {
	t.Helper()
	var r discussionResponse
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	return r
}
func TestIntegrationSocialAccessIdempotenceAndOwnDeletion(t *testing.T) {
	f := billingHarness(t, false)
	h := f.h
	id := f.post.ID
	h.expect("GET", "/posts/"+id+"/discussion", "", 403)
	h.expect("PUT", "/posts/"+id+"/like", "", 403)
	h.expect("POST", "/posts/"+id+"/comments", `{"body":"закрытый комментарий"}`, 403)
	f.paid(t)
	h.expect("POST", "/posts/"+id+"/comments", `{"body":"   "}`, 422)
	h.expect("POST", "/posts/"+id+"/comments", `{"body":"a","userId":"1"}`, 422)
	h.expect("POST", "/posts/"+id+"/comments", `{"body":"\u0000"}`, 422)
	h.expect("POST", "/posts/"+id+"/comments", `{"body":"`+strings.Repeat("я", 2001)+`"}`, 422)
	status, _, _ := h.call("PUT", "/posts/"+id+"/like", "", false)
	if status != 403 {
		t.Fatal("like bypassed CSRF")
	}
	var wg sync.WaitGroup
	for range 6 {
		wg.Go(func() {
			code, _, _ := h.call("PUT", "/posts/"+id+"/like", "", true)
			if code != 200 {
				t.Error("concurrent like", code)
			}
		})
	}
	wg.Wait()
	d := discussionResult(t, h.expect("GET", "/posts/"+id+"/discussion", "", 200))
	if d.Summary.LikeCount != 1 || !d.Summary.Liked {
		t.Fatal("duplicate likes")
	}
	raw := h.expect("POST", "/posts/"+id+"/comments", `{"body":"<script>alert(1)</script>\nМой отзыв"}`, 201)
	var posted struct {
		Comment Comment `json:"comment"`
	}
	if err := json.Unmarshal([]byte(raw), &posted); err != nil {
		t.Fatal(err)
	}
	d = discussionResult(t, h.expect("GET", "/posts/"+id+"/discussion", "", 200))
	if len(d.Comments) != 1 || !d.Comments[0].Own || d.Comments[0].Nickname != "reader" || d.Summary.CommentCount != 1 {
		t.Fatal("wrong discussion DTO")
	}
	if strings.Contains(raw, "email") || strings.Contains(raw, "user_id") {
		t.Fatal("private commenter fields leaked")
	}
	h.expect("DELETE", "/posts/"+id+"/like", "", 200)
	h.expect("DELETE", "/posts/"+id+"/like", "", 200)
	readerClient, readerCSRF := h.client, h.csrf
	newTestSession(t, h, "other")
	h.expect("DELETE", "/posts/"+id+"/comments/"+posted.Comment.ID, "", 404)
	h.expect("GET", "/posts/"+id+"/discussion", "", 403)
	h.client, h.csrf = readerClient, readerCSRF
	if _, err := h.pool.Exec(t.Context(), "UPDATE subscriptions SET expires_at=now()-interval '1 second',starts_at=now()-interval '2 months'"); err != nil {
		t.Fatal(err)
	}
	h.expect("GET", "/posts/"+id+"/discussion", "", 403)
	h.expect("POST", "/posts/"+id+"/comments", `{"body":"new"}`, 403)
	h.expect("DELETE", "/posts/"+id+"/comments/"+posted.Comment.ID, "", 200)
	h.expect("DELETE", "/posts/"+id+"/comments/"+posted.Comment.ID, "", 404)
}
func TestIntegrationSocialPublicDraftAndCommentPagination(t *testing.T) {
	h := publicationHarness(t)
	post := postResult(t, h.expect("POST", "/me/posts", postBody("null"), 201))
	h.expect("GET", "/posts/"+post.ID+"/discussion", "", 404)
	h.expect("PUT", "/posts/"+post.ID+"/like", "", 404)
	h.expect("POST", "/me/posts/"+post.ID+"/publish", "", 200)
	writerClient, writerCSRF := h.client, h.csrf
	newTestSession(t, h, "visitor")
	h.expect("PUT", "/posts/"+post.ID+"/like", "", 200)
	if _, err := h.pool.Exec(t.Context(), "INSERT INTO post_comments(post_id,user_id,body) SELECT $1,2,'Текст '||i FROM generate_series(1,27) i", mustID(t, post.ID)); err != nil {
		t.Fatal(err)
	}
	d := discussionResult(t, h.expect("GET", "/posts/"+post.ID+"/discussion", "", 200))
	if len(d.Comments) != 24 || d.Next == nil || d.Summary.CommentCount != 27 {
		t.Fatal("first comment page")
	}
	next := discussionResult(t, h.expect("GET", "/posts/"+post.ID+"/discussion?before="+*d.Next, "", 200))
	if len(next.Comments) != 3 || next.Next != nil {
		t.Fatal("second comment page")
	}
	h.expect("GET", "/posts/"+post.ID+"/discussion?before=-1", "", 422)
	if _, err := h.pool.Exec(t.Context(), "UPDATE users SET blocked=true WHERE nickname='visitor'"); err != nil {
		t.Fatal(err)
	}
	h.client, h.csrf = writerClient, writerCSRF
	hidden := discussionResult(t, h.expect("GET", "/posts/"+post.ID+"/discussion", "", 200))
	if len(hidden.Comments) != 0 || hidden.Summary.CommentCount != 0 || hidden.Summary.LikeCount != 0 {
		t.Fatal("blocked interactions visible")
	}
}
func TestIntegrationSocialFeedPaginationAndGrantRules(t *testing.T) {
	f := billingHarness(t, false)
	h := f.h
	f.paid(t)
	if _, err := h.pool.Exec(t.Context(), "INSERT INTO posts(author_id,title,teaser,body_markdown,minimum_tier_id,published_at) SELECT 1,'Лента '||i,'Анонс','PRIVATE_BODY',$1,'2026-01-01T12:00:00Z' FROM generate_series(1,27) i", mustID(t, f.low.ID)); err != nil {
		t.Fatal(err)
	}
	if _, err := h.pool.Exec(t.Context(), "INSERT INTO posts(author_id,title,teaser,body_markdown,minimum_tier_id,published_at) VALUES(1,'Старший','Публично','TOP_SECRET',$1,now()),(1,'Черновик','','DRAFT',$1,NULL)", mustID(t, f.high.ID)); err != nil {
		t.Fatal(err)
	}
	newPost := func() feedResponse {
		raw := h.expect("GET", "/me/feed", "", 200)
		if strings.Contains(raw, "PRIVATE_BODY") || strings.Contains(raw, "TOP_SECRET") || strings.Contains(raw, "DRAFT") {
			t.Fatal("feed leaked bodies/draft")
		}
		var result feedResponse
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	page := newPost()
	if len(page.Posts) != 24 || page.Next == nil {
		t.Fatalf("feed page size: len=%d next=%v", len(page.Posts), page.Next)
	}
	if page.Posts[0].Title != "Старший" || page.Posts[0].CanRead {
		t.Fatal("higher tier teaser/access")
	}
	seen := map[string]bool{}
	for _, p := range page.Posts {
		seen[p.ID] = true
	}
	var next feedResponse
	if err := json.Unmarshal([]byte(h.expect("GET", "/me/feed?cursor="+url.QueryEscape(*page.Next), "", 200)), &next); err != nil {
		t.Fatal(err)
	}
	if len(next.Posts) != 5 || next.Next != nil {
		t.Fatal("feed next page", len(next.Posts))
	}
	for _, p := range next.Posts {
		if seen[p.ID] {
			t.Fatal("duplicate across timestamp ties")
		}
	}
	h.expect("GET", "/me/feed?cursor=invalid", "", 422)
	h.expect("POST", "/me/subscriptions/"+f.payment.AgreementID+"/cancel", "", 200)
	if len(newPost().Posts) == 0 {
		t.Fatal("canceled renewal hid paid feed")
	}
	for _, change := range []string{"UPDATE subscriptions SET revoked_at=now()", "UPDATE subscriptions SET revoked_at=NULL,starts_at=now()+interval '1 day',expires_at=now()+interval '2 days'", "UPDATE subscriptions SET starts_at=now()-interval '2 months',expires_at=now()-interval '1 second'"} {
		if _, err := h.pool.Exec(t.Context(), change); err != nil {
			t.Fatal(err)
		}
		if len(newPost().Posts) != 0 {
			t.Fatal("feed accepted inactive grant")
		}
	}
}
func TestAnalyticsPeriodUTCAndBounds(t *testing.T) {
	now := time.Date(2026, 2, 1, 1, 0, 0, 0, time.FixedZone("plus3", 3*3600))
	r := httpreq(t, "/me/analytics")
	from, to, err := analyticsPeriod(r, now)
	if err != nil || from.Format(time.DateOnly) != "2026-01-02" || to.Format(time.DateOnly) != "2026-02-01" {
		t.Fatal(from, to, err)
	}
	for _, q := range []string{"from=2026-02-30", "from=2026-02-02&to=2026-02-01", "from=2024-01-01&to=2025-01-01", "from=1999-01-01&to=1999-01-01"} {
		if _, _, err = analyticsPeriod(httpreq(t, "/me/analytics?"+q), now); err == nil {
			t.Fatal("invalid period", q)
		}
	}
}
func TestIntegrationSocialFeedOnlySubscribedAuthors(t *testing.T) {
	h := publicationHarness(t)
	h.expect("POST", "/me/posts", postBody("null"), 201)
	newTestSession(t, h, "reader")
	h.expect("GET", "/me/analytics", "", 409)
	raw := h.expect("GET", "/me/feed", "", 200)
	if !strings.Contains(raw, `"posts":[]`) {
		t.Fatal("unsubscribed author appeared", raw)
	}
	h.expect("POST", "/auth/logout", "", 200)
	h.expect("GET", "/me/feed", "", 401)
	h.expect("GET", "/me/analytics", "", 401)
	h.expect("GET", "/posts/1/discussion", "", 401)
	h.expect("PUT", "/posts/1/like", "", 401)
}
func httpreq(t *testing.T, path string) *http.Request {
	t.Helper()
	return httptest.NewRequest("GET", path, nil)
}
