package httpapi

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"etheyl.local/backend/internal/billing"
	"etheyl.local/backend/internal/config"
	"etheyl.local/backend/internal/mailqueue"
	"etheyl.local/backend/internal/store"
)

type fakeYooKassa struct {
	mu            sync.Mutex
	payments      map[string]billing.RemotePayment
	bodies        map[string]string
	createCalls   int
	getCalls      int
	loseResponse  bool
	failGet       bool
	renewalStatus string
}

func (f *fakeYooKassa) Create(_ context.Context, key, body string) (billing.RemotePayment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls++
	if prev, ok := f.bodies[key]; ok && prev != body {
		return billing.RemotePayment{}, errors.New("idempotence payload changed")
	}
	f.bodies[key] = body
	p, ok := f.payments[key]
	if !ok {
		var req billing.CreateRequest
		if err := json.Unmarshal([]byte(body), &req); err != nil {
			return p, err
		}
		p = billing.RemotePayment{ID: "remote-" + key, Status: "pending", Test: true, Amount: req.Amount, Metadata: req.Metadata}
		p.Recipient.AccountID = "1531842"
		p.Confirmation.URL = "https://yoomoney.ru/checkout?payment=" + p.ID
		if req.SaveMethod {
			p.Method.ID = "saved-method"
			p.Method.Saved = true
		}
		if req.MethodID != "" {
			p.Status = f.renewalStatus
			p.Paid = p.Status == "succeeded"
		}
		f.payments[key] = p
	}
	if f.loseResponse {
		f.loseResponse = false
		return billing.RemotePayment{}, billing.ErrProvider
	}
	return p, nil
}
func (f *fakeYooKassa) Get(_ context.Context, id string) (billing.RemotePayment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	if f.failGet {
		return billing.RemotePayment{}, billing.ErrProvider
	}
	for _, p := range f.payments {
		if p.ID == id {
			return p, nil
		}
	}
	return billing.RemotePayment{}, billing.ErrProvider
}
func (f *fakeYooKassa) succeed(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.payments[id]
	p.Status = "succeeded"
	p.Paid = true
	f.payments[id] = p
}
func (f *fakeYooKassa) count() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.payments) }

func (f *fakeYooKassa) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.createCalls + f.getCalls
}

type billingFixture struct {
	h         *harness
	provider  *fakeYooKassa
	svc       *billing.Service
	low, high Tier
	post      Post
	payment   Payment
}

