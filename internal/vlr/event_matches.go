package vlr

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// EventMatches scrapes /event/matches/{id} into the teams[]-shaped rows the
// bracket/standings reconstruction consumes.
func (c *Client) EventMatches(eventID int) ([]EventMatch, error) {
	body, err := c.get(fmt.Sprintf("/event/matches/%d", eventID))
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return parseEventMatches(body)
}

func parseEventMatches(r io.Reader) ([]EventMatch, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var out []EventMatch
	doc.Find("a.match-item").Each(func(_ int, card *goquery.Selection) {
		href, _ := card.Attr("href")
		m := matchIDRe.FindStringSubmatch(href)
		if m == nil {
			return
		}
		id, _ := strconv.Atoi(m[1])

		var teams []EventTeam
		card.Find(".match-item-vs-team").EachWithBreak(func(_ int, te *goquery.Selection) bool {
			name := norm(te.Find(".match-item-vs-team-name .text-of").First().Text())
			if name == "" {
				name = norm(te.Find(".match-item-vs-team-name").First().Text())
			}
			if name == "" {
				return len(teams) < 2
			}
			var score *int
			if n, err := strconv.Atoi(strings.TrimSpace(te.Find(".match-item-vs-team-score").First().Text())); err == nil {
				score = &n
			}
			teams = append(teams, EventTeam{
				Name:     name,
				Score:    score,
				IsWinner: te.HasClass("mod-winner"),
			})
			return len(teams) < 2
		})
		if len(teams) != 2 {
			return
		}

		// Status from the .match-item-eta .ml class.
		status := "upcoming"
		if cls, ok := card.Find(".match-item-eta .ml").First().Attr("class"); ok {
			switch {
			case strings.Contains(cls, "mod-completed"):
				status = "completed"
			case strings.Contains(cls, "mod-live"), strings.Contains(cls, "mod-ongoing"):
				status = "live"
			}
		}

		out = append(out, EventMatch{
			MatchID: id,
			Teams:   teams,
			Phase:   norm(card.Find(".match-item-event-series").First().Text()),
			Status:  status,
			Time:    norm(card.Find(".match-item-time").First().Text()),
		})
	})
	return out, nil
}
