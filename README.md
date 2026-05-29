# valo-tui

> Valorant esports in your terminal.

`valo-tui` is a read-only terminal UI for tracking live and historical Valorant
esports. It tracks the four Tier-1 regional leagues (Americas, EMEA, Pacific,
China), international events (Masters / Champions / Game Changers), and full
per-map scoreboards — all from a cached backend so the UI is fast and never
rate-limited.

## Architecture

A decoupled **worker / cache** design. The TUI clients only ever read SQLite;
a separate fetcher process polls [vlr.gg](https://vlr.gg) (via `vlrdevapi`) and
writes JSON blobs into the cache.

```
users ──ssh / browser──▶ Textual app ──reads──▶ SQLite cache ◀──writes── fetcher worker ──▶ vlr.gg
```

## Quick start (local)

```bash
python3.12 -m venv .venv && source .venv/bin/activate
pip install -e .

# 1. Seed the cache once (or run the worker loop continuously):
python worker/fetcher.py --once

# 2. Launch the TUI:
python -m valo_tui
```

The cache lives at `~/.cache/valo-tui/cache.db` by default; override with the
`VALO_TUI_DB` environment variable.

## Navigation

| Key | Action |
| --- | --- |
| `g` | Global live dashboard |
| `m` | Matches list |
| `j` / `k` | Move down / up |
| `Enter` | Drill into a match (scoreboards) |
| `r` | Refresh from cache |
| `Esc` / `q` | Back / quit |

## Zero-install serving

```bash
pip install -e ".[serve]"
python serve/web.py    # browser → http://localhost:8000
python serve/ssh.py    # ssh -p 2222 localhost
```

## Deployment

`docker compose up` brings up the worker, web server, SSH server, and a Caddy
reverse proxy for HTTPS. Point the domain in `Caddyfile` at your host.

## Layout

```
valo_tui/ TUI package (app, screens, styles, data layer)
worker/   fetcher.py — polls vlr.gg → SQLite
serve/    web.py (textual-serve) and ssh.py (asyncssh)
```
