package vlr

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestRateLimit proves the polite floor between requests: the first goes out
// immediately, the next is held back by at least minInterval. Guards the
// vlr.gg-friendliness the deployment relies on.
func TestRateLimit(t *testing.T) {
	var hits []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits = append(hits, time.Now())
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewWithBase(srv.URL)
	for i := 0; i < 2; i++ {
		body, err := c.get("/matches")
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		body.Close()
	}
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}
	if gap := hits[1].Sub(hits[0]); gap < minInterval {
		t.Errorf("requests spaced %v, want >= %v", gap, minInterval)
	}
}

// TestRetryThenSucceed checks transient 5xx responses are retried (with backoff)
// rather than surfaced as failures.
func TestRetryThenSucceed(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		if n == 1 {
			w.WriteHeader(http.StatusBadGateway) // 502 once
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewWithBase(srv.URL)
	body, err := c.get("/matches")
	if err != nil {
		t.Fatalf("expected retry to recover, got %v", err)
	}
	body.Close()
	if n != 2 {
		t.Errorf("expected 2 attempts (1 fail + 1 ok), got %d", n)
	}
}

// TestNoRetryOn404 checks permanent 4xx errors fail fast without retrying.
func TestNoRetryOn404(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewWithBase(srv.URL)
	if _, err := c.get("/nope"); err == nil {
		t.Fatal("expected an error for 404")
	}
	if n != 1 {
		t.Errorf("expected exactly 1 attempt for 404, got %d", n)
	}
}
