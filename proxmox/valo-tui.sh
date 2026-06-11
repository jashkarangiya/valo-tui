#!/usr/bin/env bash
#
# Proxmox VE helper: create an UNPRIVILEGED LXC and deploy valo-tui into it.
# Run on the Proxmox host:
#
#   bash -c "$(curl -fsSL https://raw.githubusercontent.com/jashkarangiya/valo-tui/main/proxmox/valo-tui.sh)"
#
# or download and read it first (recommended), then run. Everything is
# parameterised by env vars with sane defaults:
#
#   CTID=141 HOSTNAME=valo IP=192.168.1.50/24,gw=192.168.1.1 SSH_PORT=22 ./valo-tui.sh
#
set -euo pipefail

CTID="${CTID:-$(pvesh get /cluster/nextid)}"
HOSTNAME="${HOSTNAME:-valo-tui}"
STORAGE="${STORAGE:-local-lvm}"          # where the rootfs lives
TEMPLATE_STORAGE="${TEMPLATE_STORAGE:-local}" # where CT templates live
BRIDGE="${BRIDGE:-vmbr0}"
IP="${IP:-dhcp}"                         # or "10.0.0.5/24,gw=10.0.0.1"
DISK_GB="${DISK_GB:-4}"
RAM_MB="${RAM_MB:-512}"
CORES="${CORES:-1}"
SSH_PORT="${SSH_PORT:-22}"
REPO_URL="${REPO_URL:-https://github.com/jashkarangiya/valo-tui.git}"
BRANCH="${BRANCH:-main}"

say() { echo -e "\e[1;34m[valo-tui]\e[0m $*"; }
die() { echo -e "\e[1;31m[valo-tui] $*\e[0m" >&2; exit 1; }
command -v pct >/dev/null || die "must run on a Proxmox VE host (pct not found)"

say "Resolving the latest Debian 12 template"
pveam update >/dev/null 2>&1 || true
TEMPLATE=$(pveam available --section system | awk '/debian-12-standard/{print $2}' | sort -V | tail -1)
[ -n "$TEMPLATE" ] || die "no debian-12-standard template available via pveam"
if ! pveam list "$TEMPLATE_STORAGE" 2>/dev/null | grep -q "$TEMPLATE"; then
	say "Downloading $TEMPLATE to $TEMPLATE_STORAGE"
	pveam download "$TEMPLATE_STORAGE" "$TEMPLATE"
fi

say "Creating unprivileged LXC $CTID ($HOSTNAME)"
pct create "$CTID" "${TEMPLATE_STORAGE}:vztmpl/${TEMPLATE}" \
	--hostname "$HOSTNAME" \
	--cores "$CORES" --memory "$RAM_MB" \
	--rootfs "${STORAGE}:${DISK_GB}" \
	--net0 "name=eth0,bridge=${BRIDGE},ip=${IP}" \
	--unprivileged 1 \
	--features nesting=0,keyctl=0 \
	--onboot 1

# Drop the LXC's own capability to ever gain new privileges; the workload needs
# none beyond binding a port (handled per-service in systemd).
pct start "$CTID"

say "Waiting for container networking"
for _ in $(seq 1 30); do
	pct exec "$CTID" -- getent hosts deb.debian.org >/dev/null 2>&1 && break
	sleep 2
done

say "Provisioning inside the container"
TMP=$(mktemp)
curl -fsSL "https://raw.githubusercontent.com/jashkarangiya/valo-tui/${BRANCH}/proxmox/install.sh" -o "$TMP" \
	|| die "could not fetch install.sh from $BRANCH"
pct push "$CTID" "$TMP" /root/install.sh
rm -f "$TMP"
pct exec "$CTID" -- env \
	REPO_URL="$REPO_URL" BRANCH="$BRANCH" SSH_PORT="$SSH_PORT" \
	bash /root/install.sh

IP_ADDR=$(pct exec "$CTID" -- hostname -I 2>/dev/null | awk '{print $1}')
say "Deployed. Container $CTID is up."
say "Connect:  ssh ${IP_ADDR:-<container-ip>}$([ "$SSH_PORT" = 22 ] || echo " -p $SSH_PORT")"
say "Manage:   pct enter $CTID     (no SSH login exists inside — by design)"
