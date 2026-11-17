package billing

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

var providerID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)
var ErrProvider = errors.New("payment provider unavailable")

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}
type Confirmation struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url,omitempty"`
	URL       string `json:"confirmation_url,omitempty"`
}
type CreateRequest struct {
	Amount       Amount            `json:"amount"`
	Capture      bool              `json:"capture"`
	Confirmation *Confirmation     `json:"confirmation,omitempty"`
	SaveMethod   bool              `json:"save_payment_method,omitzero"`
	MethodID     string            `json:"payment_method_id,omitempty"`
	Description  string            `json:"description"`
	Metadata     map[string]string `json:"metadata"`
}
type RemotePayment struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Paid      bool   `json:"paid"`
	Test      bool   `json:"test"`
	Amount    Amount `json:"amount"`
	Recipient struct {
		AccountID string `json:"account_id"`
	} `json:"recipient"`
	Metadata     map[string]string `json:"metadata"`
	Confirmation Confirmation      `json:"confirmation"`
	Method       struct {
		ID    string `json:"id"`
		Saved bool   `json:"saved"`
	} `json:"payment_method"`
	Cancellation struct {
		Reason string `json:"reason"`
	} `json:"cancellation_details"`
}
type Provider interface {
	Create(context.Context, string, string) (RemotePayment, error)
	Get(context.Context, string) (RemotePayment, error)
}
type Client struct {
	shop, key, base string
	http            *http.Client
}

func NewClient(shop, key string) *Client {
	return &Client{shop: shop, key: key, base: "https://api.yookassa.ru/v3", http: &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Provider error payloads and URLs may contain payment data; never expose them.
type Rejected struct{}

func (*Rejected) Error() string { return "payment request rejected" }
func (c *Client) call(ctx context.Context, method, path, key, body string) (RemotePayment, error) {
	var out RemotePayment
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewBufferString(body))
	if err != nil {
		return out, ErrProvider
	}
	req.SetBasicAuth(c.shop, c.key)
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotence-Key", key)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return out, ErrProvider
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if res.StatusCode >= 400 && res.StatusCode < 500 && res.StatusCode != 429 && res.StatusCode != 408 {
			return out, &Rejected{}
		}
		return out, ErrProvider
	}
	if err := json.UnmarshalRead(io.LimitReader(res.Body, 1<<20), &out); err != nil || !providerID.MatchString(out.ID) {
		return out, ErrProvider
	}
	return out, nil
}
func (c *Client) Create(ctx context.Context, key, body string) (RemotePayment, error) {
	return c.call(ctx, "POST", "/payments", key, body)
}
func (c *Client) Get(ctx context.Context, id string) (RemotePayment, error) {
	if !providerID.MatchString(id) {
		return RemotePayment{}, ErrProvider
	}
	return c.call(ctx, "GET", "/payments/"+id, "", "")
}
func money(n int64) string {
	return strconv.FormatInt(n/100, 10) + "." + string([]byte{byte('0' + n%100/10), byte('0' + n%10)})
}
func safeConfirmation(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return ""
	}
	switch u.Hostname() {
	case "yoomoney.ru", "yookassa.ru":
		return raw
	}
	return ""
}

// Preserve the original billing day across February and other short months.
func NextMonth(start time.Time, anchor int) time.Time {
	start = start.UTC()
	first := time.Date(start.Year(), start.Month()+1, 1, start.Hour(), start.Minute(), start.Second(), start.Nanosecond(), time.UTC)
	last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return first.AddDate(0, 0, min(anchor, last)-1)
}
