package httpapi

import (
	"context"
	"encoding/base64"
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
)

type feedCursor struct {
	Time time.Time `json:"time"`
	ID   int64     `json:"id"`
}

func readFeedCursor(raw string) (feedCursor, error) {
	var c feedCursor
	if raw == "" {
		return c, nil
	}
	if len(raw) > 256 {
		return c, invalid()
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return c, invalid()
	}
	if err = json.Unmarshal(b, &c, json.RejectUnknownMembers(true)); err != nil || c.ID < 1 || c.Time.IsZero() {
		return c, invalid()
	}
	return c, nil
}
func (s *Server) feed(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	c, err := readFeedCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		return err
	}
	rows, err := s.q.ListFeed(r.Context(), store.ListFeedParams{ViewerID: u.ID, BeforeID: c.ID, BeforeTime: pgtype.Timestamptz{Time: c.Time, Valid: !c.Time.IsZero()}})
	if err != nil {
		return err
	}
	type item struct {
		PostSummary
		AuthorName string `json:"authorName"`
		CanRead    bool   `json:"canRead"`
	}
	items := make([]item, 0, min(24, len(rows)))
	for _, p := range rows[:min(24, len(rows))] {
		items = append(items, item{PostSummary: postSummary(p.ID, p.Nickname, p.Title, p.Teaser, p.MinimumTierID, p.PublishedAt), AuthorName: p.DisplayName, CanRead: p.CanRead})
	}
	var next *string
	if len(rows) > 24 {
		p := rows[23]
		b, err := json.Marshal(feedCursor{Time: p.PublishedAt.Time.UTC(), ID: p.ID})
		if err != nil {
			return err
		}
		next = new(base64.RawURLEncoding.EncodeToString(b))
	}
	return respond(w, 200, map[string]any{"posts": items, "nextCursor": next})
}

type InteractionSummary struct {
	LikeCount    int64 `json:"likeCount"`
	Liked        bool  `json:"liked"`
	CommentCount int64 `json:"commentCount"`
}

func summaryDTO(s store.InteractionSummaryRow) InteractionSummary {
	return InteractionSummary{s.LikeCount, s.Liked, s.CommentCount}
}

