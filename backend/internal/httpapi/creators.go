package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Profile struct {
	ID          string         `json:"id"`
	Nickname    string         `json:"nickname"`
	DisplayName string         `json:"displayName"`
	Description string         `json:"description"`
	Avatar      *ImageMetadata `json:"avatar"`
	Cover       *ImageMetadata `json:"cover"`
}
type ImageMetadata struct {
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	Width       int32  `json:"width"`
	Height      int32  `json:"height"`
}

const expectedUserHeader = "X-Expected-User-ID"

func missing() error { return problem(404, "not_found", "Страница не найдена.") }
func (s *Server) requireUser(r *http.Request) (store.User, error) {
	u, err := s.currentUser(r)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, problem(401, "unauthorized", "Войдите в аккаунт.")
	}
	if err != nil {
		return u, err
	}
	if expected := r.Header.Values(expectedUserHeader); len(expected) > 0 {
		section, _, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/api/v1/me/"), "/")
		switch section {
		case "profile", "tiers", "posts", "analytics":
			if len(expected) != 1 || expected[0] != strconv.FormatInt(u.ID, 10) {
				return store.User{}, problem(http.StatusPreconditionFailed, "account_changed", "Аккаунт изменился. Откройте студию заново.")
			}
		}
	}
	return u, nil
}
func (s *Server) profile(ctx context.Context, id int64) (Profile, error) {
	p, err := s.q.CreatorProfile(ctx, id)
	if err != nil {
		return Profile{}, err
	}
	result := Profile{ID: strconv.FormatInt(id, 10), Nickname: p.Nickname, DisplayName: p.DisplayName, Description: p.Description}
	images, err := s.q.ProfileImages(ctx, id)
	if err != nil {
		return Profile{}, err
	}
	for _, img := range images {
		meta := publicImageMetadata(p.Nickname, img.Kind, img.Sha256, img.ContentType, img.SizeBytes, img.Width, img.Height)
		if img.Kind == "avatar" {
			result.Avatar = meta
		} else {
			result.Cover = meta
		}
	}
	return result, nil
}
func (s *Server) ownProfile(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	ownerID := strconv.FormatInt(u.ID, 10)
	p, err := s.profile(r.Context(), u.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return respond(w, 200, map[string]any{"ownerId": ownerID, "profile": nil})
	}
	if err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"ownerId": ownerID, "profile": p})
}

type profileInput struct {
	Nickname    string  `json:"nickname"`
	DisplayName string  `json:"displayName"`
	Description *string `json:"description"`
}

func (s *Server) saveProfile(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	in, err := decode[profileInput](r)
	if err != nil {
		return err
	}
	if in.Description == nil {
		return invalid()
	}
	nickname, ok := auth.NormalizeNickname(in.Nickname)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	description := strings.TrimSpace(*in.Description)
	if !ok || utf8.RuneCountInString(in.DisplayName) < 1 || utf8.RuneCountInString(in.DisplayName) > 80 || utf8.RuneCountInString(description) > 5000 {
		return invalid()
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	q := store.New(tx)
	locked, err := q.LockUser(r.Context(), u.ID)
	if err != nil {
		return err
	}
	if locked.Blocked || !locked.EmailVerified {
		return problem(403, "forbidden", "Недостаточно прав.")
	}
	if err := q.UpdateNickname(r.Context(), store.UpdateNicknameParams{ID: u.ID, Nickname: nickname}); err != nil {
		if pg, ok := errors.AsType[*pgconn.PgError](err); ok && pg.Code == "23505" {
			return problem(409, "nickname_exists", "Этот никнейм уже занят.")
		}
		return fmt.Errorf("update nickname: %w", err)
	}
	if _, err := q.SaveCreatorProfile(r.Context(), store.SaveCreatorProfileParams{UserID: u.ID, DisplayName: in.DisplayName, Description: description}); err != nil {
		return err
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	return s.ownProfile(w, r)
}
func (s *Server) catalog(w http.ResponseWriter, r *http.Request) error {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	offset := int64(0)
	var err error
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.ParseInt(raw, 10, 32)
	}
	if err != nil || offset < 0 || offset > 100000 || utf8.RuneCountInString(search) > 100 {
		return invalid()
	}
	rows, err := s.q.ListCreators(r.Context(), store.ListCreatorsParams{Limit: 25, Offset: int32(offset), Search: search})
	if err != nil {
		return err
	}
	profiles := make([]Profile, 0, min(len(rows), 24))
	for _, row := range rows[:min(len(rows), 24)] {
		p := Profile{
			ID:          strconv.FormatInt(row.UserID, 10),
			Nickname:    row.Nickname,
			DisplayName: row.DisplayName,
			Description: row.Description,
		}
		if row.AvatarSha256 != "" {
			p.Avatar = publicImageMetadata(row.Nickname, "avatar", row.AvatarSha256, row.AvatarContentType, row.AvatarSizeBytes, row.AvatarWidth, row.AvatarHeight)
		}
		if row.CoverSha256 != "" {
			p.Cover = publicImageMetadata(row.Nickname, "cover", row.CoverSha256, row.CoverContentType, row.CoverSizeBytes, row.CoverWidth, row.CoverHeight)
		}
		profiles = append(profiles, p)
	}
	return respond(w, 200, map[string]any{"authors": profiles, "hasMore": len(rows) > 24})
}
func publicImageMetadata(nickname, kind, sha256, contentType string, sizeBytes int64, width, height int32) *ImageMetadata {
	return &ImageMetadata{
		URL:         "/api/v1/authors/" + nickname + "/images/" + kind + "?v=" + sha256,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		Width:       width,
		Height:      height,
	}
}

func (s *Server) author(w http.ResponseWriter, r *http.Request) error {
	nickname, ok := auth.NormalizeNickname(r.PathValue("nickname"))
	if !ok {
		return missing()
	}
	row, err := s.q.PublicCreator(r.Context(), nickname)
	if errors.Is(err, pgx.ErrNoRows) {
		return missing()
	}
	if err != nil {
		return err
	}
	p, err := s.profile(r.Context(), row.UserID)
	if err != nil {
		return err
	}
	rows, err := s.q.ListTiers(r.Context(), row.UserID)
	if err != nil {
		return err
	}
	tiers := make([]Tier, 0, len(rows))
	for _, row := range rows {
		if !row.ArchivedAt.Valid {
			tiers = append(tiers, tierDTO(row))
		}
	}
	return respond(w, 200, map[string]any{"profile": p, "tiers": tiers})
}

// All creator mutations take locks in user -> creator order. Creation, reordering,
// usage marking and archival must share this order in future publication/payment flows.
func (s *Server) creatorTx(r *http.Request) (pgx.Tx, *store.Queries, int64, error) {
	u, err := s.requireUser(r)
	if err != nil {
		return nil, nil, 0, err
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		return nil, nil, 0, err
	}
	q := store.New(tx)
	locked, err := q.LockUser(r.Context(), u.ID)
	if err == nil && (locked.Blocked || !locked.EmailVerified) {
		err = problem(403, "forbidden", "Недостаточно прав.")
	}
	if err == nil {
		_, err = q.LockCreator(r.Context(), u.ID)
	}
	if err != nil {
		_ = tx.Rollback(r.Context())
		if errors.Is(err, pgx.ErrNoRows) {
			err = problem(409, "profile_required", "Сначала сохраните профиль автора.")
		}
		return nil, nil, 0, err
	}
	return tx, q, u.ID, nil
}
