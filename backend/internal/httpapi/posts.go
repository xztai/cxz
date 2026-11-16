package httpapi

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yuin/goldmark"
)

type PostSummary struct {
	ID             string     `json:"id"`
	AuthorNickname string     `json:"authorNickname"`
	Title          string     `json:"title"`
	Teaser         string     `json:"teaser"`
	MinimumTierID  *string    `json:"minimumTierId"`
	PublishedAt    *time.Time `json:"publishedAt"`
}

type Post struct {
	PostSummary
	CanRead      bool       `json:"canRead"`
	BodyMarkdown *string    `json:"bodyMarkdown,omitempty"`
	BodyHTML     *string    `json:"bodyHtml,omitempty"`
	Media        []PostFile `json:"media,omitempty"`
}

func postSummary(id int64, nickname, title, teaser string, tier pgtype.Int8, published pgtype.Timestamptz) PostSummary {
	p := PostSummary{ID: strconv.FormatInt(id, 10), AuthorNickname: nickname, Title: title, Teaser: teaser}
	if tier.Valid {
		p.MinimumTierID = new(strconv.FormatInt(tier.Int64, 10))
	}
	if published.Valid {
		p.PublishedAt = new(published.Time.UTC())
	}
	return p
}

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id < 1 {
		return 0, missing()
	}
	return id, nil
}

func postOffset(r *http.Request) (int32, error) {
	raw := r.URL.Query().Get("offset")
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 0 || n > 100000 {
		return 0, invalid()
	}
	return int32(n), nil
}

type postInput struct {
	Title         string  `json:"title"`
	Teaser        string  `json:"teaser"`
	BodyMarkdown  string  `json:"bodyMarkdown"`
	MinimumTierID *string `json:"minimumTierId"`
}

func readPost(r *http.Request) (postInput, pgtype.Int8, error) {
	wire, err := decode[struct {
		Title         *string        `json:"title"`
		Teaser        *string        `json:"teaser"`
		BodyMarkdown  *string        `json:"bodyMarkdown"`
		MinimumTierID jsontext.Value `json:"minimumTierId"`
	}](r)
	var in postInput
	var tier pgtype.Int8
	if err != nil {
		return in, tier, err
	}
	if wire.Title == nil || wire.Teaser == nil || wire.BodyMarkdown == nil || len(wire.MinimumTierID) == 0 {
		return in, tier, invalid()
	}
	in.Title, in.Teaser, in.BodyMarkdown = *wire.Title, *wire.Teaser, *wire.BodyMarkdown
	if string(bytes.TrimSpace(wire.MinimumTierID)) != "null" {
		var id string
		if err := json.Unmarshal(wire.MinimumTierID, &id); err != nil {
			return in, tier, invalid()
		}
		in.MinimumTierID = new(id)
	}
	in.Title = strings.TrimSpace(in.Title)
	if utf8.RuneCountInString(in.Title) < 1 || utf8.RuneCountInString(in.Title) > 200 || utf8.RuneCountInString(in.Teaser) > 2000 || utf8.RuneCountInString(in.BodyMarkdown) > 100000 {
		return in, tier, invalid()
	}
	if in.MinimumTierID != nil {
		id, err := strconv.ParseInt(*in.MinimumTierID, 10, 64)
		if err != nil || id < 1 {
			return in, tier, invalid()
		}
		tier = pgtype.Int8{Int64: id, Valid: true}
	}
	return in, tier, nil
}

func validatePostTier(ctx context.Context, q *store.Queries, author int64, tier, previous pgtype.Int8) error {
	if !tier.Valid {
		return nil
	}
	tiers, err := q.ListTiers(ctx, author)
	if err != nil {
		return err
	}
	for _, row := range tiers {
		if row.ID == tier.Int64 && (!row.ArchivedAt.Valid || previous == tier) {
			return nil
		}
	}
	return problem(422, "invalid_tier", "Выберите активный уровень этого автора.")
}