type Comment struct {
	ID        string    `json:"id"`
	PostID    string    `json:"postId"`
	Nickname  string    `json:"nickname"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	Own       bool      `json:"own"`
}

func socialRollback(tx pgx.Tx) {
	ctx, done := context.WithTimeout(context.Background(), time.Second)
	defer done()
	_ = tx.Rollback(ctx)
}
func noPostAccess() error {
	return problem(403, "post_access_required", "Для обсуждения нужен доступ к публикации.")
}
func (s *Server) interactionTx(r *http.Request) (pgx.Tx, *store.Queries, store.User, int64, error) {
	u, err := s.requireUser(r)
	if err != nil {
		return nil, nil, u, 0, err
	}
	id, err := pathID(r, "id")
	if err != nil {
		return nil, nil, u, 0, err
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		return nil, nil, u, 0, err
	}
	q := store.New(tx)
	u, err = q.LockUser(r.Context(), u.ID)
	if err == nil && (u.Blocked || !u.EmailVerified) {
		err = problem(401, "unauthorized", "Войдите в аккаунт.")
	}
	if err == nil {
		var p store.LockInteractionPostRow
		p, err = q.LockInteractionPost(r.Context(), store.LockInteractionPostParams{ID: id, ViewerID: u.ID})
		if errors.Is(err, pgx.ErrNoRows) {
			err = missing()
		} else if err == nil && !p.CanRead {
			err = noPostAccess()
		}
	}
	if err != nil {
		socialRollback(tx)
		return nil, nil, u, 0, err
	}
	return tx, q, u, id, nil
}
func (s *Server) discussion(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var before int64
	if raw := r.URL.Query().Get("before"); raw != "" {
		before, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || before < 1 {
			return invalid()
		}
	}
	tx, err := s.pool.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer socialRollback(tx)
	q := store.New(tx)
	p, err := q.PostAccess(r.Context(), store.PostAccessParams{ID: id, ViewerID: u.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return missing()
	}
	if err != nil {
		return err
	}
	if !p.PublishedAt.Valid {
		return missing()
	}
	if !p.CanRead {
		return noPostAccess()
	}
	summary, err := q.InteractionSummary(r.Context(), store.InteractionSummaryParams{PostID: id, ViewerID: u.ID})
	if err != nil {
		return err
	}
	rows, err := q.ListComments(r.Context(), store.ListCommentsParams{PostID: id, BeforeID: before})
	if err != nil {
		return err
	}
	comments := make([]Comment, 0, min(24, len(rows)))
	for _, c := range rows[:min(24, len(rows))] {
		comments = append(comments, Comment{ID: strconv.FormatInt(c.ID, 10), PostID: strconv.FormatInt(c.PostID, 10), Nickname: c.Nickname, Body: c.Body, CreatedAt: c.CreatedAt.Time.UTC(), Own: c.UserID == u.ID})
	}
	var next *string
	if len(rows) > 24 {
		next = new(strconv.FormatInt(rows[23].ID, 10))
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"summary": summaryDTO(summary), "comments": comments, "nextCursor": next})
}
func (s *Server) likePost(w http.ResponseWriter, r *http.Request) error {
	return s.changeLike(w, r, true)
}
func (s *Server) unlikePost(w http.ResponseWriter, r *http.Request) error {
	return s.changeLike(w, r, false)
}
func (s *Server) changeLike(w http.ResponseWriter, r *http.Request, like bool) error {
	tx, q, u, id, err := s.interactionTx(r)
	if err != nil {
		return err
	}
	defer socialRollback(tx)
	if like {
		err = q.LikePost(r.Context(), store.LikePostParams{PostID: id, UserID: u.ID})
	} else {
		err = q.UnlikePost(r.Context(), store.UnlikePostParams{PostID: id, UserID: u.ID})
	}
	if err != nil {
		return err
	}
	summary, err := q.InteractionSummary(r.Context(), store.InteractionSummaryParams{PostID: id, ViewerID: u.ID})
	if err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"summary": summaryDTO(summary)})
}
func (s *Server) createComment(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[struct {
		Body string `json:"body"`
	}](r)
	if err != nil {
		return err
	}
	in.Body = strings.TrimSpace(in.Body)
	if n := utf8.RuneCountInString(in.Body); n < 1 || n > 2000 || strings.ContainsRune(in.Body, 0) {
		return invalid()
	}
	tx, q, u, id, err := s.interactionTx(r)
	if err != nil {
		return err
	}
	defer socialRollback(tx)
	if err = s.limit(r.Context(), "comment:"+strconv.FormatInt(u.ID, 10), 20); err != nil {
		return err
	}
	c, err := q.CreateComment(r.Context(), store.CreateCommentParams{PostID: id, UserID: u.ID, Body: in.Body})
	if err != nil {
		return err
	}
	summary, err := q.InteractionSummary(r.Context(), store.InteractionSummaryParams{PostID: id, ViewerID: u.ID})
	if err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 201, map[string]any{"comment": Comment{ID: strconv.FormatInt(c.ID, 10), PostID: strconv.FormatInt(id, 10), Nickname: u.Nickname, Body: c.Body, CreatedAt: c.CreatedAt.Time.UTC(), Own: true}, "summary": summaryDTO(summary)})
}

// Users may remove their own text even after access expires or a post is unpublished.
// Deletion is owner-scoped and never discloses the post or comment content.
func (s *Server) deleteComment(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	comment, err := pathID(r, "commentID")
	if err != nil {
		return err
	}
	n, err := s.q.DeleteOwnComment(r.Context(), store.DeleteOwnCommentParams{ID: comment, PostID: id, UserID: u.ID})
	if err != nil {
		return err
	}
	if n == 0 {
		return missing()
	}
	return respond(w, 200, map[string]string{"message": "Комментарий удалён."})
}
