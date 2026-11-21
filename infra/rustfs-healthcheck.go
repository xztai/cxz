package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: rustfs-healthcheck URL")
		os.Exit(2)
	}
	client := &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Get(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "RustFS readiness request failed")
		os.Exit(1)
	}
	_ = response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		fmt.Fprintf(os.Stderr, "RustFS readiness returned HTTP %d\n", response.StatusCode)
		os.Exit(1)
	}
}
