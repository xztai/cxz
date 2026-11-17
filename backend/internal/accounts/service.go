package accounts

import (
	"context"
	"errors"
	"strconv"
	"time"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/config"
	"etheyl.local/backend/internal/mailqueue"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const SessionLifetime = 30 * 24 * time.Hour

type Service struct {
	pool          *pgxpool.Pool
	q             *store.Queries
	cfg           config.Config
	box           *mailqueue.Cipher
	dummyPassword string
	passwordSlots chan struct{}
}
type Session struct {
	User  store.User
	Token string
}
type Error struct{ Code, Message string }

func (e *Error) Error() string           { return e.Code }
func problem(code, message string) error { return &Error{code, message} }
func invalid() error {
	return problem("validation", "Проверьте заполненные поля.")
}
func invalidToken() error {
	return problem("invalid_token", "Ссылка недействительна или срок её действия истёк. Запросите новое письмо.")
}
func timestamp(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
func New(pool *pgxpool.Pool, cfg config.Config) (*Service, error) {
	box, err := mailqueue.NewCipher(cfg.OutboxKeys)
	if err != nil {
		return nil, err
	}
	dummy, err := auth.HashPassword("dummy-password-never-used-for-login")
	if err != nil {
		return nil, err
	}
	return &Service{pool: pool, q: store.New(pool), cfg: cfg, box: box, dummyPassword: dummy, passwordSlots: make(chan struct{}, 4)}, nil
}
func (s *Service) passwordSlot(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case s.passwordSlots <- struct{}{}:
		return func() { <-s.passwordSlots }, nil
	default:
		return nil, problem("busy", "Сервис занят. Попробуйте через несколько секунд.")
	}
}
func (s *Service) hashPassword(ctx context.Context, password string) (string, error) {
	release, err := s.passwordSlot(ctx)
	if err != nil {
		return "", err
	}
	defer release()
	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return hash, nil
}
func (s *Service) checkPassword(ctx context.Context, hash, password string) (bool, error) {
	release, err := s.passwordSlot(ctx)
	if err != nil {
		return false, err
	}
	defer release()
	matches := auth.CheckPassword(hash, password)
	return matches, ctx.Err()
}
func (s *Service) limit(ctx context.Context, key string, maxHits int32) error {
	hits, err := s.q.RateLimit(ctx, auth.Digest(key))
	if err != nil {
		return err
	}
	if hits > maxHits {
		return problem("rate_limit", "Слишком много попыток. Попробуйте через 15 минут.")
	}
	return nil
}
func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}
func (s *Service) Register(ctx context.Context, emailInput, nicknameInput, password, ip string) error {
	email, valid := auth.NormalizeEmail(emailInput)
	nickname, validNickname := auth.NormalizeNickname(nicknameInput)
	if !validNickname {
		return problem("invalid_nickname", "Никнейм должен содержать от 3 до 30 символов: латинские буквы, цифры, _ и -. Без точек и пробелов.")
	}
	if !valid || !auth.ValidPassword(password) {
		return invalid()
	}
	if err := s.limit(ctx, "register:"+ip, 5); err != nil {
		return err
	}
	hash, err := s.hashPassword(ctx, password)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	q := s.q.WithTx(tx)
	u, err := q.CreateUser(ctx, store.CreateUserParams{Email: email, Nickname: nickname, PasswordHash: hash})
	if err != nil {
		if pgerr, ok := errors.AsType[*pgconn.PgError](err); ok && pgerr.Code == "23505" {
			switch pgerr.ConstraintName {
			case "users_email_key":
				return problem("email_exists", "Этот email уже зарегистрирован. Войдите в аккаунт.")
			case "users_nickname_key":
				return problem("nickname_exists", "Этот никнейм уже занят. Выберите другой.")
			}
		}
		return err
	}
	if err := s.queueAction(ctx, q, u, "verify"); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
func (s *Service) Login(ctx context.Context, emailInput, password, ip, oldSession string) (Session, error) {
	email, valid := auth.NormalizeEmail(emailInput)
	if !valid || len(password) > 256 || password == "" {
		return Session{}, invalid()
	}
	key := "login:" + ip + ":" + email
	hits, err := s.q.FailedLoginHits(ctx, auth.Digest(key))
	if err != nil {
		return Session{}, err
	}
	if hits >= 10 {
		return Session{}, problem("rate_limit", "Слишком много попыток. Попробуйте через 15 минут.")
	}
	// Account risk adds a bounded delay, never a global account lockout.
	risk, err := s.q.FailedLoginHits(ctx, auth.Digest("risk:"+email))
	if err != nil {
		return Session{}, err
	}
	if risk >= 10 {
		select {
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
			return Session{}, ctx.Err()
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer rollback(tx)
	q := s.q.WithTx(tx)
	u, err := q.LockUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Session{}, err
	}
	hash := u.PasswordHash
	if errors.Is(err, pgx.ErrNoRows) {
		hash = s.dummyPassword
	}
	matches, err := s.checkPassword(ctx, hash, password)
	if err != nil {
		return Session{}, err
	}
	if !matches || u.ID == 0 || u.Blocked {
		if err := tx.Rollback(ctx); err != nil {
			return Session{}, err
		}
		if err := s.limit(ctx, key, 1000000); err != nil {
			return Session{}, err
		}
		if err := s.limit(ctx, "risk:"+email, 1000000); err != nil {
			return Session{}, err
		}
		return Session{}, problem("credentials", "Неверный email или пароль.")
	}
	if err := q.ClearRateLimit(ctx, auth.Digest(key)); err != nil {
		return Session{}, err
	}
	if !u.EmailVerified {
		return Session{}, problem("email_unverified", "Сначала подтвердите email. Можно запросить новое письмо.")
	}
	token, err := s.createSession(ctx, q, oldSession, u.ID)
	if err != nil {
		return Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return Session{User: u, Token: token}, nil
}

// The caller holds the user's row lock and sets the cookie only after committing.
func (s *Service) createSession(ctx context.Context, q *store.Queries, oldSession string, userID int64) (string, error) {
	token, err := auth.Token()
	if err != nil {
		return "", err
	}
	if err := q.CreateSession(ctx, store.CreateSessionParams{TokenHash: auth.Digest(token), UserID: userID, ExpiresAt: timestamp(time.Now().Add(SessionLifetime))}); err != nil {
		return "", err
	}
	if oldSession != "" {
		if err := q.DeleteSession(ctx, auth.Digest(oldSession)); err != nil {
			return "", err
		}
	}
	return token, nil
}
func (s *Service) EmailAction(ctx context.Context, emailInput, purpose, ip string) error {
	email, valid := auth.NormalizeEmail(emailInput)
	if !valid {
		return invalid()
	}
	if err := s.limit(ctx, "mail-ip:"+purpose+":"+ip+":"+email, 3); err != nil {
		return err
	}
	if err := s.limit(ctx, "mail:"+purpose+":"+email, 3); err != nil {
		if _, ok := errors.AsType[*Error](err); ok {
			return nil
		}
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	q := s.q.WithTx(tx)
	u, err := q.LockUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if u.Blocked || (purpose == "verify" && u.EmailVerified) || (purpose == "reset" && !u.EmailVerified) {
		return nil
	}
	if err := s.queueAction(ctx, q, u, purpose); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
func (s *Service) queueAction(ctx context.Context, q *store.Queries, u store.User, purpose string) error {
	token, err := auth.Token()
	if err != nil {
		return err
	}
	lifetime := 24 * time.Hour
	path := "/auth/verify"
	subject := "Подтвердите email — etheyl"
	if purpose == "reset" {
		lifetime = time.Hour
		path = "/auth/reset"
		subject = "Восстановление пароля — etheyl"
	}
	expires := timestamp(time.Now().Add(lifetime))
	if err := q.PutActionToken(ctx, store.PutActionTokenParams{TokenHash: auth.Digest(token), UserID: u.ID, Purpose: purpose, ExpiresAt: expires}); err != nil {
		return err
	}
	body := subject + "\n\n" + s.cfg.Origin + path + "#token=" + token + "\n\nСсылка действует " + strconv.Itoa(int(lifetime.Hours())) + " ч. и может быть использована один раз.\nЕсли вы не запрашивали это письмо, просто проигнорируйте его.\n\netheyl"
	encrypted := s.box.Seal(body, mailqueue.Metadata(u.Email, subject, auth.Digest(token)))
	return q.QueueMail(ctx, store.QueueMailParams{Recipient: u.Email, Subject: subject, Body: encrypted, TokenHash: auth.Digest(token), ExpiresAt: expires})
}

func (s *Service) Consume(ctx context.Context, token, purpose, password, oldSession, ip string) (Session, error) {
	if purpose != "verify" && purpose != "reset" {
		return Session{}, invalidToken()
	}
	if purpose == "reset" && !auth.ValidPassword(password) {
		return Session{}, invalid()
	}
	if !auth.ValidToken(token) {
		return Session{}, invalidToken()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer rollback(tx)
	q := s.q.WithTx(tx)
	params := store.FindActionTokenParams{TokenHash: auth.Digest(token), Purpose: purpose}
	t, err := q.FindActionToken(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, invalidToken()
	}
	if err != nil {
		return Session{}, err
	}
	u, err := q.LockUser(ctx, t.UserID)
	if err != nil {
		return Session{}, err
	}
	if u.Blocked {
		return Session{}, invalidToken()
	}
	if _, err := q.ConsumeActionToken(ctx, store.ConsumeActionTokenParams(params)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, invalidToken()
		}
		return Session{}, err
	}
	if purpose == "reset" {
		matches, err := s.checkPassword(ctx, u.PasswordHash, password)
		if err != nil {
			return Session{}, err
		}
		if matches {
			// Rollback restores the token so the user can submit a different password.
			return Session{}, problem("password_unchanged", "Новый пароль должен отличаться от текущего.")
		}
	}
	var sessionToken string
	if purpose == "verify" {
		err = q.VerifyEmail(ctx, u.ID)
		if err == nil {
			u.EmailVerified = true
			sessionToken, err = s.createSession(ctx, q, oldSession, u.ID)
		}
	} else {
		var hash string
		hash, err = s.hashPassword(ctx, password)
		if err == nil {
			err = q.UpdatePassword(ctx, store.UpdatePasswordParams{ID: u.ID, PasswordHash: hash})
		}
		if err == nil {
			err = q.RevokeSessions(ctx, u.ID)
		}
		if err == nil {
			err = q.ClearRateLimit(ctx, auth.Digest("login:"+ip+":"+u.Email))
		}
		if err == nil {
			err = q.ClearRateLimit(ctx, auth.Digest("risk:"+u.Email))
		}
	}
	if err != nil {
		return Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return Session{User: u, Token: sessionToken}, nil
}
