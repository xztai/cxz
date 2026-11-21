package main

import (
	"encoding/json/v2"
	"fmt"
	"strconv"
	"time"

	"etheyl.local/backend/internal/billing"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
)

type agreement struct {
	id, subscriber int64
	tier           store.SubscriptionTier
}

type payment struct {
	status, reason string
	created        time.Time
	start, end     time.Time
	renewal, save  bool
}

func (s *seeder) newAgreement(user, tier string, created time.Time, autoRenew bool) (agreement, error) {
	t := s.tiers[tier]
	subscriber := s.users[user]
	a, err := s.q.CreateAgreement(s.ctx, store.CreateAgreementParams{
		SubscriberID: subscriber, AuthorID: t.AuthorID, TierID: t.ID, PriceKopecks: t.PriceKopecks,
	})
	if err != nil {
		return agreement{}, err
	}
	if err := s.q.PrepareAgreement(s.ctx, store.PrepareAgreementParams{
		ID: a.ID, TierID: t.ID, PriceKopecks: t.PriceKopecks, AutoRenew: autoRenew,
	}); err != nil {
		return agreement{}, err
	}
	_, err = s.tx.Exec(s.ctx, `
		UPDATE billing_agreements SET created_at=$2::timestamptz, consent_at=CASE WHEN auto_renew THEN $2::timestamptz END WHERE id=$1
	`, a.ID, created)
	return agreement{a.ID, subscriber, t}, err
}

func (a agreement) method() string {
	return fmt.Sprintf("seed-test-method-%d", a.id)
}

func (s *seeder) putPayment(a agreement, p payment) error {
	id := fmt.Sprintf("seed-%d-%d", a.id, p.created.Unix())
	req := billing.CreateRequest{
		Amount:      billing.Amount{Value: fmt.Sprintf("%d.%02d", a.tier.PriceKopecks/100, a.tier.PriceKopecks%100), Currency: "RUB"},
		Capture:     true,
		SaveMethod:  p.save,
		Description: "Синтетическая оплата etheyl dev",
		Metadata:    map[string]string{"etheyl_payment_id": id, "etheyl_agreement_id": strconv.FormatInt(a.id, 10)},
	}
	if p.renewal {
		req.MethodID = a.method()
	} else {
		req.Confirmation = &billing.Confirmation{Type: "redirect", ReturnURL: s.origin + "/account/payments/" + id}
	}
	// The API exposes test=true. The database stores the immutable request but
	// has no test or currency column, so mark the synthetic request as well.
	body, err := json.Marshal(struct {
		billing.CreateRequest
		Test bool `json:"test"`
	}{CreateRequest: req, Test: true})
	if err != nil {
		return err
	}
	var firstSent, completed time.Time
	var providerID string
	attempts := 0
	available := instant(p.created)
	if p.status != "queued" {
		firstSent = p.created.Add(10 * time.Second)
		attempts = 1
	}
	switch p.status {
	case "queued", "pending":
		// The worker must never claim the simulated in-flight attempts.
		available = pgtype.Timestamptz{InfinityModifier: pgtype.Infinity, Valid: true}
		if p.status == "pending" {
			providerID = "test-" + id
		}
	case "succeeded":
		providerID = "test-" + id
		completed = p.start
		if p.renewal {
			completed = p.created.Add(time.Minute)
		}
	case "canceled":
		completed = p.created.Add(time.Minute)
		if p.reason == "payment_canceled" {
			providerID = "test-" + id
		}
	}
	_, err = s.tx.Exec(s.ctx, `
		INSERT INTO payments(id,agreement_id,tier_id,amount_kopecks,renewal,save_method,request_body,
		   status,provider_id,created_at,first_sent_at,available_at,lease_until,
		   completed_at,period_start,period_end,failure_code,attempts)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$10,$13,$14,$15,$16,$17)
	`, id, a.id, a.tier.ID, a.tier.PriceKopecks, p.renewal, p.save, string(body),
		p.status, providerID, p.created, instant(firstSent), available,
		instant(completed), instant(p.start), instant(p.end), p.reason, attempts)
	return err
}

type paidFixture struct {
	user, tier, stop, next string
	until, firstStart      time.Time
	months, failed         int
	renew                  bool
}

// Walk back on the original billing day. When a prior month clipped that
// day, extend the history until its first charge can establish the anchor.
func previousCharge(at time.Time, anchor int) time.Time {
	last := time.Date(at.Year(), at.Month(), 0, at.Hour(), at.Minute(), at.Second(), 0, time.UTC)
	return last.AddDate(0, 0, min(last.Day(), anchor)-last.Day())
}