func (s *Server) createPost(w http.ResponseWriter, r *http.Request) error {
	in, tier, err := readPost(r)
	if err != nil {
		return err
	}
	tx, q, author, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if err := validatePostTier(r.Context(), q, author, tier, pgtype.Int8{}); err != nil {
		return err
	}
	p, err := q.CreatePost(r.Context(), store.CreatePostParams{AuthorID: author, Title: in.Title, Teaser: in.Teaser, BodyMarkdown: in.BodyMarkdown, MinimumTierID: tier})
	if err != nil {
		return err
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	result, err := s.postFor(r, p.ID)
	if err != nil {
		return err
	}
	return respond(w, 201, map[string]any{"post": result})
}

func (s *Server) updatePost(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	in, tier, err := readPost(r)
	if err != nil {
		return err
	}
	tx, q, author, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	old, err := q.OwnPost(r.Context(), store.OwnPostParams{ID: id, AuthorID: author})
	if errors.Is(err, pgx.ErrNoRows) {
		return missing()
	}
	if err != nil {
		return err
	}
	if err := validatePostTier(r.Context(), q, author, tier, old.MinimumTierID); err != nil {
		return err
	}
	if _, err := q.UpdatePost(r.Context(), store.UpdatePostParams{ID: id, AuthorID: author, Title: in.Title, Teaser: in.Teaser, BodyMarkdown: in.BodyMarkdown, MinimumTierID: tier}); err != nil {
		return err
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	return s.getPost(w, r)
}

func (s *Server) publishPost(w http.ResponseWriter, r *http.Request) error {
	return s.changePublication(w, r, true)
}
func (s *Server) unpublishPost(w http.ResponseWriter, r *http.Request) error {
	return s.changePublication(w, r, false)
}
func (s *Server) changePublication(w http.ResponseWriter, r *http.Request, publish bool) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	tx, q, author, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	n, err := q.PublishPost(r.Context(), store.PublishPostParams{ID: id, AuthorID: author, Publish: publish})
	if err != nil {
		return err
	}
	if n == 0 {
		return missing()
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	return s.getPost(w, r)
}

func (s *Server) deletePost(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	tx, q, author, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	n, err := q.DeletePost(r.Context(), store.DeletePostParams{ID: id, AuthorID: author})
	if err != nil {
		return err
	}
	if n == 0 {
		return missing()
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 200, map[string]string{"message": "Публикация удалена."})
}

func (s *Server) viewerID(r *http.Request) (int64, error) {
	u, err := s.currentUser(r)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return u.ID, err
}

func (s *Server) postAccess(r *http.Request, id int64) (store.PostAccessRow, error) {
	viewer, err := s.viewerID(r)
	if err != nil {
		return store.PostAccessRow{}, err
	}
	p, err := s.q.PostAccess(r.Context(), store.PostAccessParams{ID: id, ViewerID: viewer})
	if errors.Is(err, pgx.ErrNoRows) {
		return p, missing()
	}
	return p, err
}

// goldmark's default renderer omits raw HTML and rejects dangerous link schemes.
// Never enable html.WithUnsafe here: this HTML is rendered by the reader UI.
var markdown = goldmark.New()

func (s *Server) postFor(r *http.Request, id int64) (Post, error) {
	p, err := s.postAccess(r, id)
	if err != nil {
		return Post{}, err
	}
	result := Post{PostSummary: postSummary(p.ID, p.Nickname, p.Title, p.Teaser, p.MinimumTierID, p.PublishedAt), CanRead: p.CanRead}
	if !p.CanRead {
		return result, nil
	}
	var html bytes.Buffer
	if err := markdown.Convert([]byte(p.BodyMarkdown), &html); err != nil {
		return Post{}, err
	}
	result.BodyMarkdown = new(p.BodyMarkdown)
	result.BodyHTML = new(html.String())
	rows, err := s.q.ListPostMedia(r.Context(), pgtype.Int8{Int64: p.ID, Valid: true})
	if err != nil {
		return Post{}, err
	}
	for _, m := range rows {
		result.Media = append(result.Media, fileDTO(m))
	}
	return result, nil
}

func (s *Server) getPost(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	p, err := s.postFor(r, id)
	if err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"post": p})
}

func (s *Server) ownPost(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.q.OwnPost(r.Context(), store.OwnPostParams{ID: id, AuthorID: u.ID}); errors.Is(err, pgx.ErrNoRows) {
		return missing()
	} else if err != nil {
		return err
	}
	return s.getPost(w, r)
}

func (s *Server) ownPosts(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	offset, err := postOffset(r)
	if err != nil {
		return err
	}
	rows, err := s.q.ListOwnPosts(r.Context(), store.ListOwnPostsParams{AuthorID: u.ID, Offset: offset})
	if err != nil {
		return err
	}
	posts := make([]PostSummary, 0, min(len(rows), 24))
	for _, p := range rows[:min(len(rows), 24)] {
		posts = append(posts, postSummary(p.ID, u.Nickname, p.Title, p.Teaser, p.MinimumTierID, p.PublishedAt))
	}
	return respond(w, 200, map[string]any{"posts": posts, "hasMore": len(rows) > 24})
}

func (s *Server) authorPosts(w http.ResponseWriter, r *http.Request) error {
	p, err := s.q.PublicCreator(r.Context(), r.PathValue("nickname"))
	if errors.Is(err, pgx.ErrNoRows) {
		return missing()
	}
	if err != nil {
		return err
	}
	offset, err := postOffset(r)
	if err != nil {
		return err
	}
	rows, err := s.q.ListPublishedPosts(r.Context(), store.ListPublishedPostsParams{AuthorID: p.UserID, Offset: offset})
	if err != nil {
		return err
	}
	posts := make([]PostSummary, 0, min(len(rows), 24))
	for _, row := range rows[:min(len(rows), 24)] {
		posts = append(posts, postSummary(row.ID, p.Nickname, row.Title, row.Teaser, row.MinimumTierID, row.PublishedAt))
	}
	return respond(w, 200, map[string]any{"posts": posts, "hasMore": len(rows) > 24})
}
