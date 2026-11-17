package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json/v2"
	"errors"

	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"etheyl.local/backend/internal/accounts"
	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/billing"
	"etheyl.local/backend/internal/config"
	"etheyl.local/backend/internal/media"
	"etheyl.local/backend/internal/observability"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	billing  *billing.Service
	pool     *pgxpool.Pool
	q        *store.Queries
	cfg      config.Config
	log      *slog.Logger
	accounts *accounts.Service
	images   media.Storage
	files    media.Objects
}

type apiError struct {
	status        int
	code, message string
}

func (e *apiError) Error() string                    { return e.code }
func problem(status int, code, message string) error { return &apiError{status, code, message} }
func invalid() error {
	return problem(422, "validation", "Проверьте заполненные поля.")
}
func invalidToken() error {
	return problem(400, "invalid_token", "Ссылка недействительна или срок её действия истёк. Запросите новое письмо.")
}

type User struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Nickname      string `json:"nickname"`
	EmailVerified bool   `json:"emailVerified"`
	IsAdmin       bool   `json:"isAdmin"`
}

func publicUser(u store.User) User {
	return User{strconv.FormatInt(u.ID, 10), u.Email, u.Nickname, u.EmailVerified, u.IsAdmin}
}
func New(pool *pgxpool.Pool, cfg config.Config, logger *slog.Logger) (*Server, error) {
	service, err := accounts.New(pool, cfg)
	if err != nil {
		return nil, err
	}
	payments, err := billing.New(pool, cfg)
	if err != nil {
		return nil, err
	}
	s := &Server{billing: payments, pool: pool, q: store.New(pool), cfg: cfg, log: logger, accounts: service}
	if cfg.S3Endpoint != "" {
		var storage *media.S3
		storage, err = media.New(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3Secure)
		if err != nil {
			return nil, err
		}
		s.images, s.files = storage, storage
	}
	return s, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/me/feed", s.endpoint(s.feed))
	mux.HandleFunc("GET /api/v1/me/analytics", s.endpoint(s.analytics))
	mux.HandleFunc("GET /api/v1/posts/{id}/discussion", s.endpoint(s.discussion))
	mux.HandleFunc("PUT /api/v1/posts/{id}/like", s.endpoint(s.likePost))
	mux.HandleFunc("DELETE /api/v1/posts/{id}/like", s.endpoint(s.unlikePost))
	mux.HandleFunc("POST /api/v1/posts/{id}/comments", s.endpoint(s.createComment))
	mux.HandleFunc("DELETE /api/v1/posts/{id}/comments/{commentID}", s.endpoint(s.deleteComment))
	mux.HandleFunc("POST /api/v1/billing/checkout", s.endpoint(s.checkout))
	mux.HandleFunc("GET /api/v1/me/subscriptions", s.endpoint(s.ownSubscriptions))
	mux.HandleFunc("POST /api/v1/me/subscriptions/{id}/cancel", s.endpoint(s.cancelSubscription))
	mux.HandleFunc("PUT /api/v1/me/subscriptions/{id}/tier", s.endpoint(s.changeSubscriptionTier))
	mux.HandleFunc("GET /api/v1/me/payments", s.endpoint(s.ownPayments))
	mux.HandleFunc("GET /api/v1/me/payments/{id}", s.endpoint(s.getPayment))
	mux.HandleFunc("POST /api/v1/billing/yookassa/webhook", s.yooKassaWebhook)
	mux.HandleFunc("GET /api/v1/me/posts", s.endpoint(s.ownPosts))
	mux.HandleFunc("POST /api/v1/me/posts", s.endpoint(s.createPost))
	mux.HandleFunc("GET /api/v1/me/posts/{id}", s.endpoint(s.ownPost))
	mux.HandleFunc("PUT /api/v1/me/posts/{id}", s.endpoint(s.updatePost))
	mux.HandleFunc("DELETE /api/v1/me/posts/{id}", s.endpoint(s.deletePost))
	mux.HandleFunc("POST /api/v1/me/posts/{id}/publish", s.endpoint(s.publishPost))
	mux.HandleFunc("POST /api/v1/me/posts/{id}/unpublish", s.endpoint(s.unpublishPost))
	mux.HandleFunc("POST /api/v1/me/posts/{id}/media", s.endpoint(s.uploadPostMedia))
	mux.HandleFunc("DELETE /api/v1/me/posts/{id}/media/{mediaID}", s.endpoint(s.detachPostMedia))
	mux.HandleFunc("GET /api/v1/authors/{nickname}/posts", s.endpoint(s.authorPosts))
	mux.HandleFunc("GET /api/v1/posts/{id}", s.endpoint(s.getPost))
	mux.HandleFunc("GET /api/v1/media/{mediaID}/{variant}", s.endpoint(s.servePostMedia))
	mux.HandleFunc("GET /api/v1/me/profile", s.endpoint(s.ownProfile))
	mux.HandleFunc("PUT /api/v1/me/profile", s.endpoint(s.saveProfile))
	mux.HandleFunc("PUT /api/v1/me/profile/images/{kind}", s.endpoint(s.uploadImage))
	mux.HandleFunc("DELETE /api/v1/me/profile/images/{kind}", s.endpoint(s.removeImage))
	mux.HandleFunc("GET /api/v1/authors", s.endpoint(s.catalog))
	mux.HandleFunc("GET /api/v1/authors/{nickname}", s.endpoint(s.author))
	mux.HandleFunc("GET /api/v1/authors/{nickname}/images/{kind}", s.endpoint(s.publicImage))
	mux.HandleFunc("GET /api/v1/me/tiers", s.endpoint(s.ownTiers))
	mux.HandleFunc("POST /api/v1/me/tiers", s.endpoint(s.createTier))
	mux.HandleFunc("PUT /api/v1/me/tiers/order", s.endpoint(s.reorderTiers))
	mux.HandleFunc("PUT /api/v1/me/tiers/{id}", s.endpoint(s.updateTier))
	mux.HandleFunc("POST /api/v1/me/tiers/{id}/archive", s.endpoint(s.archiveTier))
	mux.HandleFunc("DELETE /api/v1/me/tiers/{id}", s.endpoint(s.deleteTier))
	mux.HandleFunc("GET /api/v1/health", s.endpoint(s.health))
	mux.HandleFunc("GET /api/v1/auth/csrf", s.endpoint(s.csrf))
	mux.HandleFunc("GET /api/v1/me", s.endpoint(s.me))
	mux.HandleFunc("GET /api/v1/admin/status", s.endpoint(s.adminStatus))
	mux.HandleFunc("POST /api/v1/auth/register", s.endpoint(s.register))
	mux.HandleFunc("POST /api/v1/auth/login", s.endpoint(s.login))
	mux.HandleFunc("POST /api/v1/auth/logout", s.endpoint(s.logout))
	mux.HandleFunc("POST /api/v1/auth/verify-email", s.endpoint(s.verify))
	mux.HandleFunc("POST /api/v1/auth/verify-email/preview", s.endpoint(s.previewVerification))
	mux.HandleFunc("POST /api/v1/auth/resend-verification", s.endpoint(s.resend))
	mux.HandleFunc("POST /api/v1/auth/forgot-password", s.endpoint(s.forgot))
	mux.HandleFunc("POST /api/v1/auth/reset-password", s.endpoint(s.reset))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		s.writeError(w, problem(404, "not_found", "Страница не найдена."))
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		id, _ := auth.Token()
		w.Header().Set("X-Request-ID", id)
		recorder := &responseStatus{ResponseWriter: w, status: 200}
		w = recorder
		timeout := s.cfg.RequestTimeout
		if timeout == 0 {
			timeout = 5 * time.Second
		}
		if mediaTransfer(r) {
			timeout = media.TransferTimeout
			_ = http.NewResponseController(w).SetWriteDeadline(started.Add(timeout))
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		r = r.WithContext(ctx)
		_ = http.NewResponseController(w).SetReadDeadline(started.Add(timeout))
		defer func() {
			s.log.Info("request completed", "request_id", id, "method", r.Method, "route", r.Pattern, "status", recorder.status, "duration_ms", time.Since(started).Milliseconds())
		}()
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Vary", "Cookie")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		defer func() {
			if recover() != nil {
				s.log.Error("request panic", "request_id", w.Header().Get("X-Request-ID"), "stack", observability.PanicStack())
				s.writeError(w, problem(500, "internal", "Не удалось выполнить запрос. Попробуйте позже."))
			}
		}()
		mux.ServeHTTP(w, r)
	})
}

