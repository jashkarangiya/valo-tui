# valo-tui

> Valorant esports in your terminal.

`valo-tui` is a read-only terminal UI for tracking live and historical Valorant
esports — the four Tier-1 regional leagues (Americas, EMEA, Pacific, China),
international events (Masters / Champions), and full per-map scoreboards — served
from a cached backend so the UI is fast and never rate-limited.

Built in Go on the [Charm](https://charm.land) stack: **Bubble Tea v2** for the
runtime, **Lip Gloss v2** for styling, and **Wish v2** to serve the same TUI
over SSH.

## Architecture

A decoupled **worker / cache** design. TUI clients only ever read SQLite; a
separate fetcher polls [vlr.gg](https://vlr.gg) and writes JSON blobs into a
`kv` table.

```
users ──ssh / local──▶ valo-tui (Bubble Tea) ──reads──▶ SQLite cache ◀──writes── fetcher
```

The cache lives at `~/.cache/valo-tui/cache.db` by default; override with the
`VALO_TUI_DB` environment variable.

## Quick start

```bash
# Build the binaries
go build -o bin/ ./cmd/...

# Populate the cache from vlr.gg (one-shot), or seed sample data offline
go run ./cmd/valo-fetcher --once     # live scrape
go run ./cmd/valo-seed               # offline sample data

# Run the TUI locally
go run ./cmd/valo-tui
```

### Keep the cache fresh (fetcher)

```bash
# Poll vlr.gg on per-key cadences (live fast, results/events/detail slower).
go run ./cmd/valo-fetcher --watch --interval 30s

# Override any cadence per deployment:
go run ./cmd/valo-fetcher --watch \
  --interval 20s --series-interval 45s --results-interval 5m --events-interval 20m
```

Live scores, completed results, the events list and per-event match lists, and
the per-match broadcast detail (`series:{id}`) are each refreshed on their own
ticker. The TUI re-reads the visible screen every 15s and shows a freshness
indicator (`↻ 42s ago`) in the rail, flipping to a `⚠ stale` / `⚠ fetch
failing` warning when the fetcher falls behind or errors.

### Serve over SSH (Wish)

```bash
ssh-keygen -t ed25519 -f .ssh/id_ed25519 -N ""   # one-time host key
go run ./cmd/valo-tui-ssh                          # listens on :23234
# from another terminal:
ssh -p 23234 localhost
```

## Layout

```
cmd/
  valo-tui/       local TUI entrypoint
  valo-tui-ssh/   Wish SSH server (per-connection tea.Program)
internal/
  app/            root model — event-first routing shell
  screens/        one model per screen (splash, global_live, …)
  widgets/        sidebar, match_line
  styles/         palette + lipgloss styles
  data/           read-side SQLite cache
  vlr/            vlr.gg scraper (matches, events, event matches, match detail)
```

## Navigation

| Key       | Action                              |
| --------- | ----------------------------------- |
| `h`       | Home                                |
| `e`       | Events                              |
| `l`       | Global live dashboard               |
| `a`       | About                               |
| `↑` / `↓` | Move through the nav rail           |
| `Enter`   | Open the focused page / drill in    |
| `Esc`     | Back to the nav rail                |
| `q`       | Quit                                |

Once an event is focused, its sub-pages (overview / results / fixtures /
standings / bracket / teams) become reachable.

## Status

Full feature parity with the original Python TUI: home, events, about,
global-live dashboard, the match-detail broadcast view (hero score, series
momentum, per-map scoreboards grouped by agent role, round momentum), and the
event sub-pages — overview, results, fixtures, standings, bracket (ASCII
double-elim tree) and teams — all reading from the SQLite cache, plus the Wish
SSH server.

The **vlr.gg fetcher** (`internal/vlr` + `cmd/valo-fetcher`) is live: it scrapes
the matches/results/events listings, each active event's match list, and the
full per-match scoreboard (vetoes, per-map stats, round momentum) into the
cache. `cmd/valo-seed` remains for offline/demo use.

## Roadmap

1. ~~Go scaffold · SSH server · splash + global live~~ ✅
2. ~~Flat screens (home, events, about, match detail)~~ ✅
3. ~~Event sub-pages (overview, results, fixtures, standings, bracket, teams)~~ ✅
4. ~~Build the vlr.gg fetcher in Go (`internal/vlr` + `cmd/valo-fetcher`)~~ ✅
5. Deploy: long-running fetcher + shared-SSH TUI host.
