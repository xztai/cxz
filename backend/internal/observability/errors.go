package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"

	"etheyl.local/backend/internal/config"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrorFields deliberately excludes error messages, query parameters and PII.
func ErrorFields(err error) []any {
	if e, ok := errors.AsType[*config.Error](err); ok {
		return []any{"category", "configuration", "reason", e.Message}
	}
	category := "internal"
	state := ""
	if e, ok := errors.AsType[*pgconn.PgError](err); ok {
		category, state = "database", e.Code
	} else if errors.Is(err, context.DeadlineExceeded) {
		category = "timeout"
	} else if errors.Is(err, context.Canceled) {
		category = "canceled"
	} else if pgconn.Timeout(err) {
		category = "timeout"
	}
	return []any{"category", category, "sqlstate", state, "error_type", fmt.Sprintf("%T", err)}
}

// Callers omits panic values, function arguments and locals (which may contain secrets).
func PanicStack() string {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var out strings.Builder
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&out, "%s %s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return out.String()
}

func Failure(log *slog.Logger, operation string, err error) {
	log.Error("operation failed", append([]any{"operation", operation}, ErrorFields(err)...)...)
}
