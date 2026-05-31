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

# Run the TUI locally
go run ./cmd/valo-tui
```

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
  vlr/            vlr.gg client (planned — the Go fetcher)
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

Working: the read-side data layer, theme, sidebar, the framed event-first
shell, the splash and global-live screens, and the SSH server. The remaining
screens render a placeholder until ported, and the vlr.gg fetcher
(`internal/vlr` + a `cmd/valo-fetcher`) is the next milestone — until then the
cache must be populated externally.

## Roadmap

1. ~~Go scaffold · SSH server · splash + global live~~ ✅
2. Port the flat screens (home, events, about, match detail, standings).
3. Event sub-pages (results / fixtures / bracket / teams) + live dot + countdown.
4. Build the vlr.gg fetcher in Go (`internal/vlr` + `cmd/valo-fetcher`).