func billingHarness(t *testing.T, renew bool) *billingFixture {
	t.Helper()
	h := setup(t)
	newTestSession(t, h, "writer")
	h.expect("PUT", "/me/profile", `{"nickname":"writer","displayName":"Автор","description":""}`, 200)
	low := testTier(t, h.expect("POST", "/me/tiers", `{"name":"Читатель","description":"","priceKopecks":10000}`, 201))
	high := testTier(t, h.expect("POST", "/me/tiers", `{"name":"Друг","description":"","priceKopecks":30000}`, 201))
	post := postResult(t, h.expect("POST", "/me/posts", postBody(strconv.Quote(low.ID)), 201))
	h.expect("POST", "/me/posts/"+post.ID+"/publish", "", 200)
	newTestSession(t, h, "reader")
	provider := &fakeYooKassa{payments: make(map[string]billing.RemotePayment), bodies: make(map[string]string), renewalStatus: "succeeded"}
	svc, err := billing.NewWithProvider(h.pool, config.Config{Origin: "http://example.test", OutboxKeys: testOutboxKey, YooKassaShopID: "1531842", YooKassaAPIKey: "test_key"}, provider)
	if err != nil {
		t.Fatal(err)
	}
	h.server.billing = svc
	h.expect("POST", "/billing/checkout", fmt.Sprintf(`{"tierId":%q,"priceKopecks":9999,"autoRenew":false}`, low.ID), 409)
	raw := h.expect("POST", "/billing/checkout", fmt.Sprintf(`{"tierId":%q,"priceKopecks":10000,"autoRenew":%t}`, low.ID, renew), 202)
	var result struct {
		Payment Payment `json:"payment"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatal(err)
	}
	return &billingFixture{h: h, provider: provider, svc: svc, low: low, high: high, post: post, payment: result.Payment}
}
func (f *billingFixture) process(t *testing.T) {
	t.Helper()
	found, err := f.svc.ProcessOne(t.Context())
	if err != nil || !found {
		t.Fatal("process payment", found, err)
	}
}
func (f *billingFixture) paid(t *testing.T) {
	t.Helper()
	f.process(t)
	f.provider.succeed(f.payment.ID)
	if err := f.svc.Webhook(t.Context(), "remote-"+f.payment.ID); err != nil {
		t.Fatal(err)
	}
}
func (f *billingFixture) due(t *testing.T) {
	t.Helper()
	_, err := f.h.pool.Exec(t.Context(), "UPDATE billing_agreements SET paid_until=now()-interval '1 second' WHERE id=$1", mustID(t, f.payment.AgreementID))
	if err != nil {
		t.Fatal(err)
	}
}
func (f *billingFixture) openCheckout(t *testing.T) (store.Payment, store.BillingAgreement) {
	t.Helper()
	agreementID := mustID(t, f.payment.AgreementID)
	payment, err := f.h.server.q.OpenPayment(t.Context(), agreementID)
	if err != nil {
		t.Fatal("load open payment", err)
	}
	agreement, err := f.h.server.q.PaymentOwner(t.Context(), payment.ID)
	if err != nil {
		t.Fatal("load agreement terms", err)
	}
	return payment, agreement
}

func assertCheckoutResponse(t *testing.T, status int, raw, paymentID, code string) {
	t.Helper()
	wantStatus := http.StatusAccepted
	if code != "" {
		wantStatus = http.StatusConflict
	}
	if status != wantStatus {
		t.Fatalf("checkout returned status %d, want %d", status, wantStatus)
	}
	var result struct {
		Payment jsontext.Value `json:"payment"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatal("invalid checkout response JSON")
	}
	if code != "" {
		if result.Error.Code != code || result.Payment != nil {
			t.Errorf("checkout conflict code = %q, want %q, payment present = %t", result.Error.Code, code, result.Payment != nil)
		}
		return
	}
	var payment Payment
	if err := json.Unmarshal(result.Payment, &payment); err != nil || payment.ID != paymentID {
		t.Error("checkout did not return the original payment")
	}
}

func (f *billingFixture) assertCheckoutUnchanged(t *testing.T, before store.Payment, agreementBefore store.BillingAgreement, calls int) {
	t.Helper()
	var count int
	err := f.h.pool.QueryRow(t.Context(), `
		SELECT count(*) FROM payments WHERE agreement_id = $1 AND status IN ('queued', 'pending', 'review')`,
		before.AgreementID).Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("open payment count = %d, want 1, query error = %v", count, err)
	}
	after, agreementAfter := f.openCheckout(t)
	if after != before || agreementAfter != agreementBefore {
		t.Error("checkout changed the persisted payment or agreement terms")
	}
	if f.provider.calls() != calls {
		t.Error("checkout contacted the payment provider")
	}
}

