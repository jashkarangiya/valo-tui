#!/usr/bin/env bash
#
# In-container provisioner for valo-tui. Runs as root inside a fresh Debian 12
# LXC (created by valo-tui.sh, or any Debian/Ubuntu container). Builds the
# binaries, creates a locked-down service user, installs the hardened systemd
# units — and removes any real sshd so the ONLY thing listening on SSH is the
# read-only TUI.
#
# Override via env: REPO_URL, BRANCH, GO_VERSION, SSH_PORT, INTERVAL.
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/jashkarangiya/valo-tui.git}"
BRANCH="${BRANCH:-main}"
GO_VERSION="${GO_VERSION:-1.26.3}"
SSH_PORT="${SSH_PORT:-22}"
INTERVAL="${INTERVAL:-30s}"
SRC=/opt/valo-tui-src

say() { echo -e "\e[1;32m[valo-tui]\e[0m $*"; }

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

say "Installing Go ${GO_VERSION}"
ARCH=$(dpkg --print-architecture) # amd64 | arm64
curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" -o /tmp/go.tgz
rm -rf /usr/local/go
tar -C /usr/local -xzf /tmp/go.tgz
rm -f /tmp/go.tgz
export PATH="$PATH:/usr/local/go/bin"

say "Building from ${REPO_URL} (${BRANCH})"
rm -rf "$SRC"
git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$SRC"
cd "$SRC"
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' \
	-o /usr/local/bin/ ./cmd/valo-fetcher ./cmd/valo-tui-ssh

# ── Locked-down service user ────────────────────────────────────────────────
# System account, no login shell, no password, no home login. The Wish process
# runs as this user; even a hypothetical escape lands on an account that can do
# nothing.
say "Creating locked-down 'valo' service user"
id valo &>/dev/null || useradd --system --home-dir /var/lib/valo-tui \
	--shell /usr/sbin/nologin valo
install -d -o valo -g valo -m 0750 /var/lib/valo-tui
install -d -o valo -g valo -m 0700 /var/lib/valo-tui/.ssh

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
systemctl enable --now valo-fetcher valo-tui-ssh

say "Priming the cache (one-shot fetch so the TUI isn't empty on first connect)"
sudo -u valo VALO_TUI_DB=/var/lib/valo-tui/cache.db /usr/local/bin/valo-fetcher --once || true

IP=$(hostname -I 2>/dev/null | awk '{print $1}')
PORT_ARG=""
[ "$SSH_PORT" = 22 ] || PORT_ARG=" -p $SSH_PORT"
say "Done. Connect with:  ssh ${IP:-<container-ip>}${PORT_ARG}"
say "Manage from the Proxmox host with:  pct enter <ctid>"
