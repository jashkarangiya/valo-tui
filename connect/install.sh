#!/bin/sh
#
# valo-tui — one-line connector for macOS / Linux / WSL.
#
#   curl -fsSL https://valo.black-pantha.com/install.sh | sh
#
# It installs Cloudflare's `cloudflared` (needed to reach the tunnel) and writes
# an `ssh valo` shortcut into ~/.ssh/config. After that, watching Valorant
# esports is just:  ssh valo
#
# It installs NOTHING that can touch your machine beyond cloudflared + one ssh
# config block, and it never needs root. The server end is a read-only TUI — no
# shell, no file access — so this connection can only ever draw a scoreboard.
#
# Override the host by exporting VALO_HOST before running.
set -eu

# The public hostname your tunnel is routed to. EDIT THIS to your domain (or
# bake it in when you host the script); users override with VALO_HOST=...
VALO_HOST="${VALO_HOST:-valo.black-pantha.com}"
ALIAS="${VALO_ALIAS:-valo}"

red()  { printf '\033[1;31m%s\033[0m\n' "$*" >&2; }
grn()  { printf '\033[1;32m%s\033[0m\n' "$*"; }
info() { printf '\033[1;34m%s\033[0m\n' "$*"; }
die()  { red "✗ $*"; exit 1; }

# ── Detect platform + arch ───────────────────────────────────────────────────
OS="$(uname -s)"
ARCH="$(uname -m)"
case "$ARCH" in
	x86_64|amd64) ARCH=amd64 ;;
	aarch64|arm64) ARCH=arm64 ;;
	armv7l|armv6l) ARCH=arm ;;
	*) die "unsupported CPU arch: $ARCH" ;;
esac

BIN_DIR="${HOME}/.local/bin"
mkdir -p "$BIN_DIR"

# ── Install cloudflared if missing ───────────────────────────────────────────
find_cloudflared() {
	if command -v cloudflared >/dev/null 2>&1; then command -v cloudflared; return; fi
	[ -x "${BIN_DIR}/cloudflared" ] && { printf '%s\n' "${BIN_DIR}/cloudflared"; return; }
	return 1
}

CF="$(find_cloudflared || true)"
if [ -n "$CF" ]; then
	grn "✓ cloudflared already installed ($CF)"
else
	info "Installing cloudflared…"
	case "$OS" in
		Darwin)
			if command -v brew >/dev/null 2>&1; then
				brew install cloudflared >/dev/null || die "brew install cloudflared failed"
				CF="$(command -v cloudflared)"
			else
				URL="https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-${ARCH}.tgz"
				curl -fsSL "$URL" -o /tmp/cf.tgz || die "download failed: $URL"
				tar -xzf /tmp/cf.tgz -C "$BIN_DIR" cloudflared || die "could not extract cloudflared"
				rm -f /tmp/cf.tgz
				chmod +x "${BIN_DIR}/cloudflared"
				CF="${BIN_DIR}/cloudflared"
			fi
			;;
		Linux)
			URL="https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-${ARCH}"
			curl -fsSL "$URL" -o "${BIN_DIR}/cloudflared" || die "download failed: $URL"
			chmod +x "${BIN_DIR}/cloudflared"
			CF="${BIN_DIR}/cloudflared"
			;;
		*) die "unsupported OS: $OS (this script covers macOS, Linux, and WSL)" ;;
	esac
	grn "✓ cloudflared installed at $CF"
fi

command -v ssh >/dev/null 2>&1 || die "the 'ssh' client is not installed — install OpenSSH and re-run"

# ── Write a managed block into ~/.ssh/config ─────────────────────────────────
# Absolute path to cloudflared in ProxyCommand: ssh runs it via /bin/sh, whose
# PATH may not include ~/.local/bin, so we don't rely on PATH at all.
SSH_DIR="${HOME}/.ssh"
CFG="${SSH_DIR}/config"
mkdir -p "$SSH_DIR"; chmod 700 "$SSH_DIR"
[ -f "$CFG" ] || { : >"$CFG"; chmod 600 "$CFG"; }

BEGIN="# >>> valo-tui (managed) >>>"
END="# <<< valo-tui (managed) <<<"

# Strip any previous managed block, then append a fresh one (idempotent).
TMP="$(mktemp)"
awk -v b="$BEGIN" -v e="$END" '
	$0==b{skip=1} !skip{print} $0==e{skip=0}
' "$CFG" >"$TMP" 2>/dev/null || cp "$CFG" "$TMP"
# Drop a trailing blank line for tidiness, then add our block.
{
	cat "$TMP"
	printf '%s\n' "$BEGIN"
	printf 'Host %s\n' "$ALIAS"
	printf '    HostName %s\n' "$VALO_HOST"
	printf '    User viewer\n'
	printf '    ProxyCommand %s access ssh --hostname %%h\n' "$CF"
	printf '    RequestTTY yes\n'
	printf '    StrictHostKeyChecking accept-new\n'
	printf '    UserKnownHostsFile %s/known_hosts_valo\n' "$SSH_DIR"
	printf '%s\n' "$END"
} >"$CFG"
rm -f "$TMP"
chmod 600 "$CFG"
grn "✓ Added 'Host ${ALIAS}' to ${CFG}"

echo
# One command only: connect right now. Re-running this same command later just
# reconnects (cloudflared + the ssh config are already in place). Set
# VALO_NO_CONNECT=1 to only set things up without launching.
if [ "${VALO_NO_CONNECT:-0}" = 1 ]; then
	grn "✓ Set up. Connect any time with:  ssh ${ALIAS}"
	exit 0
fi
grn "✓ Set up — connecting now.  (Next time, run the same command, or just: ssh ${ALIAS})"
echo
# curl|sh leaves our stdin attached to the pipe, so pull a real terminal from
# /dev/tty and force a PTY (-tt) for the TUI. accept-new (set above) means the
# first-time host-key prompt won't block.
if [ -e /dev/tty ]; then
	exec ssh -tt "$ALIAS" </dev/tty
else
	info "No terminal detected here — connect with:  ssh ${ALIAS}"
fi
