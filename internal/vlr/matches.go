package vlr

// list fetches a match-listing page and parses every row on it.
func (c *Client) list(path string) ([]Match, error) {
	body, err := c.get(path)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return parseMatches(body)
}

// Matches returns the rows on /matches — a mix of live and upcoming. The
// caller partitions by Status. Both lists come from one request.
func (c *Client) Matches() ([]Match, error) { return c.list("/matches") }

// Results returns completed matches from /matches/results (same row parser).
func (c *Client) Results() ([]Match, error) { return c.list("/matches/results") }

// FilterStatus returns only the matches with the given status.
func FilterStatus(matches []Match, status string) []Match {
	out := make([]Match, 0, len(matches))
	for _, m := range matches {
		if m.Status == status {
			out = append(out, m)
		}
	}
	return out
}