type endpoint func(http.ResponseWriter, *http.Request) error

func (s *Server) endpoint(fn endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if err := s.checkCSRF(r); err != nil {
				s.writeError(w, err)
				return
			}
			if err := s.limit(r.Context(), "ip:"+s.clientIP(r), 60); err != nil {
				s.writeError(w, err)
				return
			}
			limit := int64(16 * 1024)
			if strings.HasPrefix(r.URL.Path, "/api/v1/posts/") && strings.HasSuffix(r.URL.Path, "/comments") {
				limit = 32 * 1024
			}
			if r.URL.Path == "/api/v1/me/profile" || strings.HasPrefix(r.URL.Path, "/api/v1/me/tiers") {
				// 5000 Unicode characters may occupy up to 60 KB in escaped JSON.
				limit = 64 * 1024
			}
			if strings.HasPrefix(r.URL.Path, "/api/v1/me/posts") {
				limit = 2 << 20
			}
			if r.Method == http.MethodPost && mediaTransfer(r) {
				limit = media.MaxFileBytes
			}
			if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/v1/me/profile/images/") {
				limit = media.MaxBytes
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		if err := fn(w, r); err != nil {
			s.writeError(w, err)
		}
	}
}

func mediaTransfer(r *http.Request) bool {
	return ((r.Method == http.MethodGet || r.Method == http.MethodHead) && strings.HasPrefix(r.URL.Path, "/api/v1/media/")) ||
		(r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v1/me/posts/") && strings.HasSuffix(r.URL.Path, "/media"))
}

func (s *Server) writeError(w http.ResponseWriter, err error) {
	if failure, ok := errors.AsType[*billing.Error](err); ok {
		status := 409
		switch failure.Code {
		case "payments_disabled":
			status = 503
		case "forbidden":
			status = 403
		case "tier_unavailable":
			status = 404
		case "invalid_price":
			status = 422
		}
		err = problem(status, failure.Code, failure.Message)
	}
	if failure, ok := errors.AsType[*accounts.Error](err); ok {
		status := map[string]int{"validation": 422, "invalid_nickname": 422, "email_exists": 409, "nickname_exists": 409, "credentials": 401, "email_unverified": 403, "rate_limit": 429, "busy": 503, "invalid_token": 400, "password_unchanged": 422}[failure.Code]
		if status == 0 {
			status = 500
		}
		err = problem(status, failure.Code, failure.Message)
	}
	e, ok := errors.AsType[*apiError](err)
	if !ok {
		fields := append([]any{"request_id", w.Header().Get("X-Request-ID")}, observability.ErrorFields(err)...)
		s.log.Error("request failed", fields...)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			e = &apiError{503, "unavailable", "Сервис временно недоступен. Попробуйте позже."}
		}
		if e == nil {
			e = &apiError{500, "internal", "Не удалось выполнить запрос. Попробуйте позже."}
		}
	}
	if e.status == 429 {
		w.Header().Set("Retry-After", "900")
	}
	_ = respond(w, e.status, map[string]any{"error": map[string]string{"code": e.code, "message": e.message}})
}
func respond(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.MarshalWrite(w, value)
}
func decode[T any](r *http.Request) (T, error) {
	var value T
	if media, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";"); media != "application/json" {
		return value, problem(415, "content_type", "Ожидается JSON.")
	}
	if err := json.UnmarshalRead(r.Body, &value, json.RejectUnknownMembers(true)); err != nil {
		return value, invalid()
	}
	return value, nil
}
func (s *Server) sessionCookieName() string {
	if s.cfg.SecureCookies {
		return "__Host-etheyl_session"
	}
	return "etheyl_session"
}
func (s *Server) csrfCookieName() string {
	if s.cfg.SecureCookies {
		return "__Host-etheyl_csrf"
	}
	return "etheyl_csrf"
}
func (s *Server) cookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: s.cfg.SecureCookies, SameSite: http.SameSiteStrictMode, MaxAge: maxAge})
}
func (s *Server) csrf(w http.ResponseWriter, r *http.Request) error {
	value := ""
	if c, err := r.Cookie(s.csrfCookieName()); err == nil && auth.ValidToken(c.Value) {
		value = c.Value
	}
	if value == "" {
		var err error
		value, err = auth.Token()
		if err != nil {
			return err
		}
	}
	s.cookie(w, s.csrfCookieName(), value, 3600)
	return respond(w, 200, map[string]string{"token": value})
}
func (s *Server) checkCSRF(r *http.Request) error {
	c, err := r.Cookie(s.csrfCookieName())
	header := r.Header.Get("X-CSRF-Token")
	if r.Header.Get("Origin") != s.cfg.Origin || err != nil || !auth.ValidToken(header) || subtle.ConstantTimeCompare([]byte(c.Value), []byte(header)) != 1 {
		return problem(403, "csrf", "Не удалось проверить запрос. Обновите страницу и попробуйте снова.")
	}
	return nil
}
func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		if ip := net.ParseIP(r.Header.Get("X-Real-IP")); ip != nil {
			return ip.String()
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
func (s *Server) limit(ctx context.Context, key string, maxHits int32) error {
	hits, err := s.q.RateLimit(ctx, auth.Digest(key))
	if err != nil {
		return err
	}
	if hits > maxHits {
		return problem(429, "rate_limit", "Слишком много попыток. Попробуйте через 15 минут.")
	}
	return nil
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.pool.Ping(ctx); err != nil {
		return problem(503, "unavailable", "База данных недоступна.")
	}
	return respond(w, 200, map[string]string{"status": "ok"})
}
func (s *Server) currentUser(r *http.Request) (store.User, error) {
	c, err := r.Cookie(s.sessionCookieName())
	if err != nil || !auth.ValidToken(c.Value) {
		return store.User{}, pgx.ErrNoRows
	}
	return s.q.SessionUser(r.Context(), auth.Digest(c.Value))
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) error {
	u, err := s.currentUser(r)
	if errors.Is(err, pgx.ErrNoRows) {
		return respond(w, 200, map[string]any{"user": nil})
	}
	if err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"user": publicUser(u)})
}
func (s *Server) adminStatus(w http.ResponseWriter, r *http.Request) error {
	u, err := s.currentUser(r)
	if errors.Is(err, pgx.ErrNoRows) {
		return problem(401, "unauthorized", "Войдите в аккаунт.")
	}
	if err != nil {
		return err
	}
	if !u.IsAdmin {
		return problem(403, "forbidden", "Недостаточно прав.")
	}
	return respond(w, 200, map[string]string{"status": "ok"})
}

