package vlr

import (
	"os"
	"strings"
	"testing"
)

func TestParseEvents(t *testing.T) {
	f, err := os.Open("testdata/events.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	events, err := parseEvents(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 10 {
		t.Fatalf("expected a full events list, got %d", len(events))
	}
	for _, e := range events {
		if e.ID == 0 || e.Name == "" {
			t.Errorf("event missing id/name: %+v", e)
		}
		switch e.Status {
		case "ongoing", "upcoming", "completed":
		default:
			t.Errorf("unexpected status %q: %+v", e.Status, e)
		}
		// "Dates"/"Prize Pool" labels must be stripped.
		if strings.Contains(e.StartText, "Dates") || strings.Contains(e.Prize, "Prize") {
			t.Errorf("label not stripped: %+v", e)
		}
	}
	t.Logf("parsed %d events", len(events))
}

func TestSplitDateRange(t *testing.T) {
	cases := map[string][2]string{
		"May 26—Jul 28": {"May 26", "Jul 28"},
		"May 25—31":     {"May 25", "31"},
		"Aug 2 - Aug 17": {"Aug 2", "Aug 17"},
		"TBD":           {"TBD", ""},
	}
	for in, want := range cases {
		s, e := splitDateRange(in)
		if s != want[0] || e != want[1] {
			t.Errorf("splitDateRange(%q) = (%q,%q), want (%q,%q)", in, s, e, want[0], want[1])
		}
	}
}
