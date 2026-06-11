# Deploying valo-tui

The goal: users `ssh valo.example.com` and see live VCT data in their terminal,
with data kept fresh **without overloading vlr.gg**.

> **On Proxmox?** Use the one-command LXC deploy in [`../proxmox/`](../proxmox/)
> — it builds a hardened, unprivileged container where the only SSH surface is
> the read-only TUI (no shell to reach). This page covers the generic
> bare-metal / VM setup.

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

## Setup (bare `ssh valo.blackpantha.com`, port 22)

This is the "just SSH in, no `-p`" deployment. Wish owns port 22, so the box's
own admin sshd has to move off it first.

### 0. DNS

Add an `A` record (and `AAAA` if you have IPv6) for `valo.blackpantha.com`
pointing at the server's public IP. Confirm it resolves before continuing:

```bash
dig +short valo.blackpantha.com
```

### 1. Move the admin sshd off port 22 — FIRST, carefully

Do this before anything claims :22, and **keep your current SSH session open**
as a safety net until you've verified the new port works.

```bash
# Pick a new admin port, e.g. 2222.
sudo sed -i 's/^#\?Port .*/Port 2222/' /etc/ssh/sshd_config
# If the distro ships a socket unit, it also needs the new port:
sudo systemctl disable --now ssh.socket 2>/dev/null || true
sudo systemctl restart ssh || sudo systemctl restart sshd

# Open the firewall for the new admin port and for public :22 (the TUI).
sudo ufw allow 2222/tcp comment 'admin ssh'
sudo ufw allow 22/tcp   comment 'valo-tui public ssh'
sudo ufw --force enable
```

In a **second terminal**, verify you can still get in on the new port — do not
close the first session until this succeeds:

```bash
ssh -p 2222 you@valo.blackpantha.com
```

### 2. Build, user, host key, services

```bash
# Build static binaries (pure-Go SQLite, no CGO — runs anywhere).
CGO_ENABLED=0 go build -o /usr/local/bin/ ./cmd/valo-fetcher ./cmd/valo-tui-ssh

# Dedicated user + state dir for the shared cache + the public host key.
sudo useradd --system --home /var/lib/valo-tui --create-home valo
sudo -u valo ssh-keygen -t ed25519 -f /var/lib/valo-tui/.ssh/id_ed25519 -N ""

# Install + start both services (the SSH unit binds :22 via CAP_NET_BIND_SERVICE).
sudo cp deploy/valo-fetcher.service deploy/valo-tui-ssh.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now valo-fetcher valo-tui-ssh

# Confirm: cache filling, SSH server up on :22.
sudo journalctl -u valo-fetcher -f
sudo journalctl -u valo-tui-ssh -n 20
```

### 3. Connect

From anywhere:

```bash
ssh valo.blackpantha.com
```

(The username is ignored — there's no login, it drops straight into the TUI.
First connection shows this host's key fingerprint, which is the public TUI
host key from step 2, separate from your admin sshd's key.)

> Prefer not to touch the admin sshd? Set `Environment=VALO_TUI_SSH_PORT=23234`
> in `valo-tui-ssh.service`, remove its `AmbientCapabilities` line, and users
> connect with `ssh -p 23234 valo.blackpantha.com` instead.

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
