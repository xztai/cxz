package main

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"etheyl.local/backend/internal/billing"
)

func TestReconcileReporting(t *testing.T) {
	technical := errors.New("provider verification failed")
	for _, tt := range []struct {
		name    string
		outcome billing.ReconciliationOutcome
		err     error
		fails   bool
		field   string
	}{
		{name: "succeeded", outcome: billing.ReconciliationSucceeded, field: "succeeded"},
		{name: "canceled", outcome: billing.ReconciliationCanceled, field: "canceled"},
		{name: "already succeeded", outcome: billing.ReconciliationAlreadySucceeded, field: "already_succeeded"},
		{name: "already canceled", outcome: billing.ReconciliationAlreadyCanceled, field: "already_canceled"},
		{name: "unmatched", outcome: billing.ReconciliationUnmatched, fails: true, field: "unmatched"},
		{name: "pending", outcome: billing.ReconciliationPending, fails: true, field: "pending"},
		{name: "review", outcome: billing.ReconciliationReview, fails: true, field: "review"},
		{name: "technical failure", err: technical, fails: true},
		{name: "unknown outcome", outcome: "unknown", fails: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&output, nil))
			err := reportReconciliation(logger, tt.outcome, tt.err)
			if (err != nil) != tt.fails || (tt.err != nil && !errors.Is(err, tt.err)) {
				t.Errorf("report error = %v, want failure %t", err, tt.fails)
			}
			if tt.field == "" {
				if output.Len() != 0 {
					t.Error("report logged a result on failure")
				}
			} else if !strings.HasSuffix(output.String(), " outcome="+tt.field+"\n") || strings.Count(output.String(), "\n") != 1 {
				t.Errorf("want one report with outcome=%s", tt.field)
			}
		})
	}
}
