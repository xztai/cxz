package billing

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/config"
	"etheyl.local/backend/internal/mailqueue"
	"etheyl.local/backend/internal/observability"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Error struct{ Code, Message string }

func (e *Error) Error() string           { return e.Code }
func problem(code, message string) error { return &Error{code, message} }
func ts(t time.Time) pgtype.Timestamptz  { return pgtype.Timestamptz{Time: t, Valid: true} }
func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

type Service struct {
	pool     *pgxpool.Pool
	q        *store.Queries
	cfg      config.Config
	provider Provider
	box      *mailqueue.Cipher
}

func New(pool *pgxpool.Pool, cfg config.Config) (*Service, error) {
	return NewWithProvider(pool, cfg, NewClient(cfg.YooKassaShopID, cfg.YooKassaAPIKey))
}

func NewWithProvider(pool *pgxpool.Pool, cfg config.Config, provider Provider) (*Service, error) {
	box, err := mailqueue.NewCipher(cfg.OutboxKeys)
	if err != nil {
		return nil, err
	}
	return &Service{pool: pool, q: store.New(pool), cfg: cfg, provider: provider, box: box}, nil
}
func (s *Service) Enabled() bool { return s.cfg.YooKassaShopID != "" && s.cfg.YooKassaAPIKey != "" }
func (s *Service) newPayment(ctx context.Context, q *store.Queries, a store.BillingAgreement, tier, price int64, renewal, save bool) (store.Payment, error) {
	id, err := auth.Token()
	if err != nil {
		return store.Payment{}, err
	}
	req := CreateRequest{Amount: Amount{money(price), "RUB"}, Capture: true, SaveMethod: save, Description: "Тестовая подписка etheyl", Metadata: map[string]string{"etheyl_payment_id": id, "etheyl_agreement_id": strconv.FormatInt(a.ID, 10)}}
	if renewal {
		req.MethodID = a.PaymentMethodID
		req.SaveMethod = false
	} else {
		req.Confirmation = &Confirmation{Type: "redirect", ReturnURL: s.cfg.Origin + "/account/payments/" + id}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return store.Payment{}, err
	}
	return q.InsertPayment(ctx, store.InsertPaymentParams{ID: id, AgreementID: a.ID, TierID: tier, AmountKopecks: price, Renewal: renewal, SaveMethod: save, RequestBody: string(body)})
}
func (s *Service) Checkout(ctx context.Context, user, tier, price int64, renew bool) (store.Payment, error) {
	if !s.Enabled() {
		return store.Payment{}, problem("payments_disabled", "Оплата пока не настроена.")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return store.Payment{}, err
	}
	defer rollback(tx)
	q := store.New(tx)
	t, err := q.BillingTier(ctx, tier)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Payment{}, problem("tier_unavailable", "Уровень недоступен.")
	}
	if err != nil {
		return store.Payment{}, err
	}
	if t.AuthorID == user {
		return store.Payment{}, problem("own_subscription", "Нельзя подписаться на себя.")
	}
	if _, err = q.LockCreator(ctx, t.AuthorID); err != nil {
		return store.Payment{}, err
	}
	// Re-read after the author lock: price and archival may have changed.
	t, err = q.BillingTier(ctx, tier)
	if err != nil {
		return store.Payment{}, problem("tier_unavailable", "Уровень недоступен.")
	}
	if price != t.PriceKopecks {
		return store.Payment{}, problem("price_changed", "Цена изменилась. Обновите страницу и подтвердите актуальную цену.")
	}
	if price < 100 {
		return store.Payment{}, problem("invalid_price", "Минимальная сумма оплаты — 1 ₽.")
	}
	u, err := q.LockUser(ctx, user)
	if err != nil {
		return store.Payment{}, err
	}
	if u.Blocked || !u.EmailVerified {
		return store.Payment{}, problem("forbidden", "Аккаунт недоступен.")
	}
	a, err := q.CreateAgreement(ctx, store.CreateAgreementParams{SubscriberID: user, AuthorID: t.AuthorID, TierID: t.ID, PriceKopecks: price})
	if err != nil {
		return store.Payment{}, err
	}
	if p, e := q.OpenPayment(ctx, a.ID); e == nil {
		if p.Renewal || p.TierID != t.ID || p.AmountKopecks != price || p.SaveMethod != renew {
			return store.Payment{}, problem("checkout_terms_conflict", "Есть незавершённый платёж с другими условиями.")
		}
		return p, tx.Commit(ctx)
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return store.Payment{}, e
	}
	if a.PaidUntil.Valid && a.PaidUntil.Time.After(time.Now()) {
		return store.Payment{}, problem("subscription_active", "Подписка уже оплачена. Сменить уровень можно в управлении подписками.")
	}
	if err = q.PrepareAgreement(ctx, store.PrepareAgreementParams{ID: a.ID, TierID: t.ID, PriceKopecks: price, AutoRenew: renew}); err != nil {
		return store.Payment{}, err
	}
	if _, err = q.MarkTierUsed(ctx, store.MarkTierUsedParams{ID: t.ID, AuthorID: t.AuthorID}); err != nil {
		return store.Payment{}, err
	}
	p, err := s.newPayment(ctx, q, a, t.ID, price, false, renew)
	if err != nil {
		return p, err
	}
	return p, tx.Commit(ctx)
}

