package mailqueue

import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"etheyl.local/backend/internal/observability"
	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Monitor struct {
	mu          sync.Mutex
	heartbeat   time.Time
	lastPoll    time.Time
	lastSuccess time.Time
}

func NewMonitor() *Monitor { return &Monitor{} }
func (m *Monitor) Beat()   { m.mu.Lock(); defer m.mu.Unlock(); m.heartbeat = time.Now() }
func (m *Monitor) Refresh(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	stats, err := store.New(pool).QueueStats(ctx)
	if err != nil {
		observability.Failure(logger, "queue_stats", err)
		return
	}
	m.mu.Lock()
	m.lastPoll = time.Now()
	m.mu.Unlock()
	logger.Info("mail queue state", "pending", stats.Pending, "failed", stats.Failed, "oldest_seconds", stats.OldestSeconds)
	if stats.Failed > 0 || stats.OldestSeconds > 300 {
		logger.Error("mail delivery degraded", "pending", stats.Pending, "failed", stats.Failed, "oldest_seconds", stats.OldestSeconds)
	}
}

// Status is served only by the internal worker, separately from process liveness.
func Status(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, m *Monitor) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	stats, err := store.New(pool).QueueStats(ctx)
	m.mu.Lock()
	heartbeat, lastPoll, lastSuccess := m.heartbeat, m.lastPoll, m.lastSuccess
	m.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	status := "ok"
	if err != nil || time.Since(heartbeat) > 30*time.Second || time.Since(lastPoll) > 30*time.Second || stats.Failed > 0 || stats.OldestSeconds > 300 {
		w.WriteHeader(503)
		status = "degraded"
	}
	_ = json.MarshalWrite(w, map[string]any{"status": status, "pending": stats.Pending, "failed": stats.Failed, "oldestSeconds": stats.OldestSeconds, "heartbeat": heartbeat, "lastPoll": lastPoll, "lastSuccess": lastSuccess})
}

type monitoredSender struct {
	Sender
	monitor *Monitor
}

func (s monitoredSender) Send(ctx context.Context, item store.MailOutbox) error {
	err := s.Sender.Send(ctx, item)
	if err == nil {
		s.monitor.mu.Lock()
		s.monitor.lastSuccess = time.Now()
		s.monitor.mu.Unlock()
	}
	return err
}
