# Deploying valo-tui

The goal: users `ssh valo.example.com` and see live VCT data in their terminal,
with data kept fresh **without overloading vlr.gg**.

## The shape

```
                      one host
  ┌──────────────────────────────────────────────┐
  │  valo-fetcher --watch ──writes──▶ cache.db     │
  │     │ (the ONLY thing that hits vlr.gg)        │
  │     │                              ▲           │
  │     ▼                              │ reads      │
  │   vlr.gg                     valo-tui-ssh (Wish)│
  └──────────────────────────────────────┬─────────┘
                                          │ ssh
                            many users ───┘
```

The cache is the whole trick: **request volume to vlr.gg is fixed no matter how
many people connect.** One viewer or a thousand, vlr.gg sees the same single
polite poller. Scaling users is free for the source site.

SQLite runs in WAL mode, so the one writer (fetcher) and many readers (every SSH
session) never block each other.

## Being a good citizen to vlr.gg

Already enforced in code (`internal/vlr/client.go`):

- **One shared fetcher**, never per-user fetching.
- **Identifiable User-Agent** (`valo-tui/… (+repo url)`) — not a spoofed browser,
  so vlr.gg can see who we are and reach us.
- **Rate limit**: a ~1.5s floor between *all* requests, so the ~40-page event
  refresh trickles out over ~a minute instead of bursting.
- **Backoff**: 429 / 5xx / network errors retry with exponential backoff and
  honour `Retry-After`, then give up rather than hammering.
- **Only allowed paths**: robots.txt permits everything we read; we touch
  nothing under its `Disallow` rules.
- **Conservative cadences**: live scores refresh fast, results/events/detail
  much slower. Tune per deployment:

  ```
  valo-fetcher --watch --interval 30s \
    --series-interval 45s --results-interval 5m --events-interval 20m
  ```

If vlr.gg ever asks us to change behaviour, raising `--interval`/`minInterval`
or pausing the daemon is all it takes.

## Setup

```bash
# 1. Build static-ish binaries (pure-Go SQLite, no CGO).
CGO_ENABLED=0 go build -o /usr/local/bin/ ./cmd/valo-fetcher ./cmd/valo-tui-ssh

# 2. Dedicated user + state dir for the shared cache + host key.
sudo useradd --system --home /var/lib/valo-tui --create-home valo
sudo -u valo ssh-keygen -t ed25519 -f /var/lib/valo-tui/.ssh/id_ed25519 -N ""

# 3. Install + start both services.
sudo cp deploy/valo-fetcher.service deploy/valo-tui-ssh.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now valo-fetcher valo-tui-ssh

# 4. Confirm the cache is filling and stays fresh.
sudo journalctl -u valo-fetcher -f
```

Point DNS at the host. Either bind Wish to `:22` (move the host's real sshd to
another port) or keep the default `23234` and tell users `ssh -p 23234 …`.

## Freshness, end to end

- The fetcher refreshes the cache on its tickers and writes a `meta:fetch`
  heartbeat each live cycle.
- The TUI re-reads the visible screen every 15s and shows `↻ 42s ago` in the
  rail, flipping to `⚠ … · stale` or `⚠ fetch failing` if the fetcher falls
  behind or errors — so a dead daemon is visible to users, not silent.
- `systemd Restart=always` brings either service back after a crash/reboot.

## Hardening the front door

The TUI is read-only, so the only real abuse vector is connection count. The
server sets a 15-minute idle timeout; for a public deployment also consider:

- a firewall / fail2ban on the SSH port,
- a reverse proxy or `sshd`-level `MaxStartups` / per-IP connection limits,
- running inside a container or the provided systemd sandbox (already enabled).
