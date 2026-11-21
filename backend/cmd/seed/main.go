package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"etheyl.local/backend/internal/media"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	emailSuffix = "@seed.etheyl.test"
	devPassword = "qweqweqweqweqwe"
	markerEmail = "seed-admin" + emailSuffix
)

type settings struct {
	db                           *pgx.ConnConfig
	origin, endpoint             string
	accessKey, secretKey, bucket string
	secure                       bool
}

func loopback(host string) bool {
	ip := net.ParseIP(host)
	return strings.EqualFold(host, "localhost") || ip != nil && ip.IsLoopback()
}

func production(value string) bool {
	for part := range strings.FieldsFuncSeq(strings.ToLower(value), func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == '/' || r == ' '
	}) {
		switch part {
		case "prod", "production", "prd", "live", "stage", "staging":
			return true
		}
	}
	return false
}

func loadSettings(getenv func(string) string) (settings, error) {
	if getenv("SEED_ALLOW") != "1" {
		return settings{}, errors.New("для запуска нужен SEED_ALLOW=1")
	}
	for _, name := range []string{"APP_ENV", "ENV", "ENVIRONMENT", "GO_ENV", "NODE_ENV", "DEPLOY_ENV", "RAILS_ENV", "APP_MODE", "STAGE"} {
		if production(getenv(name)) {
			return settings{}, fmt.Errorf("запуск запрещён в окружении %s", name)
		}
	}
	key := strings.TrimSpace(getenv("YOOKASSA_API_KEY"))
	if strings.EqualFold(getenv("COOKIE_SECURE"), "true") || key != "" && !strings.HasPrefix(key, "test_") && !strings.HasPrefix(key, "secret_test_") {
		return settings{}, errors.New("обнаружены признаки production")
	}
	origin := getenv("APP_ORIGIN")
	if origin == "" {
		origin = "http://localhost:8080"
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" || !loopback(u.Hostname()) || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return settings{}, errors.New("APP_ORIGIN должен быть локальным HTTP origin")
	}
	rawDSN := getenv("DATABASE_URL")
	dsn, err := url.Parse(rawDSN)
	if err != nil || (dsn.Scheme != "postgres" && dsn.Scheme != "postgresql") || !loopback(dsn.Hostname()) || production(dsn.Path) || dsn.Fragment != "" {
		return settings{}, errors.New("DATABASE_URL должен быть локальным PostgreSQL URL для dev-базы")
	}
	query, err := url.ParseQuery(dsn.RawQuery)
	if err != nil {
		return settings{}, errors.New("неверные параметры DATABASE_URL")
	}
	for name, values := range query {
		if name != "sslmode" || len(values) != 1 {
			return settings{}, errors.New("в DATABASE_URL разрешён только параметр sslmode")
		}
	}
	if getenv("PGSERVICE") != "" || getenv("PGHOSTADDR") != "" || getenv("PGSSLCERT") != "" || getenv("PGSSLKEY") != "" {
		return settings{}, errors.New("уберите настройки сервиса, адреса и клиентских сертификатов libpq")
	}
	db, err := pgx.ParseConfig(rawDSN)
	if err != nil || !loopback(db.Host) || production(db.Database) {
		return settings{}, errors.New("не удалось разобрать настройки локального Postgres")
	}
	for _, fallback := range db.Fallbacks {
		if !loopback(fallback.Host) {
			return settings{}, errors.New("резервный адрес Postgres должен быть локальным")
		}
	}
	db.RuntimeParams["search_path"] = "public, pg_catalog"
	db.RuntimeParams["timezone"] = "UTC"
	db.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		if err != nil || net.ParseIP(host) == nil || !loopback(host) {
			return nil, errors.New("фактический адрес Postgres должен быть loopback")
		}
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}
	endpoint := getenv("S3_ENDPOINT")
	if endpoint != "" {
		host, _, err := net.SplitHostPort(endpoint)
		if err != nil || !loopback(host) {
			return settings{}, errors.New("S3_ENDPOINT должен указывать на loopback")
		}
	}
	bucket := getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "etheyl-media"
	}
	if production(bucket) {
		return settings{}, errors.New("имя S3-бакета похоже на production")
	}
	return settings{db, origin, endpoint, getenv("S3_ACCESS_KEY"), getenv("S3_SECRET_KEY"), bucket, getenv("S3_SECURE") != "false"}, nil
}

// Only labels and SQLSTATE are printed. Driver and S3 errors may contain data.
type phaseError struct {
	phase string
	err   error
}

func (e phaseError) Error() string { return e.phase }
func (e phaseError) Unwrap() error { return e.err }

