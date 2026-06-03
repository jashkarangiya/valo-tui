package vlr

import (
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var matchIDRe = regexp.MustCompile(`^/(\d+)`)

// norm collapses all runs of whitespace to single spaces and trims — needed
// because goquery's .Text() preserves the source HTML's whitespace, unlike
// BeautifulSoup's get_text(strip=True).
func norm(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// parseMatches parses the /matches listing page into Match rows. Ported from
// vlrdevapi.matches._parser._parse_matches (team-id/country batch-fetching
// omitted — our models don't use them).
func parseMatches(r io.Reader) ([]Match, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var matches []Match
	doc.Find("a.match-item").Each(func(_ int, node *goquery.Selection) {
		href, _ := node.Attr("href")
		m := matchIDRe.FindStringSubmatch(href)
		if m == nil {
			return
		}
		id, _ := strconv.Atoi(m[1])

		// Team names (first two blocks).
		var names []string
		node.Find(".match-item-vs-team").EachWithBreak(func(_ int, tb *goquery.Selection) bool {
			name := norm(tb.Find(".match-item-vs-team-name .text-of").First().Text())
			if name == "" {
				name = norm(tb.Find(".match-item-vs-team-name").First().Text())
			}
			names = append(names, name)
			return len(names) < 2
		})
		if len(names) < 2 || names[0] == "" || names[1] == "" {
			return
		}

		// Scores ("-" or empty ⇒ none).
		var scores []*int
		node.Find(".match-item-vs-team-score").Each(func(_ int, s *goquery.Selection) {
			t := strings.TrimSpace(s.Text())
			if n, err := strconv.Atoi(t); err == nil {
				scores = append(scores, &n)
			} else {
				scores = append(scores, nil)
			}
		})
		score := func(i int) *int {
			if i < len(scores) {
				return scores[i]
			}
			return nil
		}

		// Event = combined text minus the series sub-label.
		eventSel := node.Find(".match-item-event").First()
		series := norm(eventSel.Find(".match-item-event-series").Text())
		event := strings.TrimSpace(strings.Replace(norm(eventSel.Text()), series, "", 1))

		timeText := norm(node.Find(".match-item-time").First().Text())
		if timeText == "" {
			timeText = norm(node.Find(".match-item-eta").First().Text())
		}

		status := strings.ToUpper(norm(node.Find(".match-item-status").First().Text()))
		if status == "" {
			status = strings.ToUpper(norm(node.Find(".ml-status").First().Text()))
		}

		matches = append(matches, Match{
			MatchID:    id,
			Team1:      Team{Name: names[0], Score: score(0)},
			Team2:      Team{Name: names[1], Score: score(1)},
			Event:      event,
			EventPhase: series,
			Status:     classify(status, score(0), score(1)),
			Time:       timeText,
		})
	})
	return matches, nil
}

// classify maps the page's status string into our upcoming|live|completed.
func classify(rawStatus string, s1, s2 *int) string {
	switch {
	case rawStatus == "LIVE":
		return "live"
	case s1 != nil || s2 != nil:
		return "completed"
	default:
		return "upcoming"
	}
}
