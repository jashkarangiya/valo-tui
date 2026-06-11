#!/usr/bin/env bash
#
# In-container provisioner for valo-tui. Runs as root *inside* a fresh Debian 12
# (or any Debian/Ubuntu) LXC — normally created and invoked by ../proxmox/
# install.sh, but safe to run by hand inside any such container.
#
# It builds the binaries, creates a locked-down nologin service user, installs
# the hardened systemd units, and removes any real sshd so the ONLY thing
# listening on SSH is the read-only TUI (Wish). Re-running it pulls the latest
# code, rebuilds, and restarts the services — it doubles as the update path and
# never touches /var/lib/valo-tui (the cache + host key survive).
#
# Override via env: REPO_URL, BRANCH, GO_VERSION, SSH_PORT, INTERVAL.
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/jashkarangiya/valo-tui.git}"
BRANCH="${BRANCH:-main}"
GO_VERSION="${GO_VERSION:-1.26.3}"
SSH_PORT="${SSH_PORT:-22}"
INTERVAL="${INTERVAL:-30s}"
SRC=/opt/valo-tui-src
STATE_DIR=/var/lib/valo-tui

say() { echo -e "\e[1;32m[valo-tui]\e[0m $*"; }
die() { echo -e "\e[1;31m[valo-tui] $*\e[0m" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "must run as root inside the container"

say "Installing build dependencies"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq --no-install-recommends git ca-certificates curl

# ── No real SSH server in this container ─────────────────────────────────────
# Manage it from the Proxmox host with `pct enter`. Removing openssh-server means
# there is no shell-bearing SSH daemon to break into — only Wish, which serves
# the TUI and nothing else. (Wish generates its own host key, so we don't need
# ssh-keygen / openssh-client either.)
say "Removing any real SSH daemon (TUI is the only SSH surface)"
systemctl disable --now ssh sshd 2>/dev/null || true
apt-get purge -y -qq openssh-server 2>/dev/null || true
apt-get autoremove -y -qq 2>/dev/null || true

# ── Go toolchain (upstream; Debian's is far older than go.mod requires) ──────
# Idempotent: skip the download when the wanted version is already in place, so
# re-runs (updates) are quick.
if /usr/local/go/bin/go version 2>/dev/null | grep -q "go${GO_VERSION} "; then
	say "Go ${GO_VERSION} already installed"
else
	say "Installing Go ${GO_VERSION}"
	ARCH=$(dpkg --print-architecture) # amd64 | arm64
	curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" -o /tmp/go.tgz \
		|| die "could not download Go ${GO_VERSION} for ${ARCH}"
	rm -rf /usr/local/go
	tar -C /usr/local -xzf /tmp/go.tgz
	rm -f /tmp/go.tgz
fi
export PATH="$PATH:/usr/local/go/bin"

say "Building from ${REPO_URL} (${BRANCH})"
rm -rf "$SRC"
git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$SRC" \
	|| die "could not clone ${REPO_URL} @ ${BRANCH}"
cd "$SRC"
# Pure-Go build (modernc.org/sqlite) — CGO off, so no C toolchain needed and the
# binaries are static. Only the two server binaries; the TUI/seed aren't served.
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' \
	-o /usr/local/bin/ ./cmd/valo-fetcher ./cmd/valo-tui-ssh \
	|| die "go build failed"

# ── Locked-down service user ────────────────────────────────────────────────
# System account, no login shell, no password, no home login. The Wish process
# runs as this user; even a hypothetical escape lands on an account that can do
# nothing.
say "Creating locked-down 'valo' service user"
id valo &>/dev/null || useradd --system --home-dir "$STATE_DIR" \
	--shell /usr/sbin/nologin valo
install -d -o valo -g valo -m 0750 "$STATE_DIR"
install -d -o valo -g valo -m 0700 "$STATE_DIR/.ssh"

say "Installing systemd services"
install -m 0644 deploy/valo-fetcher.service /etc/systemd/system/valo-fetcher.service
# Render the SSH unit's port / fetcher interval from the chosen values.
sed -e "s|VALO_TUI_SSH_PORT=22|VALO_TUI_SSH_PORT=${SSH_PORT}|" \
	deploy/valo-tui-ssh.service >/etc/systemd/system/valo-tui-ssh.service
if [ "$SSH_PORT" -ge 1024 ]; then
	# Unprivileged port: needs no capability at all — drop ambient caps and
	# empty the bounding set so the process can hold none.
	sed -i -e '/^AmbientCapabilities=/d' \
		-e 's/^CapabilityBoundingSet=.*/CapabilityBoundingSet=/' \
		/etc/systemd/system/valo-tui-ssh.service
fi
sed -i "s|--interval 30s|--interval ${INTERVAL}|" /etc/systemd/system/valo-fetcher.service

systemctl daemon-reload
systemctl enable valo-fetcher valo-tui-ssh >/dev/null 2>&1 || true
# `restart` (not just `enable --now`) so an update picks up the freshly built
# binary even when the service was already running. The fetcher does a full
# initial scrape on startup, so this also primes the cache — no separate
# `--once` run (which would double the load on vlr.gg every deploy/update).
systemctl restart valo-fetcher valo-tui-ssh

# ── Verify the services actually came up ─────────────────────────────────────
ok=1
for svc in valo-fetcher valo-tui-ssh; do
	if systemctl is-active --quiet "$svc"; then
		say "$svc is active"
	else
		ok=0
		echo -e "\e[1;31m[valo-tui] $svc failed to start — recent logs:\e[0m" >&2
		journalctl -u "$svc" -n 25 --no-pager >&2 || true
	fi
done
[ "$ok" -eq 1 ] || die "one or more services failed; see logs above"

IP=$(hostname -I 2>/dev/null | awk '{print $1}')
PORT_ARG=""
[ "$SSH_PORT" = 22 ] || PORT_ARG=" -p $SSH_PORT"
say "Done. Connect with:  ssh ${IP:-<container-ip>}${PORT_ARG}"
say "Manage from the Proxmox host with:  pct enter <ctid>"