func main() {
	reset := flag.Bool("reset", false, "пересоздать только сидовые данные")
	dryRun := flag.Bool("dry-run", false, "показать действия без записи в Postgres и S3")
	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 8*time.Minute)
	defer cancel()
	cfg, err := loadSettings(os.Getenv)
	if err == nil {
		err = run(ctx, cfg, *reset, *dryRun)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Сидер:", err)
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			fmt.Fprintf(os.Stderr, "PostgreSQL SQLSTATE %s (%s %s)\n", pgErr.Code, pgErr.TableName, pgErr.ConstraintName)
		}
		os.Exit(1)
	}
}

type seeder struct {
	ctx    context.Context
	tx     pgx.Tx
	q      *store.Queries
	now    time.Time
	origin string
	s3     *media.S3
	users  map[string]int64
	tiers  map[string]store.SubscriptionTier
	posts  []int64
	keys   map[string]bool
}

func run(ctx context.Context, cfg settings, reset, dryRun bool) error {
	if !dryRun && os.Getenv("SEED_BILLING_WORKER_DISABLED") != "1" {
		return errors.New("отключите биллинг у фонового воркера и задайте SEED_BILLING_WORKER_DISABLED=1")
	}
	conn, err := pgx.ConnectConfig(ctx, cfg.db)
	if err != nil {
		return phaseError{"подключение к локальному Postgres", err}
	}
	defer func() { _ = conn.Close(context.Background()) }()
	if !dryRun {
		if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(733415, 1)"); err != nil {
			return phaseError{"блокировка параллельного запуска", err}
		}
	}
	options := pgx.TxOptions{}
	if dryRun {
		options.AccessMode = pgx.ReadOnly
	}
	tx, err := conn.BeginTx(ctx, options)
	if err != nil {
		return phaseError{"начало транзакции", err}
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	s := &seeder{
		ctx: ctx, tx: tx, q: store.New(tx), origin: cfg.origin,
		users: make(map[string]int64), tiers: make(map[string]store.SubscriptionTier), keys: make(map[string]bool),
	}
	var present bool
	if err := tx.QueryRow(ctx, `
		SELECT date_trunc('second', now()), EXISTS(SELECT 1 FROM users WHERE email=$1)
	`, markerEmail).Scan(&s.now, &present); err != nil {
		return phaseError{"проверка мигрированной базы и маркера сида", err}
	}
	s.now = s.now.UTC()
	ffmpeg, _ := exec.LookPath("ffmpeg")
	if dryRun {
		fmt.Printf("Dry run: %d аккаунта, 35 профилей, 15 именованных сценариев читателей. ffmpeg: %t.\n", len(accounts), ffmpeg != "")
		switch {
		case reset:
			fmt.Println("Действие: пересоздать данные домена " + emailSuffix)
		case present:
			fmt.Println("Действие: сохранить существующий сид")
		default:
			fmt.Println("Действие: создать сид")
		}
		return s.summary()
	}
	if present && !reset {
		fmt.Println("Сид уже существует. Записей в Postgres и S3 не меняли.")
		return s.summary()
	}
	if cfg.endpoint == "" || cfg.accessKey == "" || cfg.secretKey == "" {
		return errors.New("для создания сида нужны локальные S3_ENDPOINT, S3_ACCESS_KEY и S3_SECRET_KEY")
	}
	s.s3, err = media.New(cfg.endpoint, cfg.accessKey, cfg.secretKey, cfg.bucket, cfg.secure)
	if err != nil {
		return phaseError{"настройка локального S3", err}
	}
	if reset {
		if err := s.reset(); err != nil {
			return err
		}
	}
	for _, step := range []struct {
		label string
		do    func() error
	}{
		{"аккаунты и профили", s.createAccounts},
		{"уровни", s.createTiers},
		{"публикации", s.createPosts},
		{"платежи и подписки", s.createBilling},
		{"архивные уровни", s.archiveTiers},
		{"лайки и комментарии", s.createInteractions},
		{"загрузка медиа в локальный S3", func() error { return s.createMedia(ffmpeg) }},
	} {
		if err := step.do(); err != nil {
			return phaseError{step.label, err}
		}
	}
	if err := s.verify(); err != nil {
		return err
	}
	if err := s.summary(); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return phaseError{"фиксация транзакции", err}
	}
	if reset {
		if err := s.pruneMedia(conn); err != nil {
			return phaseError{"сид сохранён, очистку S3 можно повторить через --reset (" + err.Error() + ")", err}
		}
	}
	if ffmpeg == "" {
		fmt.Println("Видео пропущено: ffmpeg отсутствует.")
	}
	fmt.Println("Сид сохранён.")
	return nil
}

func instant(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

func nullableID(id int64) pgtype.Int8 {
	return pgtype.Int8{Int64: id, Valid: id != 0}
}
