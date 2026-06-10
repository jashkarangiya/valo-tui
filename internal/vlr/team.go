package vlr

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

// Roster is a team's full lineup scraped from /team/{id}. Marshals to the
// team:{id} cache key the roster overlay reads.
type Roster struct {
	TeamID  int      `json:"team_id"`
	Team    string   `json:"team"`
	Members []Member `json:"members"`
}

// Member is one roster entry. Role is "" for active players, else a staff label
// ("head coach", "manager", …).
type Member struct {
	ID      int    `json:"id"`
	Alias   string `json:"alias"`
	Name    string `json:"name"`
	Country string `json:"country"`
	Role    string `json:"role"`
	Captain bool   `json:"captain"`
}

// playerIDRe pulls the numeric id out of a /player/{id}/{slug} href.
var playerIDRe = regexp.MustCompile(`^/player/(\d+)`)

// TeamRoster scrapes /team/{id} into the lineup the roster overlay renders.
func (c *Client) TeamRoster(teamID int) (Roster, error) {
	body, err := c.get(fmt.Sprintf("/team/%d", teamID))
	if err != nil {
		return Roster{}, err
	}
	defer body.Close()
	r, err := parseTeam(body)
	r.TeamID = teamID
	return r, err
}

func parseTeam(rd io.Reader) (Roster, error) {
	doc, err := goquery.NewDocumentFromReader(rd)
	if err != nil {
		return Roster{}, err
	}

	r := Roster{Team: norm(doc.Find(".team-header .wf-title").First().Text())}

	doc.Find(".team-roster-item").Each(func(_ int, it *goquery.Selection) {
		link := it.Find(`a[href^="/player/"]`).First()
		href, ok := link.Attr("href")
		if !ok {
			return
		}
		var id int
		if m := playerIDRe.FindStringSubmatch(href); m != nil {
			id, _ = strconv.Atoi(m[1])
		}
		alias := it.Find(".team-roster-item-name-alias").First()
		m := Member{
			ID:      id,
			Alias:   norm(alias.Text()),
			Name:    norm(it.Find(".team-roster-item-name-real").First().Text()),
			Role:    norm(it.Find(".team-roster-item-name-role").First().Text()),
			Country: flagCountry(alias),
			Captain: alias.Find(".fa-star").Length() > 0,
		}
		if m.Alias == "" && m.Name == "" {
			return
		}
		r.Members = append(r.Members, m)
	})
	return r, nil
}

// flagCountry reads the two-letter country code from a `flag mod-xx` icon.
func flagCountry(sel *goquery.Selection) string {
	cls, ok := sel.Find(".flag").First().Attr("class")
	if !ok {
		return ""
	}
	if m := flagRe.FindStringSubmatch(cls); m != nil {
		return m[1]
	}
	return ""
}

var flagRe = regexp.MustCompile(`mod-([a-z]{2})\b`)
