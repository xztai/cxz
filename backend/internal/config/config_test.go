package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	cases := []struct {
		name  string
		env   map[string]string
		valid bool
	}{
		{"local", nil, true},
		{"test shop", map[string]string{"YOOKASSA_SHOP_ID": "1531842", "YOOKASSA_API_KEY": "test_key"}, true},
		{"missing shop", map[string]string{"YOOKASSA_API_KEY": "test_key"}, false},
		{"missing key", map[string]string{"YOOKASSA_SHOP_ID": "1531842"}, false},
		{"live key", map[string]string{"YOOKASSA_SHOP_ID": "1531842", "YOOKASSA_API_KEY": "live_key"}, false},
		{"invalid shop", map[string]string{"YOOKASSA_SHOP_ID": "not-number", "YOOKASSA_API_KEY": "test_key"}, false},
		{"https", map[string]string{"APP_ORIGIN": "https://example.com", "COOKIE_SECURE": "true"}, true},
		{"https insecure", map[string]string{"APP_ORIGIN": "https://example.com"}, false},
		{"http secure", map[string]string{"COOKIE_SECURE": "true"}, false},
		{"origin path", map[string]string{"APP_ORIGIN": "https://example.com/path"}, false},
		{"origin query", map[string]string{"APP_ORIGIN": "https://example.com?token=x"}, false},
		{"origin user", map[string]string{"APP_ORIGIN": "https://user@example.com"}, false},
		{"bad bool", map[string]string{"TRUST_PROXY": "yes"}, false},
		{"bad cookie bool", map[string]string{"COOKIE_SECURE": "0"}, false},
		{"bad tls bool", map[string]string{"SMTP_INSECURE": "1"}, false},
		{"bad port", map[string]string{"SMTP_PORT": "65536"}, false},
		{"bad addr", map[string]string{"HTTP_ADDR": "8081"}, false},
		{"bad database", map[string]string{"DATABASE_URL": "file:///secret"}, false},
		{"empty database", map[string]string{"DATABASE_URL": ""}, false},
		{"bad timeout", map[string]string{"REQUEST_TIMEOUT": "1m"}, false},
		{"timeout", map[string]string{"REQUEST_TIMEOUT": "500ms"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{"DATABASE_URL", "APP_ORIGIN", "COOKIE_SECURE", "TRUST_PROXY", "SMTP_INSECURE", "SMTP_PORT", "HTTP_ADDR", "REQUEST_TIMEOUT", "SMTP_HOST", "SMTP_FROM", "OUTBOX_KEYS", "YOOKASSA_SHOP_ID", "YOOKASSA_API_KEY"} {
				t.Setenv(key, "")
			}
			t.Setenv("DATABASE_URL", "postgres://test:password@localhost/test")
			t.Setenv("COOKIE_SECURE", "false")
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			cfg, err := Load()
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
			if tc.name == "timeout" && cfg.RequestTimeout != 500*time.Millisecond {
				t.Fatal("timeout ignored")
			}
		})
	}
}
