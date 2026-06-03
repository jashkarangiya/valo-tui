package vlr

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBase = "https://www.vlr.gg"
	// A real browser UA — vlr.gg serves bot-y requests differently. Mirrors
	// vlrdevapi's default.
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// Client is a polite vlr.gg HTTP client.
type Client struct {
	base string
	http *http.Client
}

// New returns a client with a sane timeout.
func New() *Client {
	return &Client{
		base: defaultBase,
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// get fetches a path (e.g. "/matches") and returns the response body.
func (c *Client) get(path string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("vlr.gg %s: status %d", path, resp.StatusCode)
	}
	return resp.Body, nil
}
