# Contributing to valo-tui

Thanks for taking a look! `valo-tui` is a young project. That means it is moving
fast and almost certainly has rough edges and bugs. If you find one, please
don't assume it's just you: open an issue or send a PR. Both are very welcome,
and small contributions (a typo, a clearer error message, a flaky test) count
just as much as big ones.

## Ways to help

- **Report a bug.** Tell us what you did, what you expected, and what happened.
  A terminal size, the command you ran, and any log output go a long way.
- **Suggest a feature.** Open an issue describing the screen or data you'd like.
- **Send a PR.** Fixes, new screens, parser improvements, docs, tests: all good.

## Dev setup

You need Go 1.26+ (the SQLite driver is pure Go, so there is no CGO toolchain to
install).

```bash
# Clone and build all four binaries
git clone https://github.com/jashkarangiya/valo-tui
cd valo-tui
go build -o bin/ ./cmd/...

# Run the full test suite (parsers, cache, app)
go test ./...

# Populate the cache, then run the TUI
go run ./cmd/valo-seed        # offline sample data (no network)
go run ./cmd/valo-fetcher --once   # OR a one-shot live scrape from vlr.gg
go run ./cmd/valo-tui
```

The four binaries:

- `cmd/valo-fetcher` scrapes vlr.gg and writes the SQLite cache.
- `cmd/valo-tui` is the local TUI (read-only over the cache).
- `cmd/valo-tui-ssh` serves that same TUI over SSH with Wish.
- `cmd/valo-seed` fills the cache with sample data so you can work offline.

## Project layout

See the "Project layout" section of the [README](README.md). In short: scraping
lives in `internal/vlr`, the read-side cache in `internal/data`, the UI shell in
`internal/app`, and one model per screen in `internal/screens`.

## Working on the scraper

The vlr.gg parsers are tested against saved HTML fixtures in
`internal/vlr/testdata`. When you change a parser:

1. Add or update a fixture and a golden test next to it.
2. Run `go test ./internal/vlr/...`.
3. If your change alters the *shape* of what gets cached, bump
   `vlr.CacheVersion`. On the next start the fetcher will notice the stale stamp,
   wipe the `kv` table, and repopulate through the new parsers, so nobody has to
   clear the DB by hand.

## The one hard rule: be a good citizen to vlr.gg

There is exactly **one** shared worker that ever touches vlr.gg, never per-user
fetching. Please keep it that way. Any networking change should preserve:

- an honest, identifiable `User-Agent`,
- the ~1.5s floor between all requests,
- exponential backoff that honours `Retry-After`,
- conservative per-feed cadences.

If a feature seems to need more requests, open an issue first so we can find a
cache-friendly shape together.

## Style & PR checklist

- Match the surrounding code: small, focused functions and short doc comments.
- Run `gofmt` (or `go fmt ./...`) and `go vet ./...` before pushing.
- Make sure `go test ./...` passes.
- Keep commits focused and write a short, plain description of the change.
- New behaviour deserves a test where it's reasonable to add one.

That's it. Open the PR, and thanks for helping out.
