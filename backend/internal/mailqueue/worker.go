package mailqueue

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"etheyl.local/backend/internal/config"
	"etheyl.local/backend/internal/observability"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mail "github.com/wneessen/go-mail"
)

type Sender interface {
	Send(context.Context, store.MailOutbox) error
}

type SMTP struct {
	client *mail.Client
	from   string
}

func NewSMTP(cfg config.Config) (*SMTP, error) {
	port, err := strconv.Atoi(cfg.SMTPPort)
	if err != nil {
		return nil, err
	}
	policy := mail.TLSMandatory
	if cfg.SMTPInsecure {
		policy = mail.NoTLS
	}
	opts := []mail.Option{mail.WithPort(port), mail.WithTLSPolicy(policy), mail.WithTimeout(10 * time.Second)}
	if cfg.SMTPUsername != "" {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(cfg.SMTPUsername), mail.WithPassword(cfg.SMTPPassword))
	}
	client, err := mail.NewClient(cfg.SMTPHost, opts...)
	if err != nil {
		return nil, err
	}
	return &SMTP{client: client, from: cfg.SMTPFrom}, nil
}
func (s *SMTP) Send(ctx context.Context, item store.MailOutbox) error {
	m := mail.NewMsg()
	if err := m.From(s.from); err != nil {
		return err
	}
	if err := m.To(item.Recipient); err != nil {
		return err
	}
	m.Subject(item.Subject)
	m.SetBodyString(mail.TypeTextPlain, item.Body)
	return s.client.DialAndSendWithContext(ctx, m)
}

// DeliverOne locks a row until SMTP delivery has finished. SMTP is at-least-once:
// a crash after delivery can repeat a mail, but its action token remains single-use.
func DeliverOne(ctx context.Context, pool *pgxpool.Pool, sender Sender, box *Cipher, logger *slog.Logger) (bool, error) {
	ctx, cancelOperation := context.WithTimeout(ctx, 20*time.Second)
	defer cancelOperation()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		rollbackCtx, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		_ = tx.Rollback(rollbackCtx)
	}()
	q := store.New(tx)
	item, err := q.LockMail(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	active, err := q.MailTokenActive(ctx, item.TokenHash)
	if err != nil {
		return true, err
	}
	if !active || !item.ExpiresAt.Time.After(time.Now()) {
		if err := q.DeleteMail(ctx, item.ID); err != nil {
			return true, err
		}
		return true, tx.Commit(ctx)
	}
	plain, err := box.Open(item.Body, Metadata(item.Recipient, item.Subject, item.TokenHash))
	if err != nil {
		logger.Error("mail permanently failed", "mail_id", item.ID, "category", "decrypt")
		if err := q.FailMail(ctx, item.ID); err != nil {
			return true, err
		}
		return true, tx.Commit(ctx)
	}
	item.Body = plain
	sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := sender.Send(sendCtx, item); err != nil {
		if item.Attempts >= 7 {
			logger.Error("mail permanently failed", "mail_id", item.ID, "attempt", item.Attempts+1, "category", "smtp")
		} else {
			logger.Warn("mail delivery deferred", "mail_id", item.ID, "attempt", item.Attempts+1, "category", "smtp")
		}
		if err := q.RetryMail(ctx, item.ID); err != nil {
			return true, err
		}
	} else {
		if err := q.DeleteMail(ctx, item.ID); err != nil {
			return true, err
		}
	}
	return true, tx.Commit(ctx)
}

func Run(ctx context.Context, pool *pgxpool.Pool, sender Sender, box *Cipher, logger *slog.Logger, monitor *Monitor) {
	cleanup := time.NewTicker(time.Minute)
	defer cleanup.Stop()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	q := store.New(pool)
	sender = monitoredSender{Sender: sender, monitor: monitor}
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			monitor.Beat()
			batchDeadline := time.Now().Add(10 * time.Second)
			for range 20 {
				if time.Now().After(batchDeadline) {
					break
				}
				monitor.Beat()
				found, err := DeliverOne(ctx, pool, sender, box, logger)
				if err != nil {
					if ctx.Err() == nil {
						observability.Failure(logger, "deliver_mail", err)
					}
					break
				}
				if !found {
					break
				}
			}
			monitor.Refresh(ctx, pool, logger)
		case <-cleanup.C:
			cleanupCtx, done := context.WithTimeout(ctx, 5*time.Second)
			operations := []struct {
				name string
				fn   func(context.Context) error
			}{
				{"cleanup_sessions", q.CleanupSessions}, {"cleanup_tokens", q.CleanupTokens},
				{"cleanup_rate_limits", q.CleanupRateLimits}, {"cleanup_mail", q.CleanupMail},
			}
			for _, op := range operations {
				if err := op.fn(cleanupCtx); err != nil && ctx.Err() == nil {
					observability.Failure(logger, op.name, err)
				}
			}
			done()
		}
	}
}
