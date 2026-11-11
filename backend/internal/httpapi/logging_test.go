package httpapi

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"etheyl.local/backend/internal/config"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSafeRequestDiagnostics(t *testing.T) {
	var logs bytes.Buffer
	s := &Server{log: slog.New(slog.NewJSONHandler(&logs, nil)), cfg: config.Config{}}
	w := httptest.NewRecorder()
	w.Header().Set("X-Request-ID", "correlation")
	s.writeError(w, &pgconn.PgError{Code: "23505", Message: "private person@example.com", Detail: "password and token"})
	if !strings.Contains(logs.String(), `"sqlstate":"23505"`) || strings.Contains(logs.String(), "person@example.com") || strings.Contains(logs.String(), "password and token") {
		t.Fatal("unsafe or incomplete diagnostics", logs.String())
	}
	s.writeError(httptest.NewRecorder(), context.DeadlineExceeded)
	if !strings.Contains(logs.String(), `"category":"timeout"`) {
		t.Fatal("timeout unclassified")
	}
	logs.Reset()
	// A nil pool provokes a panic without a request body ever entering the logs.
	recorder := httptest.NewRecorder()
	s.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/api/v1/health?token=never-log-this", nil))
	if recorder.Code != 500 || recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing panic response/correlation")
	}
	if strings.Contains(logs.String(), "never-log-this") || !strings.Contains(logs.String(), "stack") {
		t.Fatal("panic diagnostics unsafe or missing")
	}
	for line := range strings.SplitSeq(strings.TrimSpace(logs.String()), "\n") {
		var value map[string]any
		if err := json.Unmarshal([]byte(line), &value); err != nil {
			t.Fatal(err)
		}
	}
}
