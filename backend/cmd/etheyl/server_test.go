package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestShutdownCancelsActiveRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := make(chan struct{})
	finished := make(chan struct{})
	server := httpServer(ctx, "127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(finished)
		w.WriteHeader(503)
	}))
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		resp, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	shutdownCtx, done := context.WithTimeout(t.Context(), time.Second)
	defer done()
	if err := server.Shutdown(shutdownCtx); err != nil {
		t.Fatal("shutdown stalled", err)
	}
	<-finished
	<-clientDone
	if err := <-serveDone; !errors.Is(err, http.ErrServerClosed) {
		t.Fatal(err)
	}
}

func TestDatabaseTimeouts(t *testing.T) {
	for _, tc := range []struct{ command, statement, lock, idle string }{
		{"api", "4000", "2000", "10000"},
		{"worker", "4000", "2000", "20000"},
		{"migrate", "30000", "5000", ""},
	} {
		t.Run(tc.command, func(t *testing.T) {
			pool, err := openPool(t.Context(), "postgres://test:password@localhost/test", tc.command)
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()
			params := pool.Config().ConnConfig.RuntimeParams
			if params["statement_timeout"] != tc.statement || params["lock_timeout"] != tc.lock || params["idle_in_transaction_session_timeout"] != tc.idle {
				t.Fatal("incorrect timeouts", params)
			}
		})
	}
}
