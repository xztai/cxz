package httpapi

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"strconv"
	"time"

	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Payment struct {
	ID              string  `json:"id"`
	AgreementID     string  `json:"agreementId"`
	Status          string  `json:"status"`
	AmountKopecks   int64   `json:"amountKopecks"`
	ConfirmationURL string  `json:"confirmationUrl"`
	Renewal         bool    `json:"renewal"`
	Test            bool    `json:"test"`
	CreatedAt       string  `json:"createdAt"`
	PeriodStart     *string `json:"periodStart"`
	PeriodEnd       *string `json:"periodEnd"`
	FailureCode     string  `json:"failureCode"`
}

func billingTime(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	return new(t.Time.UTC().Format(time.RFC3339))
}
func paymentDTO(p store.Payment) Payment {
	return Payment{ID: p.ID, AgreementID: strconv.FormatInt(p.AgreementID, 10), Status: p.Status, AmountKopecks: p.AmountKopecks, ConfirmationURL: p.ConfirmationUrl, Renewal: p.Renewal, Test: true, CreatedAt: p.CreatedAt.Time.UTC().Format(time.RFC3339), PeriodStart: billingTime(p.PeriodStart), PeriodEnd: billingTime(p.PeriodEnd), FailureCode: p.FailureCode}
}
func (s *Server) checkout(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	in, err := decode[struct {
		TierID    string `json:"tierId"`
		Price     *int64 `json:"priceKopecks"`
		AutoRenew *bool  `json:"autoRenew"`
	}](r)
	if err != nil {
		return err
	}
	id, err := strconv.ParseInt(in.TierID, 10, 64)
	if err != nil || id < 1 || in.Price == nil || in.AutoRenew == nil {
		return invalid()
	}
	p, err := s.billing.Checkout(r.Context(), u.ID, id, *in.Price, *in.AutoRenew)
	if err != nil {
		return err
	}
	return respond(w, 202, map[string]any{"payment": paymentDTO(p)})
}
func (s *Server) getPayment(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	a, err := s.q.PaymentOwner(r.Context(), r.PathValue("id"))
	if errors.Is(err, pgx.ErrNoRows) || err == nil && a.SubscriberID != u.ID {
		return missing()
	}
	if err != nil {
		return err
	}
	p, err := s.q.PaymentByID(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"payment": paymentDTO(p)})
}
func (s *Server) ownSubscriptions(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	offset, err := postOffset(r)
	if err != nil {
		return err
	}
	rows, err := s.q.ListAgreements(r.Context(), store.ListAgreementsParams{SubscriberID: u.ID, Offset: offset})
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, min(24, len(rows)))
	for _, a := range rows[:min(24, len(rows))] {
		var nextID *string
		if a.NextTierID.Valid {
			nextID = new(strconv.FormatInt(a.NextTierID.Int64, 10))
		}
		var nextPrice *int64
		if a.NextPriceKopecks.Valid {
			nextPrice = new(a.NextPriceKopecks.Int64)
		}
		items = append(items, map[string]any{"id": strconv.FormatInt(a.ID, 10), "authorId": strconv.FormatInt(a.AuthorID, 10), "nickname": a.Nickname, "displayName": a.DisplayName, "tierId": strconv.FormatInt(a.TierID, 10), "tierName": a.TierName, "priceKopecks": a.PriceKopecks, "autoRenew": a.AutoRenew, "paidUntil": billingTime(a.PaidUntil), "nextTierId": nextID, "nextPriceKopecks": nextPrice, "nextTierName": a.NextTierName.String, "stoppedReason": a.StoppedReason, "active": a.Active})
	}
	return respond(w, 200, map[string]any{"subscriptions": items, "hasMore": len(rows) > 24, "paymentsEnabled": s.billing.Enabled()})
}
func (s *Server) ownPayments(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	offset, err := postOffset(r)
	if err != nil {
		return err
	}
	rows, err := s.q.ListPayments(r.Context(), store.ListPaymentsParams{SubscriberID: u.ID, Offset: offset})
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, min(24, len(rows)))
	for _, p := range rows[:min(24, len(rows))] {
		items = append(items, map[string]any{"id": p.ID, "agreementId": strconv.FormatInt(p.AgreementID, 10), "amountKopecks": p.AmountKopecks, "status": p.Status, "renewal": p.Renewal, "createdAt": p.CreatedAt.Time.UTC().Format(time.RFC3339), "periodStart": billingTime(p.PeriodStart), "periodEnd": billingTime(p.PeriodEnd), "failureCode": p.FailureCode, "nickname": p.Nickname, "displayName": p.DisplayName, "tierName": p.TierName, "test": true})
	}
	return respond(w, 200, map[string]any{"payments": items, "hasMore": len(rows) > 24})
}
func (s *Server) cancelSubscription(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return missing()
	}
	if err = s.billing.Cancel(r.Context(), id, u.ID); errors.Is(err, pgx.ErrNoRows) {
		return missing()
	} else if err != nil {
		return err
	}
	return respond(w, 200, map[string]string{"message": "Автопродление отключено. Оплаченный доступ сохранён."})
}
func (s *Server) changeSubscriptionTier(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return missing()
	}
	in, err := decode[struct {
		TierID string `json:"tierId"`
		Price  *int64 `json:"priceKopecks"`
	}](r)
	if err != nil {
		return err
	}
	tier, err := strconv.ParseInt(in.TierID, 10, 64)
	if err != nil || tier < 1 || in.Price == nil {
		return invalid()
	}
	if err = s.billing.ChangeTier(r.Context(), id, u.ID, tier, *in.Price); errors.Is(err, pgx.ErrNoRows) {
		return missing()
	} else if err != nil {
		return err
	}
	return respond(w, 200, map[string]string{"message": "Уровень изменится со следующего оплаченного периода."})
}

// This machine-to-machine endpoint never accepts browser credentials or trusts the event payload.
func (s *Server) yooKassaWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Cookie") != "" || r.Header.Get("Origin") != "" {
		s.writeError(w, problem(403, "forbidden", "Запрос недоступен."))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var event struct {
		Type   string `json:"type"`
		Event  string `json:"event"`
		Object struct {
			ID string `json:"id"`
		} `json:"object"`
	}
	if err := json.UnmarshalRead(r.Body, &event); err != nil || event.Type != "notification" {
		s.writeError(w, invalid())
		return
	}
	if event.Event != "payment.succeeded" && event.Event != "payment.canceled" && event.Event != "payment.waiting_for_capture" {
		_ = respond(w, 200, map[string]string{"message": "Принято."})
		return
	}
	if err := s.billing.Webhook(r.Context(), event.Object.ID); err != nil {
		s.writeError(w, problem(503, "provider_unavailable", "Не удалось проверить оплату."))
		return
	}
	_ = respond(w, 200, map[string]string{"message": "Принято."})
}