func TestIntegrationBillingCheckoutReplayTerms(t *testing.T) {
	for _, tc := range []struct {
		name        string
		openStatus  string
		tierChanged bool
		openRenew   bool
		renew       bool
		livePrice   int64
		price       int64
		automatic   bool
		code        string
	}{
		{name: "identical queued", openStatus: "queued", price: 10000},
		{name: "identical pending", openStatus: "pending", price: 10000},
		{name: "identical review", openStatus: "review", price: 10000, openRenew: true, renew: true},
		{name: "same price different tier", openStatus: "pending", tierChanged: true, price: 10000, code: "checkout_terms_conflict"},
		{name: "grant consent", openStatus: "queued", renew: true, price: 10000, code: "checkout_terms_conflict"},
		{name: "withdraw consent", openStatus: "review", openRenew: true, price: 10000, code: "checkout_terms_conflict"},
		{name: "changed price confirmed", openStatus: "queued", livePrice: 15000, price: 15000, code: "checkout_terms_conflict"},
		{name: "obsolete price rejected first", openStatus: "queued", livePrice: 15000, price: 10000, code: "price_changed"},
		{name: "automatic renewal", openStatus: "queued", openRenew: true, renew: true, price: 10000, automatic: true, code: "checkout_terms_conflict"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := billingHarness(t, tc.openRenew)
			if tc.openStatus == "pending" || tc.openStatus == "review" {
				f.process(t)
			}
			if tc.openStatus == "review" {
				if _, err := f.h.pool.Exec(t.Context(), "UPDATE payments SET status = 'review' WHERE id = $1", f.payment.ID); err != nil {
					t.Fatal(err)
				}
			}
			if tc.automatic {
				f.paid(t)
				f.due(t)
				if err := f.svc.Schedule(t.Context()); err != nil {
					t.Fatal(err)
				}
			}
			tier := f.low.ID
			if tc.tierChanged {
				tier = f.high.ID
				tc.livePrice = tc.price
			}
			if tc.livePrice != 0 {
				if _, err := f.h.pool.Exec(t.Context(), "UPDATE subscription_tiers SET price_kopecks = $1 WHERE id = $2", tc.livePrice, mustID(t, tier)); err != nil {
					t.Fatal(err)
				}
			}
			before, agreement := f.openCheckout(t)
			if before.Status != tc.openStatus || before.Renewal != tc.automatic || before.SaveMethod != tc.openRenew {
				t.Fatal("open payment fixture has unexpected status, charge type or consent")
			}
			calls := f.provider.calls()
			body := fmt.Sprintf(`{"tierId":%q,"priceKopecks":%d,"autoRenew":%t}`, tier, tc.price, tc.renew)
			status, raw, _ := f.h.call("POST", "/billing/checkout", body, true)
			assertCheckoutResponse(t, status, raw, f.payment.ID, tc.code)
			f.assertCheckoutUnchanged(t, before, agreement, calls)
		})
	}
}

func TestIntegrationBillingConcurrentCheckoutReplayTerms(t *testing.T) {
	f := billingHarness(t, true)
	before, agreement := f.openCheckout(t)
	calls := f.provider.calls()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			<-start
			renew := i%2 == 0
			body := fmt.Sprintf(`{"tierId":%q,"priceKopecks":10000,"autoRenew":%t}`, f.low.ID, renew)
			status, raw, _ := f.h.call("POST", "/billing/checkout", body, true)
			var code string
			if !renew {
				code = "checkout_terms_conflict"
			}
			assertCheckoutResponse(t, status, raw, f.payment.ID, code)
		})
	}
	close(start)
	wg.Wait()
	f.assertCheckoutUnchanged(t, before, agreement, calls)
}

