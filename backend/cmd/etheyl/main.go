package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/config"
	"etheyl.local/backend/internal/httpapi"
	"etheyl.local/backend/internal/mailqueue"
	"etheyl.local/backend/internal/media"
	"etheyl.local/backend/internal/observability"
	"etheyl.local/backend/internal/store"
	"etheyl.local/backend/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"net"

	"golang.org/x/term"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		observability.Failure(logger, "command", err)
		os.Exit(1)
	}
}
func run(logger *slog.Logger) error {
	command := "api"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command == "healthcheck" {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://127.0.0.1:8081/api/v1/health")
		if err != nil {
			return errors.New("healthcheck failed")
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != 200 {
			return errors.New("healthcheck failed")
		}
		return nil
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	pool, err := openPool(ctx, cfg.DatabaseURL, command)
	if err != nil {
		return errors.New("invalid database configuration")
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return errors.New("database unavailable")
	}
	switch command {
	case "migrate":
		migrationCtx, done := context.WithTimeout(ctx, 5*time.Minute)
		defer done()
		db := stdlib.OpenDBFromPool(pool)
		defer func() { _ = db.Close() }()
		provider, err := migrations.Provider(db)
		if err != nil {
			return err
		}
		if _, err := provider.Up(migrationCtx); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		logger.Info("migrations applied")
		return nil
	case "storage-init":
		storage, err := media.New(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3Secure)
		if err != nil {
			return err
		}
		return storage.Initialize(ctx)
	case "admin":
		return createAdmin(ctx, pool, os.Args[2:])
	case "requeue-mail":
		flags := flag.NewFlagSet("requeue-mail", flag.ContinueOnError)
		id := flags.Int64("id", 0, "ID письма с окончательным отказом")
		if err := flags.Parse(os.Args[2:]); err != nil {
			return err
		}
		retryCtx, done := context.WithTimeout(ctx, 5*time.Second)
		defer done()
		n, err := store.New(pool).RequeueMail(retryCtx, *id)
		if err != nil {
			return err
		}
		if n != 1 {
			return errors.New("mail unavailable, expired or action no longer valid")
		}
		logger.Info("mail requeued", "mail_id", *id)
		return nil
	case "api", "worker":
		s, err := httpapi.New(pool, cfg, logger)
		if err != nil {
			return err
		}
		handler := s.Handler()
		var workers sync.WaitGroup
		if command == "worker" {
			sender, err := mailqueue.NewSMTP(cfg)
			if err != nil {
				return errors.New("invalid SMTP configuration")
			}
			box, err := mailqueue.NewCipher(cfg.OutboxKeys)
			if err != nil {
				return err
			}
			monitor := mailqueue.NewMonitor()
			workers.Go(func() { mailqueue.Run(ctx, pool, sender, box, logger, monitor) })
			full := handler
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/mail/status" && r.Method == http.MethodGet {
					mailqueue.Status(w, r, pool, monitor)
					return
				}
				if r.URL.Path != "/api/v1/health" {
					http.NotFound(w, r)
					return
				}
				full.ServeHTTP(w, r)
			})
		}
		server := httpServer(ctx, cfg.Addr, handler)
		workers.Go(func() {
			<-ctx.Done()
			shutdownCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
			defer done()
			_ = server.Shutdown(shutdownCtx)
		})
		logger.Info("service started", "service", command, "addr", cfg.Addr)
		err = server.ListenAndServe()
		cancel()
		workers.Wait()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	default:
		return errors.New("usage: etheyl api|worker|migrate|admin|requeue-mail|healthcheck")
	}
}

func createAdmin(ctx context.Context, pool *pgxpool.Pool, args []string) error {
	flags := flag.NewFlagSet("admin", flag.ContinueOnError)
	emailArg := flags.String("email", "", "email администратора")
	nicknameArg := flags.String("nickname", "", "уникальный никнейм: 3–30 символов, латиница, цифры, _ и -")
	stdin := flags.Bool("password-stdin", false, "читать пароль из stdin без интерактивного ввода")
	if err := flags.Parse(args); err != nil {
		return err
	}
	email, valid := auth.NormalizeEmail(*emailArg)
	nickname, validNickname := auth.NormalizeNickname(*nicknameArg)
	if !valid || !validNickname {
		return errors.New("provide a valid --email and --nickname")
	}
	var password string
	if *stdin {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return errors.New("cannot read password")
		}
		password = strings.TrimRight(line, "\r\n")
	} else {
		fmt.Fprint(os.Stderr, "Пароль (от 12 символов): ")
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return errors.New("use a terminal or --password-stdin")
		}
		password = string(value)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = store.New(pool).CreateUser(ctx, store.CreateUserParams{Email: email, Nickname: nickname, PasswordHash: hash, EmailVerified: true, IsAdmin: true})
	if err != nil {
		return errors.New("cannot create administrator; email or nickname may already exist")
	}
	_, err = fmt.Fprintln(os.Stdout, "Администратор создан.")
	return err
}

// Server-side limits complement handler cancellation, including clients that disconnect.
func openPool(ctx context.Context, url string, command string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 16
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	if command != "migrate" {
		cfg.ConnConfig.RuntimeParams["statement_timeout"] = "4000"
		cfg.ConnConfig.RuntimeParams["lock_timeout"] = "2000"
		cfg.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "10000"
		if command == "worker" {
			cfg.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "20000"
		}
	} else {
		cfg.ConnConfig.RuntimeParams["lock_timeout"] = "5000"
		cfg.ConnConfig.RuntimeParams["statement_timeout"] = "30000"
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

func httpServer(ctx context.Context, addr string, handler http.Handler) *http.Server {
	return &http.Server{Addr: addr, Handler: handler, BaseContext: func(net.Listener) context.Context { return ctx }, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
}
