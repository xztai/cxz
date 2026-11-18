package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/media"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
)

func validKind(kind string) bool { return kind == "avatar" || kind == "cover" }
func (s *Server) cleanupImage(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.images.Delete(ctx, key); err != nil {
		s.log.Error("image cleanup failed")
	}
}
func (s *Server) uploadImage(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireUser(r); err != nil {
		return err
	}
	kind := r.PathValue("kind")
	if !validKind(kind) {
		return missing()
	}
	if s.images == nil {
		return problem(503, "storage_unavailable", "Загрузка изображений временно недоступна.")
	}
	contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	img, err := media.Validate(r.Body, contentType)
	if err != nil {
		return problem(422, "invalid_image", "Выберите исправное изображение JPEG, PNG или WebP до 5 МБ, не более 8192 пикселей по стороне и 16 Мп.")
	}
	tx, q, id, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	old, err := q.ProfileImages(r.Context(), id)
	if err != nil {
		return err
	}
	token, err := auth.Token()
	if err != nil {
		return err
	}
	key := fmt.Sprintf("profiles/%d/%s/%s", id, kind, token)
	if err := s.images.Put(r.Context(), key, img); err != nil {
		s.cleanupImage(key)
		return err
	}
	if err := s.saveUploadedProfileImage(r.Context(), tx, store.SaveProfileImageParams{UserID: id, Kind: kind, ObjectKey: key, ContentType: img.ContentType, SizeBytes: int64(len(img.Data)), Width: int32(img.Width), Height: int32(img.Height), Sha256: img.SHA256}, old); err != nil {
		return err
	}
	return s.ownProfile(w, r)
}

func (s *Server) saveUploadedProfileImage(ctx context.Context, tx pgx.Tx, image store.SaveProfileImageParams, old []store.ProfileImage) error {
	if err := store.New(tx).SaveProfileImage(ctx, image); err != nil {
		s.cleanupImage(image.ObjectKey)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		// The commit may have succeeded. The worker checks references before deleting either object.
		return err
	}
	for _, previous := range old {
		if previous.Kind == image.Kind {
			s.cleanupImage(previous.ObjectKey)
		}
	}
	return nil
}

func (s *Server) removeImage(w http.ResponseWriter, r *http.Request) error {
	kind := r.PathValue("kind")
	if !validKind(kind) {
		return missing()
	}
	tx, q, id, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	old, err := q.ProfileImages(r.Context(), id)
	if err != nil {
		return err
	}
	if err := q.DeleteProfileImage(r.Context(), store.DeleteProfileImageParams{UserID: id, Kind: kind}); err != nil {
		return err
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	if s.images != nil {
		for _, previous := range old {
			if previous.Kind == kind {
				s.cleanupImage(previous.ObjectKey)
			}
		}
	}
	return s.ownProfile(w, r)
}
func (s *Server) publicImage(w http.ResponseWriter, r *http.Request) error {
	kind := r.PathValue("kind")
	if !validKind(kind) {
		return missing()
	}
	img, err := s.q.ProfileImage(r.Context(), store.ProfileImageParams{Nickname: r.PathValue("nickname"), Kind: kind})
	if errors.Is(err, pgx.ErrNoRows) {
		return missing()
	}
	if err != nil {
		return err
	}
	if s.images == nil {
		return problem(503, "storage_unavailable", "Изображение временно недоступно.")
	}
	reader, err := s.images.Get(r.Context(), img.ObjectKey)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	w.Header().Set("Content-Type", img.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(img.SizeBytes, 10))
	_, err = io.CopyN(w, reader, img.SizeBytes)
	return err
}