func (s *seeder) paid(f paidFixture) error {
	anchor := f.until.Day()
	start, months := f.firstStart, f.months
	if start.IsZero() {
		start = f.until
		for range months {
			start = previousCharge(start, anchor)
		}
		for start.Day() != anchor {
			start = previousCharge(start, anchor)
			months++
		}
	} else {
		anchor = start.Day()
	}
	autoRenew := f.renew || months > 1
	first := start.Add(-3 * time.Minute)
	a, err := s.newAgreement(f.user, f.tier, first.AddDate(0, 0, -2*f.failed), autoRenew)
	if err != nil {
		return err
	}
	for i := f.failed; i > 0; i-- {
		if err := s.putPayment(a, payment{
			status: "canceled", reason: []string{"payment_canceled", "provider_rejected"}[i%2],
			created: first.AddDate(0, 0, -2*i),
		}); err != nil {
			return err
		}
	}
	for i := range months {
		end := billing.NextMonth(start, anchor)
		created := start.Add(-3 * time.Minute)
		if i > 0 {
			created = start
		}
		if err := s.putPayment(a, payment{
			status: "succeeded", created: created,
			start: start, end: end, renewal: i > 0, save: i == 0 && autoRenew,
		}); err != nil {
			return err
		}
		method := ""
		if i == 0 && autoRenew {
			method = a.method()
		}
		if err := s.q.CompleteAgreement(s.ctx, store.CompleteAgreementParams{
			ID: a.id, TierID: a.tier.ID, PriceKopecks: a.tier.PriceKopecks,
			PaidUntil: instant(end), AnchorDay: int32(anchor), MethodID: method,
		}); err != nil {
			return err
		}
		if err := s.q.PutPaidAccess(s.ctx, store.PutPaidAccessParams{
			SubscriberID: a.subscriber, AuthorID: a.tier.AuthorID, TierID: a.tier.ID,
			StartsAt: instant(start), ExpiresAt: instant(end),
		}); err != nil {
			return err
		}
		start = end
	}
	if autoRenew && !f.renew {
		f.stop = "canceled"
	}
	if f.stop != "" {
		return s.q.StopAgreement(s.ctx, store.StopAgreementParams{ID: a.id, StoppedReason: f.stop})
	}
	if f.next != "" {
		next := s.tiers[f.next]
		return s.q.ChangeAgreementTier(s.ctx, store.ChangeAgreementTierParams{
			ID: a.id, NextTierID: nullableID(next.ID), NextPriceKopecks: nullableID(next.PriceKopecks),
		})
	}
	return nil
}

// Analytics events are authored as one-time purchases on distinct agreements.
func (s *seeder) analyticsPayment(user, tier string, daysAgo int) error {
	start := s.now.AddDate(0, 0, -daysAgo)
	return s.paid(paidFixture{user: user, tier: tier, firstStart: start, months: 1})
}

func (s *seeder) createBilling() error {
	day := func(n int) time.Time { return s.now.AddDate(0, 0, n) }
	for _, f := range []paidFixture{
		{user: readers[0].nick, tier: silver, until: day(20), months: 3, renew: true},
		{user: readers[1].nick, tier: gold, until: day(5), months: 1},
		{user: readers[2].nick, tier: silver, until: day(10), months: 1, renew: true, stop: "canceled"},
		{user: readers[3].nick, tier: silver, until: day(-1), months: 1},
		{user: readers[4].nick, tier: silver, until: day(14), months: 1, renew: true, next: gold},
		{user: readers[5].nick, tier: archived, until: day(12), months: 1, renew: true},
		{user: readers[9].nick, tier: silver, until: day(22), months: 1},
		{user: readers[9].nick, tier: levEntry, until: day(18), months: 1},
		{user: readers[9].nick, tier: markUsed, until: day(16), months: 1},
		{user: readers[11].nick, tier: gold, until: day(6), months: 1, failed: 63},
		{user: readers[12].nick, tier: levPro, until: day(7), months: 1},
		{user: readers[13].nick, tier: markUsed, until: day(11), months: 1},
		{user: readers[14].nick, tier: platinum, until: day(23), months: 1},
	} {
		if err := s.paid(f); err != nil {
			return err
		}
	}
	for _, f := range []struct {
		user, tier string
		payments   []payment
	}{
		{readers[6].nick, silver, []payment{{status: "pending", created: s.now.Add(-2 * time.Minute)}}},
		{readers[6].nick, levEntry, []payment{{status: "queued", created: s.now.Add(-time.Minute)}}},
		{readers[7].nick, gold, []payment{{status: "review", reason: "unknown_result", created: s.now.Add(-25 * time.Hour)}}},
		{readers[8].nick, silver, []payment{
			{status: "canceled", reason: "payment_canceled", created: day(-15)},
			{status: "canceled", reason: "provider_rejected", created: day(-10)},
			{status: "canceled", reason: "payment_canceled", created: day(-4)},
		}},
		{readers[10].nick, levEntry, []payment{{status: "canceled", reason: "provider_rejected", created: day(-8)}}},
	} {
		a, err := s.newAgreement(f.user, f.tier, f.payments[0].created, false)
		if err != nil {
			return err
		}
		for _, p := range f.payments {
			if err := s.putPayment(a, p); err != nil {
				return err
			}
			if p.status == "canceled" {
				if err := s.q.StopAgreement(s.ctx, store.StopAgreementParams{ID: a.id, StoppedReason: p.reason}); err != nil {
					return err
				}
			}
		}
	}
	for i, a := range catalog {
		if err := s.analyticsPayment(a.nick, []string{silver, gold, platinum}[i%3], 2+3*i); err != nil {
			return err
		}
		if i < 18 {
			if err := s.analyticsPayment(a.nick, []string{levEntry, levPro}[i%2], 1+3*i/2); err != nil {
				return err
			}
		}
	}
	return nil
}