func TestIntegrationBillingPaidAccessAndDuplicateNotifications(t *testing.T) {
	f := billingHarness(t, true)
	h := f.h
	if postResult(t, h.expect("GET", "/posts/"+f.post.ID, "", 200)).CanRead {
		t.Fatal("checkout granted access")
	}
	f.process(t)
	if postResult(t, h.expect("GET", "/posts/"+f.post.ID, "", 200)).CanRead {
		t.Fatal("pending granted access")
	}
	f.provider.succeed(f.payment.ID)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if err := f.svc.Webhook(t.Context(), "remote-"+f.payment.ID); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if !postResult(t, h.expect("GET", "/posts/"+f.post.ID, "", 200)).CanRead {
		t.Fatal("paid post stayed locked")
	}
	var count int
	if err := h.pool.QueryRow(t.Context(), "SELECT count(*) FROM subscriptions").Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	p, err := h.server.q.PaymentByID(t.Context(), f.payment.ID)
	if err != nil || p.Status != "succeeded" {
		t.Fatal(p.Status, err)
	}
	end := p.PeriodEnd
	if err = f.svc.Webhook(t.Context(), "remote-"+f.payment.ID); err != nil {
		t.Fatal(err)
	}
	after, _ := h.server.q.PaymentByID(t.Context(), f.payment.ID)
	if after.PeriodEnd != end {
		t.Fatal("duplicate extended access")
	}
	history := h.expect("GET", "/me/payments", "", 200)
	if strings.Contains(history, "saved-method") || strings.Contains(history, "request_body") || !strings.Contains(history, "succeeded") {
		t.Fatal("unsafe payment history")
	}
	h.expect("POST", "/billing/checkout", fmt.Sprintf(`{"tierId":%q,"priceKopecks":10000,"autoRenew":true}`, f.low.ID), 409)
	newTestSession(t, h, "outsider")
	h.expect("GET", "/me/payments/"+p.ID, "", 404)
	h.expect("POST", "/me/subscriptions/"+f.payment.AgreementID+"/cancel", "", 404)
}
func TestIntegrationBillingConcurrentWorkerRestartAndUnknownResults(t *testing.T) {
	f := billingHarness(t, false)
	f.provider.loseResponse = true
	f.process(t)
	// Restart with the same database and provider. Reuse the exact stored payload/key.
	restarted, err := billing.NewWithProvider(f.h.pool, config.Config{Origin: "http://example.test", OutboxKeys: testOutboxKey, YooKassaShopID: "1531842", YooKassaAPIKey: "test_key"}, f.provider)
	if err != nil {
		t.Fatal(err)
	}
	f.svc = restarted
	_, err = f.h.pool.Exec(t.Context(), "UPDATE payments SET available_at=now() WHERE id=$1", f.payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			_, err := f.svc.ProcessOne(t.Context())
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if f.provider.count() != 1 {
		t.Fatal("duplicate provider payment")
	}
	f.provider.succeed(f.payment.ID)
	if err = f.svc.Webhook(t.Context(), "remote-"+f.payment.ID); err != nil {
		t.Fatal(err)
	}
	f.due(t)
	if err = f.svc.Schedule(t.Context()); err != nil {
		t.Fatal(err)
	}
	if f.provider.count() != 1 {
		t.Fatal("renewal without consent")
	}
	var auto bool
	if err = f.h.pool.QueryRow(t.Context(), "SELECT auto_renew FROM billing_agreements").Scan(&auto); err != nil || auto {
		t.Fatal("auto-renew without consent", err)
	}
}
func TestIntegrationBillingFrozenPriceLevelChangeAndRenewal(t *testing.T) {
	f := billingHarness(t, true)
	f.paid(t)
	if _, err := f.h.pool.Exec(t.Context(), "UPDATE subscription_tiers SET price_kopecks=15000 WHERE id=$1", mustID(t, f.low.ID)); err != nil {
		t.Fatal(err)
	}
	f.due(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if err := f.svc.Schedule(t.Context()); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	f.process(t)
	var amount int64
	if err := f.h.pool.QueryRow(t.Context(), "SELECT amount_kopecks FROM payments WHERE renewal").Scan(&amount); err != nil || amount != 10000 {
		t.Fatal("price was not frozen", amount, err)
	}
	h := f.h
	h.expect("PUT", "/me/subscriptions/"+f.payment.AgreementID+"/tier", fmt.Sprintf(`{"tierId":%q,"priceKopecks":29999}`, f.high.ID), 409)
	h.expect("PUT", "/me/subscriptions/"+f.payment.AgreementID+"/tier", fmt.Sprintf(`{"tierId":%q,"priceKopecks":30000}`, f.high.ID), 200)
	var current int64
	if err := h.pool.QueryRow(t.Context(), "SELECT tier_id FROM subscriptions").Scan(&current); err != nil || current != mustID(t, f.low.ID) {
		t.Fatal("level changed mid-period", err)
	}
	f.due(t)
	if err := f.svc.Schedule(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.process(t)
	if err := h.pool.QueryRow(t.Context(), "SELECT tier_id FROM subscriptions").Scan(&current); err != nil || current != mustID(t, f.high.ID) {
		t.Fatal("new level not granted", err)
	}
	if f.provider.count() != 3 {
		t.Fatal("unexpected payment count")
	}
}
func TestIntegrationBillingDeclineStopsRetriesAndNotifies(t *testing.T) {
	f := billingHarness(t, true)
	f.paid(t)
	f.provider.renewalStatus = "canceled"
	f.due(t)
	if err := f.svc.Schedule(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.process(t)
	for range 3 {
		if err := f.svc.Schedule(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if f.provider.count() != 2 {
		t.Fatal("declined renewal retried")
	}
	var paidUntil time.Time
	var auto bool
	if err := f.h.pool.QueryRow(t.Context(), "SELECT expires_at,auto_renew FROM subscriptions JOIN billing_agreements USING(subscriber_id,author_id)").Scan(&paidUntil, &auto); err != nil || auto || !paidUntil.After(time.Now()) {
		t.Fatal("decline revoked paid access or kept renewal", err)
	}
	var count int
	if err := f.h.pool.QueryRow(t.Context(), "SELECT count(*) FROM mail_outbox WHERE kind='billing'").Scan(&count); err != nil || count != 1 {
		t.Fatal("missing/duplicate notice", count, err)
	}
	if err := f.h.server.q.CleanupMail(t.Context()); err != nil {
		t.Fatal(err)
	}
	var mailID int64
	if err := f.h.pool.QueryRow(t.Context(), "SELECT id FROM mail_outbox WHERE kind='billing'").Scan(&mailID); err != nil {
		t.Fatal("billing mail removed by action-token cleanup", err)
	}
	if err := f.h.server.q.FailMail(t.Context(), mailID); err != nil {
		t.Fatal(err)
	}
	if n, err := f.h.server.q.RequeueMail(t.Context(), mailID); err != nil || n != 1 {
		t.Fatal("billing mail cannot be requeued", n, err)
	}
	box, _ := mailqueue.NewCipher(testOutboxKey)
	sender := &recordingSender{}
	found, err := mailqueue.DeliverOne(t.Context(), f.h.pool, sender, box, f.h.server.log)
	if err != nil || !found {
		t.Fatal("notification delivery", err)
	}
	if len(sender.items) != 1 || !strings.Contains(sender.items[0].Body, "повторных автоматических") {
		t.Fatal("wrong notice")
	}
}

type recordingSender struct{ items []store.MailOutbox }

func (s *recordingSender) Send(_ context.Context, m store.MailOutbox) error {
	s.items = append(s.items, m)
	return nil
}
func TestIntegrationBillingCancellationAndArchiveKeepAccess(t *testing.T) {
	for _, reason := range []string{"cancel", "archive", "block"} {
		t.Run(reason, func(t *testing.T) {
			f := billingHarness(t, true)
			f.paid(t)
			switch reason {
			case "cancel":
				f.h.expect("POST", "/me/subscriptions/"+f.payment.AgreementID+"/cancel", "", 200)
			case "archive":
				if _, err := f.h.pool.Exec(t.Context(), "UPDATE subscription_tiers SET archived_at=now() WHERE id=$1", mustID(t, f.low.ID)); err != nil {
					t.Fatal(err)
				}
			case "block":
				if _, err := f.h.pool.Exec(t.Context(), "UPDATE users SET blocked=true WHERE nickname='reader'"); err != nil {
					t.Fatal(err)
				}
			}
			f.due(t)
			if err := f.svc.Schedule(t.Context()); err != nil {
				t.Fatal(err)
			}
			if f.provider.count() != 1 {
				t.Fatal("charged after stop")
			}
			var auto bool
			var revoked bool
			if err := f.h.pool.QueryRow(t.Context(), "SELECT auto_renew,revoked_at IS NOT NULL FROM billing_agreements JOIN subscriptions USING(subscriber_id,author_id)").Scan(&auto, &revoked); err != nil || auto || revoked {
				t.Fatal("stop violated access", err)
			}
		})
	}
}
func TestIntegrationBillingUntrustedWebhookAndMismatchedPayment(t *testing.T) {
	for _, field := range []string{"amount", "currency", "shop", "test", "metadata"} {
		t.Run(field, func(t *testing.T) {
			f := billingHarness(t, false)
			f.process(t)
			f.provider.mu.Lock()
			p := f.provider.payments[f.payment.ID]
			p.Status = "succeeded"
			p.Paid = true
			switch field {
			case "amount":
				p.Amount.Value = "0.01"
			case "currency":
				p.Amount.Currency = "USD"
			case "shop":
				p.Recipient.AccountID = "other"
			case "test":
				p.Test = false
			case "metadata":
				p.Metadata = map[string]string{"etheyl_payment_id": f.payment.ID, "etheyl_agreement_id": "other"}
			}
			f.provider.payments[f.payment.ID] = p
			f.provider.mu.Unlock()
			if err := f.svc.Webhook(t.Context(), p.ID); err != nil {
				t.Fatal(err)
			}
			var n int
			if err := f.h.pool.QueryRow(t.Context(), "SELECT count(*) FROM subscriptions").Scan(&n); err != nil || n != 0 {
				t.Fatal("untrusted payment granted access", err)
			}
			row, _ := f.h.server.q.PaymentByID(t.Context(), f.payment.ID)
			if row.Status != "review" {
				t.Fatal("mismatch did not block retries")
			}
		})
	}
	f := billingHarness(t, false)
	status, _, _ := f.h.call("POST", "/billing/checkout", `{}`, false)
	if status != 403 {
		t.Fatal("checkout bypassed CSRF")
	}
	f.h.expect("POST", "/billing/yookassa/webhook", `{"type":"notification","event":"payment.succeeded","object":{"id":"fake"}}`, 403)
	req, err := http.NewRequestWithContext(t.Context(), "POST", f.h.url+"/api/v1/billing/yookassa/webhook", strings.NewReader(`{"type":"notification","event":"payment.succeeded","object":{"id":"fake"}}`))
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	if res.StatusCode != 503 {
		t.Fatal("forged webhook accepted")
	}
}
func TestIntegrationBillingIdempotencyExpiryBlocksPost(t *testing.T) {
	f := billingHarness(t, false)
	_, err := f.h.pool.Exec(t.Context(), "UPDATE payments SET first_sent_at=now()-interval '25 hours' WHERE id=$1", f.payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.process(t)
	if f.provider.count() != 0 {
		t.Fatal("POST beyond provider idempotence window")
	}
	p, _ := f.h.server.q.PaymentByID(t.Context(), f.payment.ID)
	if p.Status != "review" {
		t.Fatal("unknown result not held")
	}
}

func TestIntegrationBillingStopCancelsUnsentRenewal(t *testing.T) {
	f := billingHarness(t, true)
	f.paid(t)
	f.due(t)
	if err := f.svc.Schedule(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.h.expect("POST", "/me/subscriptions/"+f.payment.AgreementID+"/cancel", "", 200)
	f.process(t)
	if f.provider.count() != 1 {
		t.Fatal("unsent renewal charged after cancellation")
	}
	var status string
	if err := f.h.pool.QueryRow(t.Context(), "SELECT status FROM payments WHERE renewal").Scan(&status); err != nil || status != "canceled" {
		t.Fatal(status, err)
	}
}
func TestIntegrationBillingWebhookBeforeCreateResponse(t *testing.T) {
	f := billingHarness(t, true)
	f.provider.loseResponse = true
	f.process(t)
	f.provider.succeed(f.payment.ID)
	if err := f.svc.Webhook(t.Context(), "remote-"+f.payment.ID); err != nil {
		t.Fatal(err)
	}
	p, err := f.h.server.q.PaymentByID(t.Context(), f.payment.ID)
	if err != nil || p.Status != "succeeded" {
		t.Fatal("webhook did not recover lost response", err)
	}
	if !postResult(t, f.h.expect("GET", "/posts/"+f.post.ID, "", 200)).CanRead {
		t.Fatal("missing recovered access")
	}
}
func TestIntegrationBillingLongWorkerOutageDoesNotBackBill(t *testing.T) {
	f := billingHarness(t, true)
	f.paid(t)
	if _, err := f.h.pool.Exec(t.Context(), "UPDATE billing_agreements SET paid_until=now()-interval '4 months'"); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Schedule(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.process(t)
	if err := f.svc.Schedule(t.Context()); err != nil {
		t.Fatal(err)
	}
	if found, err := f.svc.ProcessOne(t.Context()); err != nil || found {
		t.Fatal("back billed elapsed months", err)
	}
	if f.provider.count() != 2 {
		t.Fatal("outage caused multiple charges")
	}
}
func TestIntegrationBillingDuplicateFailureNotification(t *testing.T) {
	f := billingHarness(t, true)
	f.process(t)
	f.provider.mu.Lock()
	p := f.provider.payments[f.payment.ID]
	p.Status = "canceled"
	f.provider.payments[f.payment.ID] = p
	f.provider.mu.Unlock()
	for range 3 {
		if err := f.svc.Webhook(t.Context(), p.ID); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := f.h.pool.QueryRow(t.Context(), "SELECT count(*) FROM mail_outbox WHERE kind='billing'").Scan(&n); err != nil || n != 1 {
		t.Fatal("duplicate failure email", n, err)
	}
}

func TestIntegrationBillingServerWebhookIgnoresPayloadAndStalePending(t *testing.T) {
	f := billingHarness(t, true)
	f.process(t)
	f.provider.succeed(f.payment.ID)
	body := fmt.Sprintf(`{"type":"notification","event":"payment.succeeded","object":{"id":%q,"status":"canceled","amount":{"value":"0.01","currency":"USD"}}}`, "remote-"+f.payment.ID)
	req, err := http.NewRequestWithContext(t.Context(), "POST", f.h.url+"/api/v1/billing/yookassa/webhook", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal("valid server webhook failed", res.StatusCode)
	}
	if !postResult(t, f.h.expect("GET", "/posts/"+f.post.ID, "", 200)).CanRead {
		t.Fatal("webhook trusted false payload instead of provider GET")
	}
	before, err := f.h.server.q.PaymentByID(t.Context(), f.payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.provider.mu.Lock()
	p := f.provider.payments[f.payment.ID]
	p.Status = "pending"
	p.Paid = false
	f.provider.payments[f.payment.ID] = p
	f.provider.mu.Unlock()
	if err = f.svc.Webhook(t.Context(), p.ID); err != nil {
		t.Fatal(err)
	}
	after, _ := f.h.server.q.PaymentByID(t.Context(), f.payment.ID)
	if after.Status != "succeeded" || after.PeriodEnd != before.PeriodEnd {
		t.Fatal("stale pending overwrote paid state")
	}
}

func TestIntegrationBillingManualReconciliation(t *testing.T) {
	for _, tt := range []struct {
		name        string
		remoteState string
		unmatched   bool
		mismatch    bool
		otherID     bool
		settled     bool
		failGet     bool
		want        billing.ReconciliationOutcome
		duplicate   billing.ReconciliationOutcome
		status      string
		grants      int
		notices     int
		callback    int
	}{
		{name: "unmatched", remoteState: "succeeded", unmatched: true, want: billing.ReconciliationUnmatched, status: "pending", callback: 200},
		{name: "pending", remoteState: "pending", want: billing.ReconciliationPending, status: "pending", callback: 200},
		{name: "mismatch", remoteState: "succeeded", mismatch: true, want: billing.ReconciliationReview, status: "review", callback: 200},
		{name: "unexpected status", remoteState: "waiting_for_capture", want: billing.ReconciliationReview, status: "review", callback: 200},
		{name: "succeeded", remoteState: "succeeded", want: billing.ReconciliationSucceeded, duplicate: billing.ReconciliationAlreadySucceeded, status: "succeeded", grants: 1, callback: 200},
		{name: "canceled", remoteState: "canceled", want: billing.ReconciliationCanceled, duplicate: billing.ReconciliationAlreadyCanceled, status: "canceled", notices: 1, callback: 200},
		{name: "terminal different provider payment", remoteState: "succeeded", settled: true, otherID: true, want: billing.ReconciliationReview, status: "succeeded", grants: 1, callback: 200},
		{name: "terminal different amount", remoteState: "succeeded", settled: true, mismatch: true, want: billing.ReconciliationReview, status: "succeeded", grants: 1, callback: 200},
		{name: "terminal stale provider state", remoteState: "pending", settled: true, want: billing.ReconciliationAlreadySucceeded, status: "succeeded", grants: 1, callback: 200},
		{name: "provider unavailable", failGet: true, status: "pending", callback: 503},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := billingHarness(t, true)
			if tt.settled {
				f.paid(t)
			} else {
				f.process(t)
			}
			ctx := t.Context()
			before, err := f.h.server.q.PaymentByID(ctx, f.payment.ID)
			if err != nil {
				t.Fatal("read payment before reconciliation")
			}

			f.provider.mu.Lock()
			p := f.provider.payments[f.payment.ID]
			p.Status = tt.remoteState
			p.Paid = p.Status == "succeeded"
			if tt.unmatched {
				p.Metadata = map[string]string{
					"etheyl_payment_id":   "unrelated-payment",
					"etheyl_agreement_id": p.Metadata["etheyl_agreement_id"],
				}
			}
			if tt.mismatch {
				p.Amount.Value = "0.01"
			}
			if tt.otherID {
				p.ID = "different-remote-payment"
				f.provider.payments["different"] = p
			} else {
				f.provider.payments[f.payment.ID] = p
			}
			f.provider.failGet = tt.failGet
			creates := f.provider.createCalls
			f.provider.mu.Unlock()

			assertState := func() {
				t.Helper()
				row, err := f.h.server.q.PaymentByID(ctx, f.payment.ID)
				if err != nil || row.Status != tt.status {
					t.Fatalf("payment state = %q, want %q (query failed: %t)", row.Status, tt.status, err != nil)
				}
				if tt.settled && (row.ProviderID != before.ProviderID || row.CompletedAt != before.CompletedAt || row.PeriodEnd != before.PeriodEnd || row.FailureCode != before.FailureCode) {
					t.Error("reconciliation changed a terminal payment")
				}
				var grants, notices int
				err = f.h.pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM subscriptions), (SELECT count(*) FROM mail_outbox WHERE kind='billing')").Scan(&grants, &notices)
				if err != nil || grants != tt.grants || notices != tt.notices {
					t.Fatalf("grants = %d, notices = %d, want %d and %d (query failed: %t)", grants, notices, tt.grants, tt.notices, err != nil)
				}
				f.provider.mu.Lock()
				defer f.provider.mu.Unlock()
				if f.provider.createCalls != creates {
					t.Error("reconciliation attempted provider creation")
				}
			}

			outcome, err := f.svc.Reconcile(ctx, p.ID)
			if outcome != tt.want || errors.Is(err, billing.ErrProvider) != tt.failGet || (err != nil && !tt.failGet) {
				t.Fatalf("reconciliation = %q, want %q (error: %t)", outcome, tt.want, err != nil)
			}
			assertState()
			if tt.duplicate != "" {
				outcome, err := f.svc.Reconcile(ctx, p.ID)
				if err != nil || outcome != tt.duplicate {
					t.Fatalf("duplicate reconciliation = %q, want %q (error: %t)", outcome, tt.duplicate, err != nil)
				}
				assertState()
			}

			body := fmt.Sprintf(`{"type":"notification","event":"payment.succeeded","object":{"id":%q}}`, p.ID)
			req, err := http.NewRequestWithContext(ctx, "POST", f.h.url+"/api/v1/billing/yookassa/webhook", strings.NewReader(body))
			if err != nil {
				t.Fatal("construct callback request")
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal("send callback request")
			}
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
			if res.StatusCode != tt.callback {
				t.Errorf("callback status = %d, want %d", res.StatusCode, tt.callback)
			}
			assertState()
		})
	}
}
