// Command valo-tui runs the TUI locally, against the same SQLite cache the
// Python worker writes. `go run ./cmd/valo-tui`.
package main

import (
	"log"

	tea "charm.land/bubbletea/v2"

	"github.com/jashkarangiya/valo-tui/internal/app"
)

func main() {
	p := tea.NewProgram(app.New(80, 24))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
