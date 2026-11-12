package config

import (
	"cmp"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL    string
	OutboxKeys     string
	RequestTimeout time.Duration
	Origin         string
	Addr           string
	SecureCookies  bool
	TrustProxy     bool
	SMTPHost       string
	SMTPPort       string
	SMTPUsername   string
	SMTPPassword   string
	SMTPFrom       string
	SMTPInsecure   bool
	S3Endpoint     string
	S3AccessKey    string
	S3SecretKey    string
	S3Bucket       string
	S3Secure       bool
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		S3Endpoint:     os.Getenv("S3_ENDPOINT"),
		S3AccessKey:    os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:    os.Getenv("S3_SECRET_KEY"),
		S3Bucket:       cmp.Or(os.Getenv("S3_BUCKET"), "etheyl-media"),
		S3Secure:       os.Getenv("S3_SECURE") != "false",
		OutboxKeys:     os.Getenv("OUTBOX_KEYS"),
		RequestTimeout: 5 * time.Second,
		Origin:         strings.TrimRight(cmp.Or(os.Getenv("APP_ORIGIN"), "http://localhost:8080"), "/"),
		Addr:           cmp.Or(os.Getenv("HTTP_ADDR"), ":8081"),
		SecureCookies:  os.Getenv("COOKIE_SECURE") != "false",
		TrustProxy:     os.Getenv("TRUST_PROXY") == "true",
		SMTPHost:       cmp.Or(os.Getenv("SMTP_HOST"), "localhost"),
		SMTPPort:       cmp.Or(os.Getenv("SMTP_PORT"), "1025"),
		SMTPUsername:   os.Getenv("SMTP_USERNAME"), SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     cmp.Or(os.Getenv("SMTP_FROM"), "etheyl <noreply@etheyl.local>"),
		SMTPInsecure: os.Getenv("SMTP_INSECURE") == "true",
	}
	for name, target := range map[string]*bool{"COOKIE_SECURE": &c.SecureCookies, "TRUST_PROXY": &c.TrustProxy, "SMTP_INSECURE": &c.SMTPInsecure, "S3_SECURE": &c.S3Secure} {
		if value := os.Getenv(name); value != "" {
			if value != "true" && value != "false" {
				return c, invalid(name + " must be true or false")
			}
			*target = value == "true"
		}
	}
	if value := os.Getenv("REQUEST_TIMEOUT"); value != "" {
		d, err := time.ParseDuration(value)
		if err != nil || d < 100*time.Millisecond || d > 15*time.Second {
			return c, invalid("REQUEST_TIMEOUT must be between 100ms and 15s")
		}
		c.RequestTimeout = d
	}
	if c.S3Endpoint != "" && (c.S3AccessKey == "" || c.S3SecretKey == "") {
		return c, invalid("S3 credentials are required")
	}
	db, err := url.Parse(c.DatabaseURL)
	if err != nil || (db.Scheme != "postgres" && db.Scheme != "postgresql") || db.Host == "" {
		return c, invalid("DATABASE_URL must be a PostgreSQL URL")
	}
	port, err := strconv.Atoi(c.SMTPPort)
	if err != nil || port < 1 || port > 65535 {
		return c, invalid("SMTP_PORT must be between 1 and 65535")
	}
	if _, _, err := net.SplitHostPort(c.Addr); err != nil {
		return c, invalid("HTTP_ADDR must contain host and port")
	}
	if c.DatabaseURL == "" {
		return c, invalid("DATABASE_URL is required")
	}
	u, err := url.Parse(c.Origin)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return c, invalid("APP_ORIGIN must be an http(s) origin without a path")
	}
	if u.Scheme == "http" && c.SecureCookies {
		return c, invalid("HTTP requires COOKIE_SECURE=false; use HTTPS outside local development")
	}
	if u.Scheme == "https" && !c.SecureCookies {
		return c, invalid("HTTPS requires secure cookies")
	}
	return c, nil
}

// Error contains only fixed validation text; safe for operational logs.
type Error struct{ Message string }

func (e *Error) Error() string     { return e.Message }
func invalid(message string) error { return &Error{Message: message} }
