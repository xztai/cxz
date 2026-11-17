package httpapi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/media"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostFile struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	Filename     string  `json:"filename"`
	ContentType  string  `json:"contentType"`
	SizeBytes    int64   `json:"sizeBytes"`
	URL          string  `json:"url"`
	ThumbnailURL *string `json:"thumbnailUrl"`
}

func fileDTO(m store.PostMedium) PostFile {
	id := strconv.FormatInt(m.ID, 10)
	result := PostFile{ID: id, Kind: m.Kind, Filename: m.Filename, ContentType: m.ContentType, SizeBytes: m.SizeBytes, URL: "/api/v1/media/" + id + "/original"}
	if m.ThumbnailKey != "" {
		result.ThumbnailURL = new("/api/v1/media/" + id + "/thumbnail")
	}
	return result
}

func (s *Server) uploadPostMedia(w http.ResponseWriter, r *http.Request) error {
	// Authorize before reading up to 100 MiB or allocating a temporary file.
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
	if s.files == nil {
		return problem(503, "storage_unavailable", "Загрузка файлов временно недоступна.")
	}
	kind := r.URL.Query().Get("kind")
	filename := strings.TrimSpace(r.URL.Query().Get("filename"))
	if (kind != "image" && kind != "audio" && kind != "video" && kind != "attachment") || utf8.RuneCountInString(filename) < 1 || utf8.RuneCountInString(filename) > 200 || strings.ContainsAny(filename, "/\\") || strings.ContainsFunc(filename, unicode.IsControl) {
		return invalid()
	}
	if r.ContentLength > media.MaxFileBytes {
		return problem(413, "file_too_large", "Размер файла не должен превышать 100 МБ.")
	}
	file, err := os.CreateTemp("", "etheyl-upload-*")
	if err != nil {
		return err
	}
	defer func() { _ = file.Close(); _ = os.Remove(file.Name()) }()
	size, err := io.Copy(file, r.Body)
	if err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return problem(413, "file_too_large", "Размер файла не должен превышать 100 МБ.")
		}
		return invalid()
	}
	if size < 1 || size > media.MaxFileBytes {
		return problem(422, "invalid_file", "Выберите непустой файл до 100 МБ.")
	}
	contentType, thumb, err := media.InspectFile(file, kind)
	if err != nil {
		return problem(422, "invalid_file", "Не удалось прочитать файл. Проверьте формат или загрузите его как вложение. Изображения: JPEG, PNG, GIF, WebP, до 8192 пикселей по стороне и 16 Мп.")
	}
	token, err := auth.Token()
	if err != nil {
		return err
	}
	key := fmt.Sprintf("posts/%d/%s", u.ID, token)
	thumbKey := ""
	if len(thumb) > 0 {
		thumbKey = key + "/thumbnail"
	}
	tx, q, author, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if _, err := q.OwnPost(r.Context(), store.OwnPostParams{ID: id, AuthorID: author}); errors.Is(err, pgx.ErrNoRows) {
		return missing()
	} else if err != nil {
		return err
	}
	row, err := q.ReservePostMedia(r.Context(), store.ReservePostMediaParams{AuthorID: author, PostID: pgtype.Int8{Int64: id, Valid: true}, Kind: kind, Filename: filename, ContentType: contentType, SizeBytes: size, ObjectKey: key, ThumbnailKey: thumbKey})
	if err != nil {
		return err
	}
	// Persist the reservation before S3 I/O. A crash leaves an expiring upload,
	// never an untracked object. No database transaction spans the upload.
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := s.files.Upload(r.Context(), key, file, size, contentType); err != nil {
		return err
	}
	if len(thumb) > 0 {
		if err := s.files.Upload(r.Context(), thumbKey, bytes.NewReader(thumb), int64(len(thumb)), "image/png"); err != nil {
			return err
		}
	}
	n, err := s.q.CompletePostMedia(r.Context(), store.CompletePostMediaParams{ID: row.ID, AuthorID: author})
	if err != nil {
		return err
	}
	if n == 0 {
		return missing()
	}
	return respond(w, 201, map[string]any{"media": fileDTO(row)})
}

func (s *Server) detachPostMedia(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	mediaID, err := pathID(r, "mediaID")
	if err != nil {
		return err
	}
	tx, q, author, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	n, err := q.DetachPostMedia(r.Context(), store.DetachPostMediaParams{ID: mediaID, PostID: pgtype.Int8{Int64: id, Valid: true}, AuthorID: author})
	if err != nil {
		return err
	}
	if n == 0 {
		return missing()
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 200, map[string]string{"message": "Файл удалён из публикации."})
}

func (s *Server) servePostMedia(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "mediaID")
	if err != nil {
		return err
	}
	variant := r.PathValue("variant")
	if variant != "original" && variant != "thumbnail" {
		return missing()
	}
	m, err := s.q.GetPostMedia(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		return missing()
	}
	if err != nil {
		return err
	}
	p, err := s.postAccess(r, m.PostID.Int64)
	if err != nil {
		return err
	}
	if !p.CanRead {
		return problem(403, "subscription_required", "Для чтения нужна подписка подходящего уровня.")
	}
	if s.files == nil {
		return problem(503, "storage_unavailable", "Файл временно недоступен.")
	}
	key, contentType := m.ObjectKey, m.ContentType
	disposition := "inline"
	if m.Kind == "attachment" {
		disposition = "attachment"
	}
	if variant == "thumbnail" {
		if m.ThumbnailKey == "" {
			return missing()
		}
		key, contentType = m.ThumbnailKey, "image/png"
	}
	reader, err := s.files.Open(r.Context(), key)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": m.Filename}))
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	// ServeContent checks Range/If-Range/HEAD only after authorization. No ETag
	// or modification date can turn a later unauthorized request into a 304.
	stream := &mediaResponse{ResponseWriter: w}
	http.ServeContent(stream, r, m.Filename, time.Time{}, reader)
	if stream.failure != 0 {
		w.Header().Del("Content-Length")
		switch stream.failure {
		case http.StatusRequestedRangeNotSatisfiable:
			return problem(416, "invalid_range", "Запрошенный диапазон файла недоступен.")
		case http.StatusPreconditionFailed:
			return problem(412, "precondition_failed", "Условия запроса не выполнены. Обновите страницу.")
		default:
			return problem(500, "internal", "Не удалось прочитать файл. Попробуйте позже.")
		}
	}
	return nil
}

// Translate ServeContent's plaintext errors into the stable Russian API format.
// Successful byte ranges still stream directly, without buffering originals.
type mediaResponse struct {
	http.ResponseWriter
	failure int
}

func (w *mediaResponse) WriteHeader(status int) {
	if status >= 400 {
		w.failure = status
		return
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *mediaResponse) Write(p []byte) (int, error) {
	if w.failure != 0 {
		return len(p), nil
	}
	return w.ResponseWriter.Write(p)
}
