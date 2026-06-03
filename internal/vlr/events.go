package vlr

import (
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	eventIDRe   = regexp.MustCompile(`/event/(\d+)`)
	flagModRe   = regexp.MustCompile(`mod-([a-z]{2,3})`)
	dateRangeRe = regexp.MustCompile(`\s*[—–-]\s*`) // em / en / hyphen dash
)

// flagRegion maps vlr.gg flag codes to a readable region label; unknown codes
// fall through to the uppercased code.
var flagRegion = map[string]string{
	"us": "Americas", "na": "Americas", "br": "Brazil", "ar": "LATAM", "cl": "LATAM",
	"eu": "EMEA", "tr": "EMEA", "gb": "EMEA",
	"jp": "Japan", "kr": "Korea", "id": "Pacific", "th": "Pacific", "vn": "Pacific",
	"cn": "China", "int": "International",
}

// Events scrapes the /events listing and returns the currently-active (ongoing
// or upcoming) tournaments. Unlike vlrdevapi it does not do per-event date
// lookups — the listing's own date text is enough for our cards.
func (c *Client) Events() ([]Event, error) {
	body, err := c.get("/events")
	if err != nil {
		return nil, err
	}
	defer body.Close()

	all, err := parseEvents(body)
	if err != nil {
		return nil, err
	}
	active := make([]Event, 0, 40)
	for _, e := range all {
		if e.Status == "ongoing" || e.Status == "upcoming" {
			active = append(active, e)
			if len(active) >= 40 {
				break
			}
		}
	}
	return active, nil
}

func parseEvents(r io.Reader) ([]Event, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var out []Event
	doc.Find(".events-container a.event-item[href*='/event/']").Each(func(_ int, c *goquery.Selection) {
		href, _ := c.Attr("href")
		m := eventIDRe.FindStringSubmatch(href)
		if m == nil {
			return
		}
		id, _ := strconv.Atoi(m[1])

		name := norm(c.Find(".event-item-title").First().Text())
		if name == "" {
			return
		}

		// Dates: "May 26—Jul 28" with a trailing "Dates" label to strip.
		dateText := norm(c.Find(".event-item-desc-item.mod-dates .event-item-desc-item-value").First().Text())
		if dateText == "" {
			dateText = norm(c.Find(".event-item-desc-item.mod-dates").First().Text())
		}
		dateText = strings.TrimSpace(strings.ReplaceAll(dateText, "Dates", ""))
		start, end := splitDateRange(dateText)

		prize := strings.TrimSpace(strings.ReplaceAll(
			norm(c.Find(".event-item-desc-item.mod-prize").First().Text()), "Prize Pool", ""))

		status := "upcoming"
		if cls, ok := c.Find(".event-item-desc-item-status").First().Attr("class"); ok {
			switch {
			case strings.Contains(cls, "mod-completed"):
				status = "completed"
			case strings.Contains(cls, "mod-ongoing"):
				status = "ongoing"
			}
		}

		region := ""
		if cls, ok := c.Find(".event-item-desc-item.mod-location .flag").First().Attr("class"); ok {
			if fm := flagModRe.FindStringSubmatch(cls); fm != nil {
				region = flagRegion[fm[1]]
				if region == "" {
					region = strings.ToUpper(fm[1])
				}
			}
		}

		out = append(out, Event{
			ID: id, Name: name, Status: status, Region: region,
			Prize: prize, StartText: start, EndText: end,
		})
	})
	return out, nil
}

// splitDateRange splits "May 26—Jul 28" into ("May 26", "Jul 28").
func splitDateRange(s string) (string, string) {
	if s == "" {
		return "", ""
	}
	parts := dateRangeRe.Split(s, 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(s), ""
}
