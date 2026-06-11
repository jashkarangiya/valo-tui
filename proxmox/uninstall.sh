#!/usr/bin/env bash
#
# valo-tui — stop and optionally destroy its Proxmox LXC.
#
# Run ON THE PROXMOX HOST. By default it stops the container and then asks
# whether to destroy it permanently. Destroying removes the rootfs — the cache
# and SSH host key go with it.
#
#   bash proxmox/uninstall.sh                 # auto-detect, stop, ask to destroy
#   CTID=141 bash proxmox/uninstall.sh        # target a specific CT
#   CTID=141 PURGE=1 bash proxmox/uninstall.sh  # non-interactive destroy
#
set -euo pipefail

CTID="${CTID:-}"
PURGE="${PURGE:-0}"                   # 1 = destroy without asking
NONINTERACTIVE="${NONINTERACTIVE:-0}"

if [ -t 1 ]; then
	RED=$'\e[1;31m'; GRN=$'\e[1;32m'; YLW=$'\e[1;33m'; BLU=$'\e[1;34m'; RST=$'\e[0m'
else
	RED=''; GRN=''; YLW=''; BLU=''; RST=''
fi
msg_info()  { echo -e "${BLU}ℹ${RST} $*"; }
msg_ok()    { echo -e "${GRN}✓${RST} $*"; }
msg_warn()  { echo -e "${YLW}⚠${RST} $*"; }
die()       { echo -e "${RED}✗${RST} $*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "must run as root on the Proxmox host"
command -v pct >/dev/null || die "this must run on a Proxmox VE host ('pct' not found)"

is_valo_ct() { pct exec "$1" -- test -x /usr/local/bin/valo-tui-ssh >/dev/null 2>&1; }

# Auto-detect the single CT (running or stopped) that carries valo-tui.
if [ -z "$CTID" ]; then
	msg_info "No CTID given — scanning containers for valo-tui…"
	found=()
	while read -r id _; do
		[[ "$id" =~ ^[0-9]+$ ]] || continue
		# Only a running CT can be probed; start nothing here — match by config too.
		if is_valo_ct "$id" || pct config "$id" 2>/dev/null | grep -qi 'hostname: valo'; then
			found+=("$id")
		fi
	done < <(pct list 2>/dev/null | awk 'NR>1{print $1}')
	case "${#found[@]}" in
		1) CTID="${found[0]}"; msg_info "Found valo-tui in CT ${CTID}." ;;
		0) die "no valo-tui container found — pass CTID=<id>." ;;
		*) die "multiple candidate containers (${found[*]}) — pass CTID=<id>." ;;
	esac
fi

[[ "$CTID" =~ ^[0-9]+$ ]] || die "CTID must be numeric (got '${CTID}')"
pct status "$CTID" >/dev/null 2>&1 || die "CT ${CTID} does not exist"

HOST="$(pct config "$CTID" 2>/dev/null | sed -n 's/^hostname: //p')"
msg_info "Target: CT ${CTID}${HOST:+ (${HOST})}"

# Stop it if running.
if pct status "$CTID" | grep -q running; then
	msg_info "Stopping CT ${CTID}…"
	pct stop "$CTID" >/dev/null || die "could not stop CT ${CTID}"
	msg_ok "Stopped."
else
	msg_info "CT ${CTID} is already stopped."
fi

# Decide whether to destroy.
do_destroy=0
if [ "$PURGE" = 1 ]; then
	do_destroy=1
elif [ "$NONINTERACTIVE" = 1 ] || [ ! -e /dev/tty ]; then
	msg_warn "Container stopped but NOT destroyed (set PURGE=1 to destroy non-interactively)."
else
	printf '%sDestroy CT %s permanently?%s This removes the rootfs, cache and host key. [y/N]: ' \
		"$RED" "$CTID" "$RST" >/dev/tty
	IFS= read -r ans </dev/tty || ans=""
	[[ "$ans" =~ ^[Yy] ]] && do_destroy=1
fi

if [ "$do_destroy" = 1 ]; then
	msg_info "Destroying CT ${CTID}…"
	pct destroy "$CTID" >/dev/null || die "could not destroy CT ${CTID}"
	msg_ok "CT ${CTID} destroyed."
else
	msg_ok "CT ${CTID} left in place (stopped). Start again with: pct start ${CTID}"
fi