type registerRequest struct {
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[registerRequest](r)
	if err != nil {
		return err
	}
	if err := s.accounts.Register(r.Context(), in.Email, in.Nickname, in.Password, s.clientIP(r)); err != nil {
		return err
	}
	return accepted(w)
}
func accepted(w http.ResponseWriter) error {
	return respond(w, 202, map[string]string{"message": "Если адрес подходит для этого действия, мы отправили письмо. Проверьте почту и папку «Спам»."})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) oldSession(r *http.Request) string {
	c, err := r.Cookie(s.sessionCookieName())
	if err != nil {
		return ""
	}
	return c.Value
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[loginRequest](r)
	if err != nil {
		return err
	}
	result, err := s.accounts.Login(r.Context(), in.Email, in.Password, s.clientIP(r), s.oldSession(r))
	if err != nil {
		return err
	}
	s.cookie(w, s.sessionCookieName(), result.Token, int(accounts.SessionLifetime.Seconds()))
	return respond(w, 200, map[string]any{"user": publicUser(result.User)})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) error {
	if c, err := r.Cookie(s.sessionCookieName()); err == nil {
		if err := s.q.DeleteSession(r.Context(), auth.Digest(c.Value)); err != nil {
			return err
		}
	}
	s.cookie(w, s.sessionCookieName(), "", -1)
	return respond(w, 200, map[string]string{"message": "Вы вышли из аккаунта."})
}

