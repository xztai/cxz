package httpapi

import (
	"encoding/json/v2"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"testing"
	"time"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
)

type analyticsResponse struct {
	From                string `json:"from"`
	To                  string `json:"to"`
	Test                bool   `json:"test"`
	Timezone            string `json:"timezone"`
	PaymentCount        int64  `json:"paymentCount"`
	AmountKopecks       int64  `json:"amountKopecks"`
	ActiveSubscriptions int64  `json:"activeSubscriptions"`
	Daily               []struct {
		Date          string `json:"date"`
		PaymentCount  int64  `json:"paymentCount"`
		AmountKopecks int64  `json:"amountKopecks"`
	} `json:"daily"`
	Tiers []struct {
		ID                  string `json:"id"`
		Archived            bool   `json:"archived"`
		ActiveSubscriptions int64  `json:"activeSubscriptions"`
	} `json:"tiers"`
}

func analyticsResult(t *testing.T, body string) analyticsResponse {
	t.Helper()
	var a analyticsResponse
	if err := json.Unmarshal([]byte(body), &a); err != nil {
		t.Fatal(err)
	}
	return a
}
func useExistingSession(t *testing.T, h *harness, id int64) {
	t.Helper()
	token, err := auth.Token()
	if err != nil {
		t.Fatal(err)
	}
	if err = h.server.q.CreateSession(t.Context(), store.CreateSessionParams{TokenHash: auth.Digest(token), UserID: id, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}}); err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	req, _ := http.NewRequest("GET", h.url, nil)
	jar.SetCookies(req.URL, []*http.Cookie{{Name: "etheyl_session", Value: token, Path: "/"}})
	h.client = &http.Client{Jar: jar}
	var csrf struct {
		Token string `json:"token"`
	}
	if err = json.Unmarshal([]byte(h.expect("GET", "/auth/csrf", "", 200)), &csrf); err != nil {
		t.Fatal(err)
	}
	h.csrf = csrf.Token
}
func TestIntegrationAnalyticsConfirmedPaymentsUTCAndActiveSnapshot(t *testing.T) {
	f := billingHarness(t, true)
	f.paid(t)
	h := f.h
	// Identical repeated provider events cannot change payment counts/sums.
	for range 3 {
		if err := f.svc.Webhook(t.Context(), "remote-"+f.payment.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.pool.Exec(t.Context(), "UPDATE payments SET completed_at='2026-01-31T23:59:59Z' WHERE id=$1", f.payment.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.pool.Exec(t.Context(), `INSERT INTO payments(id,agreement_id,tier_id,amount_kopecks,renewal,save_method,request_body,provider_id,status,completed_at)
 VALUES('paid-next-day',$1,$2,30000,true,false,'{}','provider-next','succeeded','2026-02-01T00:00:00Z'),
 ('declined',$1,$2,99999,true,false,'{}','provider-decline','canceled','2026-02-01T12:00:00Z'),
 ('pending',$1,$2,88888,false,false,'{}','provider-pending','pending',NULL)`, mustID(t, f.payment.AgreementID), mustID(t, f.low.ID)); err != nil {
		t.Fatal(err)
	}
	h.expect("POST", "/me/subscriptions/"+f.payment.AgreementID+"/cancel", "", 200)
	if _, err := h.pool.Exec(t.Context(), "UPDATE subscription_tiers SET archived_at=now() WHERE id=$1", mustID(t, f.low.ID)); err != nil {
		t.Fatal(err)
	}
	useExistingSession(t, h, 1)
	get := func(query string) analyticsResponse {
		return analyticsResult(t, h.expect("GET", "/me/analytics?"+query, "", 200))
	}
	report := get("from=2026-01-31&to=2026-02-02")
	if !report.Test || report.Timezone != "UTC" || report.PaymentCount != 2 || report.AmountKopecks != 40000 || len(report.Daily) != 3 || report.Daily[2].PaymentCount != 0 {
		t.Fatal("incorrect payment aggregation", report)
	}
	if report.Daily[0].AmountKopecks != 10000 || report.Daily[1].AmountKopecks != 30000 {
		t.Fatal("UTC boundary error", report.Daily)
	}
	if report.ActiveSubscriptions != 1 {
		t.Fatal("canceled/archived paid subscription missing from active snapshot")
	}
	if len(report.Tiers) != 2 || !report.Tiers[1].Archived || report.Tiers[1].ActiveSubscriptions != 1 {
		t.Fatal("wrong tier distribution")
	}
	only := get("from=2026-02-01&to=2026-02-01")
	if only.AmountKopecks != 30000 || only.PaymentCount != 1 || len(only.Daily) != 1 {
		t.Fatal("inclusive day filter")
	}
	empty := get("from=2025-01-01&to=2025-01-02")
	if empty.AmountKopecks != 0 || empty.ActiveSubscriptions != 1 {
		t.Fatal("period changed active snapshot")
	}
	raw := h.expect("GET", "/me/analytics?from=2026-01-31&to=2026-02-02", "", 200)
	for _, secret := range []string{"saved-method", "request_body", "provider-next", "reader@example"} {
		if strings.Contains(raw, secret) {
			t.Fatal("private billing data exposed")
		}
	}
	// A second creator cannot request another author's aggregates with injected parameters.
	newTestSession(t, h, "another_creator")
	h.expect("PUT", "/me/profile", `{"nickname":"another_creator","displayName":"Другой автор","description":""}`, 200)
	other := get("from=2026-01-31&to=2026-02-02&authorId=1")
	if other.PaymentCount != 0 || other.ActiveSubscriptions != 0 {
		t.Fatal("cross-author analytics leak")
	}
	useExistingSession(t, h, 1)
	for _, change := range []string{"UPDATE subscriptions SET revoked_at=now()", "UPDATE subscriptions SET revoked_at=NULL,starts_at=now()+interval '1 day',expires_at=now()+interval '2 days'", "UPDATE subscriptions SET starts_at=now()-interval '2 months',expires_at=now()-interval '1 second'"} {
		if _, err := h.pool.Exec(t.Context(), change); err != nil {
			t.Fatal(err)
		}
		if get("from=2026-01-31&to=2026-02-02").ActiveSubscriptions != 0 {
			t.Fatal("inactive grant counted")
		}
	}
	if _, err := h.pool.Exec(t.Context(), "UPDATE subscriptions SET starts_at=now()-interval '1 day',expires_at=now()+interval '1 month'; UPDATE users SET blocked=true WHERE id=2"); err != nil {
		t.Fatal(err)
	}
	if get("from=2026-01-31&to=2026-02-02").ActiveSubscriptions != 0 {
		t.Fatal("blocked subscriber counted")
	}
	h.expect("GET", "/me/analytics?from=2024-01-01&to=2025-01-01", "", 422)
	// The provider uniqueness constraint is the second defense against duplicate money.
	if _, err := h.pool.Exec(t.Context(), "INSERT INTO payments(id,agreement_id,tier_id,amount_kopecks,renewal,save_method,request_body,provider_id,status) VALUES('duplicate',$1,$2,10000,false,false,'{}','provider-next','succeeded')", mustID(t, f.payment.AgreementID), mustID(t, f.low.ID)); err == nil {
		t.Fatal("duplicate provider payment allowed")
	}
}
func TestIntegrationAnalyticsUsesFrozenPaidAmounts(t *testing.T) {
	f := billingHarness(t, false)
	f.paid(t)
	if _, err := f.h.pool.Exec(t.Context(), "UPDATE subscription_tiers SET price_kopecks=99900 WHERE id=$1", mustID(t, f.low.ID)); err != nil {
		t.Fatal(err)
	}
	useExistingSession(t, f.h, 1)
	today := time.Now().UTC().Format(time.DateOnly)
	a := analyticsResult(t, f.h.expect("GET", "/me/analytics?from="+today+"&to="+today, "", 200))
	if a.AmountKopecks != 10000 || a.PaymentCount != 1 {
		t.Fatal("analytics used mutable tier price", strconv.FormatInt(a.AmountKopecks, 10))
	}
}
