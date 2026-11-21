package main

import (
	"strings"
	"testing"
)

func TestSeedGuard(t *testing.T) {
	base := map[string]string{
		"SEED_ALLOW":   "1",
		"DATABASE_URL": "postgres://etheyl@127.0.0.1:5432/etheyl?sslmode=disable",
	}
	for _, c := range []struct{ name, variable, value, want string }{
		{"missing opt-in", "SEED_ALLOW", "", "SEED_ALLOW=1"},
		{"remote database", "DATABASE_URL", "postgres://etheyl@db.example/etheyl", "DATABASE_URL"},
		{"multiple hosts", "DATABASE_URL", "postgres://etheyl@remote.example,localhost/etheyl", "DATABASE_URL"},
		{"overridden hostname", "DATABASE_URL", base["DATABASE_URL"] + "&host=remote.example", "sslmode"},
		{"libpq service", "PGSERVICE", "example", "libpq"},
		{"production database", "DATABASE_URL", "postgres://etheyl@localhost/etheyl_prod", "DATABASE_URL"},
		{"production environment", "APP_ENV", "Production", "APP_ENV"},
		{"secure production cookies", "COOKIE_SECURE", "true", "production"},
		{"live payment key", "YOOKASSA_API_KEY", "live_placeholder", "production"},
		{"remote app", "APP_ORIGIN", "http://example.test", "APP_ORIGIN"},
		{"remote object store", "S3_ENDPOINT", "objects.example:9000", "S3_ENDPOINT"},
	} {
		t.Run(c.name, func(t *testing.T) {
			getenv := func(key string) string {
				if key == c.variable {
					return c.value
				}
				return base[key]
			}
			_, err := loadSettings(getenv)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected rejection with %q, got %v", c.want, err)
			}
		})
	}
}