type emailRequest struct {
	Email string `json:"email"`
}

func (s *Server) resend(w http.ResponseWriter, r *http.Request) error {
	return s.emailAction(w, r, "verify")
}
func (s *Server) forgot(w http.ResponseWriter, r *http.Request) error {
	return s.emailAction(w, r, "reset")
}
func (s *Server) emailAction(w http.ResponseWriter, r *http.Request, purpose string) error {
	in, err := decode[emailRequest](r)
	if err != nil {
		return err
	}
	if err := s.accounts.EmailAction(r.Context(), in.Email, purpose, s.clientIP(r)); err != nil {
		return err
	}
	return accepted(w)
}

type tokenRequest struct {
	Token string `json:"token"`
}
type resetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (s *Server) previewVerification(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[tokenRequest](r)
	if err != nil {
		return err
	}
	if !auth.ValidToken(in.Token) {
		return invalidToken()
	}
	email, err := s.q.VerificationEmail(r.Context(), auth.Digest(in.Token))
	if errors.Is(err, pgx.ErrNoRows) {
		return invalidToken()
	}
	if err != nil {
		return err
	}
	return respond(w, 200, map[string]string{"email": email})
}

func (s *Server) verify(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[tokenRequest](r)
	if err != nil {
		return err
	}
	return s.consumeAction(w, r, in.Token, "verify", "")
}
func (s *Server) reset(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[resetRequest](r)
	if err != nil {
		return err
	}
	if !auth.ValidPassword(in.Password) {
		return invalid()
	}
	return s.consumeAction(w, r, in.Token, "reset", in.Password)
}
func (s *Server) consumeAction(w http.ResponseWriter, r *http.Request, token, purpose, password string) error {
	result, err := s.accounts.Consume(r.Context(), token, purpose, password, s.oldSession(r), s.clientIP(r))
	if err != nil {
		return err
	}
	if purpose == "verify" {
		s.cookie(w, s.sessionCookieName(), result.Token, int(accounts.SessionLifetime.Seconds()))
		return respond(w, 200, map[string]any{"user": publicUser(result.User)})
	}
	return respond(w, 200, map[string]string{"message": "Пароль изменён. Войдите с новым паролем."})
}

type responseStatus struct {
	http.ResponseWriter
	status  int
	written bool
}

func (w *responseStatus) WriteHeader(status int) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "Cookie")
	if !w.written {
		w.status = status
		w.written = true
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *responseStatus) Write(p []byte) (int, error) {
	if !w.written {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(p)
}
func (w *responseStatus) Unwrap() http.ResponseWriter { return w.ResponseWriter }
