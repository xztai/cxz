package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func analyticsPeriod(r *http.Request, now time.Time) (time.Time, time.Time, error) {
	today := now.UTC().Truncate(24 * time.Hour)
	from, to := today.AddDate(0, 0, -29), today
	var err error
	if raw := r.URL.Query().Get("from"); raw != "" {
		from, err = time.Parse(time.DateOnly, raw)
		if err != nil {
			return from, to, invalid()
		}
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		to, err = time.Parse(time.DateOnly, raw)
		if err != nil {
			return from, to, invalid()
		}
	}
	if from.Year() < 2000 || to.Year() > 9998 || to.Before(from) || to.Sub(from) > 365*24*time.Hour {
		return from, to, problem(422, "invalid_period", "Выберите период от 1 до 366 дней.")
	}
	return from, to.AddDate(0, 0, 1), nil
}
func (s *Server) analytics(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	from, to, err := analyticsPeriod(r, time.Now())
	if err != nil {
		return err
	}
	tx, err := s.pool.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer socialRollback(tx)
	q := store.New(tx)
	if _, err = q.CreatorProfile(r.Context(), u.ID); errors.Is(err, pgx.ErrNoRows) {
		return problem(409, "profile_required", "Сначала сохраните профиль автора.")
	} else if err != nil {
		return err
	}
	days, err := q.PaymentDailyStats(r.Context(), store.PaymentDailyStatsParams{AuthorID: u.ID, PeriodFrom: pgtype.Timestamptz{Time: from, Valid: true}, PeriodTo: pgtype.Timestamptz{Time: to, Valid: true}, DayCount: int32(to.Sub(from) / (24 * time.Hour))})
	if err != nil {
		return err
	}
	type day struct {
		Date          string `json:"date"`
		PaymentCount  int64  `json:"paymentCount"`
		AmountKopecks int64  `json:"amountKopecks"`
	}
	daily := make([]day, 0, len(days))
	var count, amount int64
	for _, d := range days {
		daily = append(daily, day{d.Day, d.PaymentCount, d.AmountKopecks})
		count += d.PaymentCount
		amount += d.AmountKopecks
	}
	rows, err := q.ActiveSubscriptionsByTier(r.Context(), u.ID)
	if err != nil {
		return err
	}
	type tier struct {
		ID                  string `json:"id"`
		Name                string `json:"name"`
		Archived            bool   `json:"archived"`
		ActiveSubscriptions int64  `json:"activeSubscriptions"`
	}
	tiers := make([]tier, 0, len(rows))
	var active int64
	for _, t := range rows {
		tiers = append(tiers, tier{strconv.FormatInt(t.ID, 10), t.Name, t.Archived, t.SubscriberCount})
		active += t.SubscriberCount
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"from": from.Format(time.DateOnly), "to": to.AddDate(0, 0, -1).Format(time.DateOnly), "timezone": "UTC", "currency": "RUB", "test": true, "paymentCount": count, "amountKopecks": amount, "activeSubscriptions": active, "daily": daily, "tiers": tiers, "asOf": time.Now().UTC()})
}
