#!/usr/bin/env bash
#
# valo-tui — update an existing Proxmox LXC in place.
#
# Run ON THE PROXMOX HOST. Re-pulls the chosen branch inside the container,
# rebuilds the binaries, reinstalls the (possibly updated) hardened units, and
# restarts the services. The cache + SSH host key in /var/lib/valo-tui are left
# untouched, and the listen port is preserved from the running service unless
# you override SSH_PORT.
#
#   bash proxmox/update.sh                 # auto-detect the CT, update to main
#   CTID=141 BRANCH=main bash proxmox/update.sh
#
set -euo pipefail

CTID="${CTID:-}"
BRANCH="${BRANCH:-main}"
REPO_URL="${REPO_URL:-https://github.com/jashkarangiya/valo-tui.git}"
GO_VERSION="${GO_VERSION:-1.26.3}"
SSH_PORT="${SSH_PORT:-}"        # blank = keep whatever the unit already uses
INTERVAL="${INTERVAL:-}"        # blank = keep current fetcher interval
NONINTERACTIVE="${NONINTERACTIVE:-0}"
RAW_BASE="${RAW_BASE:-https://raw.githubusercontent.com/jashkarangiya/valo-tui}"

if [ -t 1 ]; then
	B=$'\e[1m'; DIM=$'\e[2m'; RED=$'\e[1;31m'; GRN=$'\e[1;32m'; BLU=$'\e[1;34m'; RST=$'\e[0m'
else
	B=''; DIM=''; RED=''; GRN=''; BLU=''; RST=''
fi
msg_info()  { echo -e "${BLU}ℹ${RST} $*"; }
msg_ok()    { echo -e "${GRN}✓${RST} $*"; }
die()       { echo -e "${RED}✗${RST} $*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "must run as root on the Proxmox host"
command -v pct >/dev/null || die "this must run on a Proxmox VE host ('pct' not found)"

# is_valo_ct CTID -> 0 if that container has our binary installed.
is_valo_ct() { pct exec "$1" -- test -x /usr/local/bin/valo-tui-ssh >/dev/null 2>&1; }

# Auto-detect: the single running CT that has valo-tui installed.
if [ -z "$CTID" ]; then
	msg_info "No CTID given — scanning running containers for valo-tui…"
	found=()
	while read -r id _; do
		[[ "$id" =~ ^[0-9]+$ ]] || continue
		is_valo_ct "$id" && found+=("$id")
	done < <(pct list 2>/dev/null | awk 'NR>1 && $2=="running"{print $1}')
	case "${#found[@]}" in
		1) CTID="${found[0]}"; msg_info "Found valo-tui in CT ${CTID}." ;;
		0) die "no running valo-tui container found — pass CTID=<id>." ;;
		*) die "multiple valo-tui containers (${found[*]}) — pass CTID=<id>." ;;
	esac
fi

[[ "$CTID" =~ ^[0-9]+$ ]] || die "CTID must be numeric (got '${CTID}')"
pct status "$CTID" >/dev/null 2>&1 || die "CT ${CTID} does not exist"
is_valo_ct "$CTID" || die "CT ${CTID} doesn't look like a valo-tui container (no /usr/local/bin/valo-tui-ssh)"

# Make sure it's running so we can exec into it.
if ! pct status "$CTID" | grep -q running; then
	msg_info "Starting CT ${CTID}…"
	pct start "$CTID" >/dev/null
	sleep 3
fi

# Preserve the current listen port unless the caller overrides it.
if [ -z "$SSH_PORT" ]; then
	SSH_PORT="$(pct exec "$CTID" -- bash -c \
		'systemctl show valo-tui-ssh -p Environment --value 2>/dev/null' \
		| tr ' ' '\n' | sed -n 's/^VALO_TUI_SSH_PORT=//p' | head -1)"
	SSH_PORT="${SSH_PORT:-22}"
fi
msg_info "Updating CT ${CTID} → branch '${BRANCH}', port ${SSH_PORT} (cache + host key preserved)."

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
curl -fsSL "${RAW_BASE}/${BRANCH}/proxmox/ct-setup.sh" -o "$TMP" \
	|| die "could not fetch proxmox/ct-setup.sh from branch '${BRANCH}'"
pct push "$CTID" "$TMP" /root/ct-setup.sh

# Only pass INTERVAL when overriding; ct-setup defaults it otherwise.
EXTRA_ENV=()
[ -n "$INTERVAL" ] && EXTRA_ENV+=("INTERVAL=$INTERVAL")
pct exec "$CTID" -- env \
	REPO_URL="$REPO_URL" BRANCH="$BRANCH" GO_VERSION="$GO_VERSION" SSH_PORT="$SSH_PORT" \
	"${EXTRA_ENV[@]}" \
	bash /root/ct-setup.sh

IP_ADDR="$(pct exec "$CTID" -- hostname -I 2>/dev/null | awk '{print $1}')"
PORT_ARG=""; [ "$SSH_PORT" = 22 ] || PORT_ARG=" -p ${SSH_PORT}"
echo
msg_ok "${B}CT ${CTID} updated.${RST}  Connect:  ssh ${IP_ADDR:-<container-ip>}${PORT_ARG}"
echo -e "  ${DIM}journalctl -u valo-tui-ssh -f   (inside: pct enter ${CTID})${RST}"
