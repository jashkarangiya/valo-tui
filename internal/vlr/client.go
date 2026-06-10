package vlr

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	defaultBase = "https://www.vlr.gg"
	// Honest, identifiable User-Agent. vlr.gg serves it normally (verified), and
	// unlike a spoofed browser string it lets them see who we are and reach the
	// project rather than silently blocking an anonymous scraper.
	userAgent = "valo-tui/0.2 (+https://github.com/jashkarangiya/valo-tui)"

	// minInterval is the floor between any two requests. vlr.gg publishes no
	// Crawl-delay, so we self-impose one: a burst (e.g. ~40 event pages) trickles
	// out politely instead of hammering the site. ~1.5s drains a 40-page cycle in
	// about a minute, comfortably inside its 15-minute cadence.
	minInterval = 1500 * time.Millisecond

	// maxRetries bounds transient-failure retries before a fetch gives up and the
	// caller logs/skips. We back off between attempts so an unhappy server gets
	// breathing room rather than a retry storm.
	maxRetries = 3
)

// Client is a polite vlr.gg HTTP client: identifiable, rate-limited, and
// backing off on transient errors.
type Client struct {
	base string
	http *http.Client

	mu   sync.Mutex // serialises the rate gate
	next time.Time  // earliest time the next request may start
}

// New returns a client with a sane timeout.
func New() *Client {
	return &Client{
		base: defaultBase,
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// NewWithBase returns a client pointed at a custom base URL, for tests that
// serve fixtures from a local httptest server.
func NewWithBase(base string) *Client {
	c := New()
	c.base = base
	return c
}

// get fetches a path (e.g. "/matches") and returns the response body. It waits
// out the rate limiter before each attempt and retries transient failures
// (429 / 5xx / network) with backoff, respecting Retry-After when present.
func (c *Client) get(path string) (io.ReadCloser, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		c.wait()

		body, retryAfter, retryable, err := c.attempt(path)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retryable || attempt == maxRetries {
			return nil, err
		}
		d := retryAfter
		if d <= 0 {
			d = backoff(attempt)
		}
		time.Sleep(d)
	}
	return nil, lastErr
}

// attempt makes one request. retryable reports whether err is worth retrying.
func (c *Client) attempt(path string) (body io.ReadCloser, retryAfter time.Duration, retryable bool, err error) {
	req, err := http.NewRequest(http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, 0, false, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, true, err // network/timeout: worth a retry
	}
	if resp.StatusCode == http.StatusOK {
		return resp.Body, 0, false, nil
	}
	ra := parseRetryAfter(resp.Header.Get("Retry-After"))
	resp.Body.Close()
	// 429 (rate limited) and 5xx are transient; 4xx (e.g. 404) are not.
	transient := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
	return nil, ra, transient, fmt.Errorf("vlr.gg %s: status %d", path, resp.StatusCode)
}

// wait blocks until the rate-limit floor has elapsed, then reserves the next
// slot. Holding the lock across the sleep serialises concurrent callers into a
// polite single-file queue.
func (c *Client) wait() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if now := time.Now(); now.Before(c.next) {
		time.Sleep(c.next.Sub(now))
	}
	c.next = time.Now().Add(minInterval)
}

// backoff is exponential: 1s, 2s, 4s, …
func backoff(attempt int) time.Duration {
	return time.Duration(1<<attempt) * time.Second
}

// parseRetryAfter reads a Retry-After header given in seconds (the HTTP-date
// form is rare here and treated as "use our own backoff").
func parseRetryAfter(v string) time.Duration {
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return 0
}