// All billing mutations lock author -> agreement -> payment. Provider calls run outside transactions.
func (s *Service) lockAgreement(ctx context.Context, id, user int64) (pgx.Tx, *store.Queries, store.BillingAgreement, error) {
	var a store.BillingAgreement
	// Owner lookup is read-only; revalidate after locking.
	authorID, err := s.q.AgreementAuthor(ctx, id)
	a.AuthorID = authorID
	if err != nil {
		return nil, nil, a, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, a, err
	}
	q := store.New(tx)
	if _, err = q.LockCreator(ctx, a.AuthorID); err != nil {
		rollback(tx)
		return nil, nil, a, err
	}
	a, err = q.LockAgreement(ctx, id)
	if err != nil {
		rollback(tx)
		return nil, nil, a, err
	}
	if user != 0 && a.SubscriberID != user {
		rollback(tx)
		return nil, nil, a, pgx.ErrNoRows
	}
	return tx, q, a, nil
}
func (s *Service) Cancel(ctx context.Context, id, user int64) error {
	tx, q, _, err := s.lockAgreement(ctx, id, user)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err = q.StopAgreement(ctx, store.StopAgreementParams{ID: id, StoppedReason: "canceled"}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Service) ChangeTier(ctx context.Context, id, user, tier, price int64) error {
	tx, q, a, err := s.lockAgreement(ctx, id, user)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if !a.AutoRenew || !a.PaidUntil.Valid || !a.PaidUntil.Time.After(time.Now()) || a.PaymentMethodID == "" {
		return problem("subscription_inactive", "Смена уровня доступна для оплаченной подписки с автопродлением.")
	}
	if _, err = q.OpenPayment(ctx, a.ID); err == nil {
		return problem("payment_in_progress", "Дождитесь завершения текущей оплаты.")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	t, err := q.BillingTier(ctx, tier)
	if err != nil || t.AuthorID != a.AuthorID {
		return problem("tier_unavailable", "Уровень недоступен.")
	}
	if price != t.PriceKopecks {
		return problem("price_changed", "Цена изменилась. Обновите страницу и подтвердите актуальную цену.")
	}
	if price < 100 {
		return problem("invalid_price", "Минимальная сумма оплаты — 1 ₽.")
	}
	if _, err = q.MarkTierUsed(ctx, store.MarkTierUsedParams{ID: t.ID, AuthorID: a.AuthorID}); err != nil {
		return err
	}
	if err = q.ChangeAgreementTier(ctx, store.ChangeAgreementTierParams{ID: id, NextTierID: pgtype.Int8{Int64: t.ID, Valid: true}, NextPriceKopecks: pgtype.Int8{Int64: price, Valid: true}}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Service) Schedule(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	ids, err := s.q.DueAgreementIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err = s.scheduleOne(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
func (s *Service) scheduleOne(ctx context.Context, id int64) error {
	tx, q, a, err := s.lockAgreement(ctx, id, 0)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if !a.AutoRenew || !a.PaidUntil.Valid || a.PaidUntil.Time.After(time.Now()) || a.PaymentMethodID == "" {
		return nil
	}
	if _, err = q.OpenPayment(ctx, id); err == nil {
		return nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	tier, price := a.TierID, a.PriceKopecks
	if a.NextTierID.Valid {
		tier, price = a.NextTierID.Int64, a.NextPriceKopecks.Int64
	}
	// Both the current and scheduled tiers must remain available.
	if _, err = q.BillingTier(ctx, a.TierID); err != nil {
		return err
	}
	if _, err = q.BillingTier(ctx, tier); err != nil {
		return err
	}
	allowed, err := q.BillingUserActive(ctx, a.SubscriberID)
	if err != nil {
		return err
	}
	if !allowed.Bool {
		return nil
	}
	if _, err = s.newPayment(ctx, q, a, tier, price, true, true); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Service) ProcessOne(ctx context.Context) (bool, error) {
	if !s.Enabled() {
		return false, nil
	}
	p, err := s.q.ClaimPayment(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, s.process(ctx, p)
}

// Sending is serialized with cancellation/archival. A send already authorized before
// cancellation may settle later; it must be reconciled instead of silently discarded.
func (s *Service) prepareSend(ctx context.Context, p store.Payment) (store.Payment, bool, error) {
	tx, q, a, err := s.lockAgreement(ctx, p.AgreementID, 0)
	if err != nil {
		return p, false, err
	}
	defer rollback(tx)
	p, err = q.LockPayment(ctx, p.ID)
	if err != nil {
		return p, false, err
	}
	if p.Status == "succeeded" || p.Status == "canceled" || p.Status == "review" {
		return p, false, nil
	}
	if p.FirstSentAt.Valid {
		return p, true, tx.Commit(ctx)
	}
	if p.Renewal {
		allowed, e := q.BillingUserActive(ctx, a.SubscriberID)
		if e != nil {
			return p, false, e
		}
		_, tierErr := q.BillingTier(ctx, p.TierID)
		if tierErr != nil && !errors.Is(tierErr, pgx.ErrNoRows) {
			return p, false, tierErr
		}
		if !a.AutoRenew || !allowed.Bool || tierErr != nil {
			if err = q.FinishPayment(ctx, store.FinishPaymentParams{ID: p.ID, Status: "canceled", FailureCode: "renewal_stopped"}); err != nil {
				return p, false, err
			}
			return p, false, tx.Commit(ctx)
		}
	}
	p, err = q.StartPaymentSend(ctx, p.ID)
	if err != nil {
		return p, false, err
	}
	return p, true, tx.Commit(ctx)
}
func (s *Service) process(ctx context.Context, p store.Payment) error {
	if p.ProviderID == "" {
		prepared, proceed, err := s.prepareSend(ctx, p)
		if err != nil || !proceed {
			return err
		}
		p = prepared
	}
	var remote RemotePayment
	var err error
	if p.ProviderID == "" {
		// YooKassa remembers idempotence keys for 24h. Never POST after that window.
		if time.Since(p.FirstSentAt.Time) >= 23*time.Hour {
			return s.q.ReviewPayment(ctx, store.ReviewPaymentParams{ID: p.ID, FailureCode: "unknown_result"})
		}
		remote, err = s.provider.Create(ctx, p.ID, p.RequestBody)
		if err == nil {
			if !s.matches(p, remote) {
				return s.q.ReviewPayment(ctx, store.ReviewPaymentParams{ID: p.ID, FailureCode: "provider_mismatch"})
			}
			if err = s.q.UpdateProviderPayment(ctx, store.UpdateProviderPaymentParams{ID: p.ID, ProviderID: remote.ID, ConfirmationUrl: safeConfirmation(remote.Confirmation.URL)}); err != nil {
				return err
			}
			p.ProviderID = remote.ID
		} else if _, ok := errors.AsType[*Rejected](err); ok {
			return s.reject(ctx, p, "provider_rejected")
		}
	}
	if err == nil {
		remote, err = s.provider.Get(ctx, p.ProviderID)
	}
	if err != nil {
		return s.q.RetryPayment(ctx, store.RetryPaymentParams{ID: p.ID, FailureCode: "provider_unavailable"})
	}
	_, err = s.apply(ctx, p, remote)
	return err
}
func (s *Service) matches(p store.Payment, r RemotePayment) bool {
	return providerID.MatchString(r.ID) && (p.ProviderID == "" || p.ProviderID == r.ID) && r.Test && r.Recipient.AccountID == s.cfg.YooKassaShopID && r.Amount.Currency == "RUB" && r.Amount.Value == money(p.AmountKopecks) && r.Metadata["etheyl_payment_id"] == p.ID && r.Metadata["etheyl_agreement_id"] == strconv.FormatInt(p.AgreementID, 10)
}

// Webhooks carry only a hint; authoritative payment data is fetched with shop credentials.
func (s *Service) Webhook(ctx context.Context, id string) error {
	_, err := s.Reconcile(ctx, id)
	return err
}

// ReconciliationOutcome describes the result of a verified provider lookup.
type ReconciliationOutcome string

const (
	ReconciliationUnmatched        ReconciliationOutcome = "unmatched"
	ReconciliationSucceeded        ReconciliationOutcome = "succeeded"
	ReconciliationCanceled         ReconciliationOutcome = "canceled"
	ReconciliationPending          ReconciliationOutcome = "pending"
	ReconciliationReview           ReconciliationOutcome = "review"
	ReconciliationAlreadySucceeded ReconciliationOutcome = "already_succeeded"
	ReconciliationAlreadyCanceled  ReconciliationOutcome = "already_canceled"
)

// Reconcile verifies an existing provider payment and applies it to its local attempt.
func (s *Service) Reconcile(ctx context.Context, id string) (ReconciliationOutcome, error) {
	if !s.Enabled() {
		return "", ErrProvider
	}
	remote, err := s.provider.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if remote.ID != id {
		return "", ErrProvider
	}
	p, err := s.q.PaymentByID(ctx, remote.Metadata["etheyl_payment_id"])
	if errors.Is(err, pgx.ErrNoRows) {
		return ReconciliationUnmatched, nil
	}
	if err != nil {
		return "", err
	}
	return s.apply(ctx, p, remote)
}
func (s *Service) apply(ctx context.Context, p store.Payment, r RemotePayment) (ReconciliationOutcome, error) {
	tx, q, a, err := s.lockAgreement(ctx, p.AgreementID, 0)
	if err != nil {
		return "", err
	}
	defer rollback(tx)
	p, err = q.LockPayment(ctx, p.ID)
	if err != nil {
		return "", err
	}
	matched := s.matches(p, r)
	if p.Status == "succeeded" || p.Status == "canceled" {
		if p.ProviderID != r.ID || !matched {
			return ReconciliationReview, nil
		}
		if p.Status == "succeeded" {
			return ReconciliationAlreadySucceeded, nil
		}
		return ReconciliationAlreadyCanceled, nil
	}
	commit := func(outcome ReconciliationOutcome) (ReconciliationOutcome, error) {
		if err := tx.Commit(ctx); err != nil {
			return "", err
		}
		return outcome, nil
	}
	if !matched {
		if err = q.ReviewPayment(ctx, store.ReviewPaymentParams{ID: p.ID, FailureCode: "provider_mismatch"}); err != nil {
			return "", err
		}
		return commit(ReconciliationReview)
	}
	if r.Status == "canceled" {
		if err = s.finishFailure(ctx, q, a, p, r.ID, "payment_canceled"); err != nil {
			return "", err
		}
		return commit(ReconciliationCanceled)
	}
	if r.Status == "pending" {
		if err = q.UpdateProviderPayment(ctx, store.UpdateProviderPaymentParams{ID: p.ID, ProviderID: r.ID, ConfirmationUrl: safeConfirmation(r.Confirmation.URL)}); err != nil {
			return "", err
		}
		return commit(ReconciliationPending)
	}
	if r.Status != "succeeded" {
		if err = q.ReviewPayment(ctx, store.ReviewPaymentParams{ID: p.ID, FailureCode: "unexpected_status"}); err != nil {
			return "", err
		}
		return commit(ReconciliationReview)
	}
	if !r.Paid {
		return "", ErrProvider
	}
	start := time.Now().UTC()
	anchor := start.Day()
	if p.Renewal && a.PaidUntil.Valid {
		start = a.PaidUntil.Time
		anchor = int(a.AnchorDay)
		// After a long outage never bill repeatedly for already elapsed months.
		if !NextMonth(start, anchor).After(time.Now()) {
			start = time.Now().UTC()
			anchor = start.Day()
		}
	}
	end := NextMonth(start, anchor)
	method := ""
	if p.SaveMethod && r.Method.Saved {
		method = r.Method.ID
	}
	if err = q.CompleteAgreement(ctx, store.CompleteAgreementParams{ID: a.ID, TierID: p.TierID, PriceKopecks: p.AmountKopecks, PaidUntil: ts(end), AnchorDay: int32(anchor), MethodID: method}); err != nil {
		return "", err
	}
	if err = q.PutPaidAccess(ctx, store.PutPaidAccessParams{SubscriberID: a.SubscriberID, AuthorID: a.AuthorID, TierID: p.TierID, StartsAt: ts(start), ExpiresAt: ts(end)}); err != nil {
		return "", err
	}
	if err = q.FinishPayment(ctx, store.FinishPaymentParams{ID: p.ID, ProviderID: r.ID, Status: "succeeded", PeriodStart: ts(start), PeriodEnd: ts(end)}); err != nil {
		return "", err
	}
	return commit(ReconciliationSucceeded)
}
func (s *Service) finishFailure(ctx context.Context, q *store.Queries, a store.BillingAgreement, p store.Payment, provider, reason string) error {
	if err := q.FinishPayment(ctx, store.FinishPaymentParams{ID: p.ID, ProviderID: provider, Status: "canceled", FailureCode: reason}); err != nil {
		return err
	}
	if err := q.StopAgreement(ctx, store.StopAgreementParams{ID: a.ID, StoppedReason: reason}); err != nil {
		return err
	}
	// Same transaction as the terminal transition: duplicate events cannot queue duplicate notices.
	u, err := q.BillingRecipient(ctx, a.SubscriberID)
	if err != nil {
		return err
	}
	subject := "Оплата подписки не прошла — etheyl"
	body := "Оплата не прошла. Автопродление отключено, повторных автоматических списаний не будет. Оплаченный доступ сохраняется до конца периода. Повторить оплату вручную: " + s.cfg.Origin + "/account/subscriptions"
	return q.QueueBillingMail(ctx, store.QueueBillingMailParams{Recipient: u.Email, Subject: subject, Body: s.box.Seal(body, mailqueue.Metadata(u.Email, subject, "")), RecipientUserID: pgtype.Int8{Int64: u.ID, Valid: true}})
}
func (s *Service) reject(ctx context.Context, p store.Payment, reason string) error {
	tx, q, a, err := s.lockAgreement(ctx, p.AgreementID, 0)
	if err != nil {
		return err
	}
	defer rollback(tx)
	p, err = q.LockPayment(ctx, p.ID)
	if err != nil {
		return err
	}
	if p.Status == "succeeded" || p.Status == "canceled" {
		return nil
	}
	if err = s.finishFailure(ctx, q, a, p, p.ProviderID, reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Service) Run(ctx context.Context, log *slog.Logger) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if err := s.Schedule(ctx); err != nil && ctx.Err() == nil {
				observability.Failure(log, "schedule_billing", err)
			}
			for range 10 {
				found, err := s.ProcessOne(ctx)
				if err != nil && ctx.Err() == nil {
					observability.Failure(log, "process_payment", err)
				}
				if err != nil || !found {
					break
				}
			}
		}
	}
}
