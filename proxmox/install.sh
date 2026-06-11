#!/usr/bin/env bash
#
# valo-tui — Proxmox VE one-command installer.
#
# Run this ON THE PROXMOX HOST. It creates an UNPRIVILEGED Debian 12 LXC and
# deploys valo-tui into it: the vlr.gg fetcher + the Wish SSH server, and
# nothing else. There is NO real sshd in the container — Wish is the only thing
# listening on SSH and it serves only the read-only TUI (no shell/exec/SFTP/
# forwarding). See ./README.md for the full security model.
#
#   bash -c "$(curl -fsSL https://raw.githubusercontent.com/jashkarangiya/valo-tui/main/proxmox/install.sh)"
#
# Prefer to read before you run (recommended): download this file, skim it, run
# it. Everything is interactive with sane defaults; every value can also be set
# non-interactively via the env vars shown in parentheses below.
#
set -euo pipefail

# ── Defaults (override any of these via environment) ─────────────────────────
CTID="${CTID:-}"                                  # blank = next free id
# NB: use CT_HOSTNAME, not HOSTNAME — bash auto-sets $HOSTNAME to the PVE host's
# own name, which would otherwise leak in as the default container name.
CT_HOSTNAME="${CT_HOSTNAME:-valo-tui}"
STORAGE="${STORAGE:-local-lvm}"                   # rootfs storage
TEMPLATE_STORAGE="${TEMPLATE_STORAGE:-local}"     # where CT templates live
BRIDGE="${BRIDGE:-vmbr0}"
IP="${IP:-dhcp}"                                  # dhcp | CIDR e.g. 192.168.1.50/24,gw=192.168.1.1
MAC="${MAC:-}"                                     # optional fixed MAC (for DHCP reservations)
DISK_GB="${DISK_GB:-4}"
RAM_MB="${RAM_MB:-512}"
CORES="${CORES:-1}"
SSH_PORT="${SSH_PORT:-22}"                         # 22 = bare `ssh host`; or e.g. 23234
REPO_URL="${REPO_URL:-https://github.com/jashkarangiya/valo-tui.git}"
BRANCH="${BRANCH:-main}"
GO_VERSION="${GO_VERSION:-1.26.3}"
INTERVAL="${INTERVAL:-30s}"                        # fetcher live-refresh interval
NONINTERACTIVE="${NONINTERACTIVE:-0}"             # 1 = never prompt, use defaults/env

# Where to fetch the in-container provisioner from (matches the chosen branch).
RAW_BASE="${RAW_BASE:-https://raw.githubusercontent.com/jashkarangiya/valo-tui}"

# ── Pretty output ────────────────────────────────────────────────────────────
if [ -t 1 ]; then
	B=$'\e[1m'; DIM=$'\e[2m'; RED=$'\e[1;31m'; GRN=$'\e[1;32m'
	YLW=$'\e[1;33m'; BLU=$'\e[1;34m'; RST=$'\e[0m'
else
	B=''; DIM=''; RED=''; GRN=''; YLW=''; BLU=''; RST=''
fi
msg_info()  { echo -e "${BLU}ℹ${RST} $*"; }
msg_ok()    { echo -e "${GRN}✓${RST} $*"; }
msg_warn()  { echo -e "${YLW}⚠${RST} $*"; }
msg_error() { echo -e "${RED}✗${RST} $*" >&2; }

CREATED_CTID=""
die() { msg_error "$*"; cleanup_hint; exit 1; }

cleanup_hint() {
	[ -n "$CREATED_CTID" ] || return 0
	echo >&2
	msg_warn "Container ${CREATED_CTID} was created but setup did not finish."
	echo -e "  ${DIM}inspect:${RST} pct enter ${CREATED_CTID}" >&2
	echo -e "  ${DIM}remove :${RST} bash proxmox/uninstall.sh   ${DIM}(CTID=${CREATED_CTID})${RST}" >&2
}

on_err() {
	local rc=$?
	msg_error "install failed at line ${BASH_LINENO[0]} (exit ${rc})"
	cleanup_hint
	exit "$rc"
}
trap on_err ERR

banner() {
	echo -e "${B}"
	cat <<'ART'
            _         _        _
 __ ____ _| |___ ___| |_ _  _(_)
 \ V / _` | / _ \___|  _| || | |
  \_/\__,_|_\___/   \__|\_,_|_|
ART
	echo -e "${RST}${DIM}  Valorant esports in your terminal — Proxmox LXC deploy${RST}\n"
}

# ── Prompt helper: whiptail when available + interactive, else read /dev/tty ──
USE_WHIPTAIL=0
ask() { # ask VAR "Label"
	local _var="$1" _label="$2" _cur="${!1}" _ans
	[ "$NONINTERACTIVE" = 1 ] && return 0
	if [ "$USE_WHIPTAIL" = 1 ]; then
		_ans=$(whiptail --title "valo-tui installer" --inputbox "$_label" 8 72 "$_cur" 3>&1 1>&2 2>&3) \
			|| die "cancelled"
	else
		printf '%s%s%s [%s%s%s]: ' "$B" "$_label" "$RST" "$GRN" "$_cur" "$RST" >/dev/tty
		IFS= read -r _ans </dev/tty || _ans=""
	fi
	[ -n "$_ans" ] && printf -v "$_var" '%s' "$_ans"
	return 0
}

confirm() { # confirm "Question"  -> 0 yes / 1 no
	[ "$NONINTERACTIVE" = 1 ] && return 0
	if [ "$USE_WHIPTAIL" = 1 ]; then
		whiptail --title "valo-tui installer" --yesno "$1" 20 72
	else
		local _a
		printf '%s [y/N]: ' "$1" >/dev/tty
		IFS= read -r _a </dev/tty || _a=""
		[[ "$_a" =~ ^[Yy] ]]
	fi
}

# ── Pre-flight checks ────────────────────────────────────────────────────────
banner
[ "$(id -u)" -eq 0 ] || die "must run as root on the Proxmox host"
command -v pct  >/dev/null || die "this must run on a Proxmox VE host ('pct' not found)"
command -v pvesh >/dev/null || die "this must run on a Proxmox VE host ('pvesh' not found)"

if [ "$NONINTERACTIVE" != 1 ] && [ -e /dev/tty ]; then
	command -v whiptail >/dev/null && USE_WHIPTAIL=1
else
	NONINTERACTIVE=1   # no tty -> can't prompt safely
fi

# Default CTID to the next free id (only if the user didn't pin one).
[ -n "$CTID" ] || CTID="$(pvesh get /cluster/nextid 2>/dev/null || echo 100)"

# ── Gather settings ──────────────────────────────────────────────────────────
if [ "$NONINTERACTIVE" = 1 ]; then
	msg_info "Non-interactive mode — using defaults/env values."
else
	summary="CT ID:        ${CTID}
Hostname:     ${CT_HOSTNAME}
Storage:      ${STORAGE}
Bridge:       ${BRIDGE}
Network:      ${IP}${MAC:+ (mac ${MAC})}
Cores:        ${CORES}
RAM (MB):     ${RAM_MB}
Disk (GB):    ${DISK_GB}
SSH/TUI port: ${SSH_PORT}
Repo:         ${REPO_URL}
Branch:       ${BRANCH}"
	if confirm "Use these default settings?

${summary}"; then
		msg_info "Using defaults."
	else
		ask CTID         "CT ID (numeric, unused)"
		ask CT_HOSTNAME  "Hostname"
		ask STORAGE      "Storage for the rootfs (pvesm status)"
		ask BRIDGE       "Network bridge"
		ask IP           "IP: 'dhcp' or CIDR e.g. 192.168.1.50/24,gw=192.168.1.1"
		ask MAC          "Fixed MAC (blank = auto)"
		ask CORES        "CPU cores"
		ask RAM_MB       "RAM in MB"
		ask DISK_GB      "Disk in GB"
		ask SSH_PORT     "SSH/TUI listen port (22 = bare 'ssh host')"
		ask REPO_URL     "Git repo URL"
		ask BRANCH       "Git branch"
	fi
fi

# ── Validate ─────────────────────────────────────────────────────────────────
is_uint() { [[ "$1" =~ ^[0-9]+$ ]]; }
is_uint "$CTID"    || die "CT ID must be a number (got '${CTID}')"
is_uint "$CORES"   && [ "$CORES" -ge 1 ]   || die "cores must be a positive integer"
is_uint "$RAM_MB"  && [ "$RAM_MB" -ge 128 ] || die "RAM must be >= 128 MB"
is_uint "$DISK_GB" && [ "$DISK_GB" -ge 1 ]  || die "disk must be >= 1 GB"
is_uint "$SSH_PORT" && [ "$SSH_PORT" -ge 1 ] && [ "$SSH_PORT" -le 65535 ] \
	|| die "SSH port must be 1-65535"
[[ "$CT_HOSTNAME" =~ ^[a-zA-Z0-9][a-zA-Z0-9.-]*$ ]] || die "invalid hostname '${CT_HOSTNAME}'"

if pct status "$CTID" >/dev/null 2>&1; then
	die "CT ${CTID} already exists. Pick another CTID, or remove it with proxmox/uninstall.sh."
fi
pvesm status -storage "$STORAGE" >/dev/null 2>&1 \
	|| die "storage '${STORAGE}' not found. Available: $(pvesm status 2>/dev/null | awk 'NR>1{print $1}' | paste -sd' ' -)"
[ -d "/sys/class/net/${BRIDGE}" ] \
	|| die "bridge '${BRIDGE}' not found. Available: $(for n in /sys/class/net/*; do printf '%s ' "${n##*/}"; done)"

# ── Resolve + download the Debian 12 template ────────────────────────────────
msg_info "Resolving the latest Debian 12 LXC template…"
pveam update >/dev/null 2>&1 || true
TEMPLATE=$(pveam available --section system 2>/dev/null \
	| awk '/debian-12-standard/{print $2}' | sort -V | tail -1)
[ -n "$TEMPLATE" ] || die "no debian-12-standard template available via pveam"
if ! pveam list "$TEMPLATE_STORAGE" 2>/dev/null | grep -q "$TEMPLATE"; then
	msg_info "Downloading ${TEMPLATE} to ${TEMPLATE_STORAGE}…"
	pveam download "$TEMPLATE_STORAGE" "$TEMPLATE" >/dev/null \
		|| die "template download failed (is '${TEMPLATE_STORAGE}' a template-capable storage?)"
fi
msg_ok "Template ready: ${TEMPLATE}"

# ── Create the unprivileged container ────────────────────────────────────────
NET0="name=eth0,bridge=${BRIDGE},ip=${IP}"
[ -n "$MAC" ] && NET0="${NET0},hwaddr=${MAC}"

msg_info "Creating unprivileged LXC ${CTID} (${CT_HOSTNAME})…"
pct create "$CTID" "${TEMPLATE_STORAGE}:vztmpl/${TEMPLATE}" \
	--hostname "$CT_HOSTNAME" \
	--cores "$CORES" --memory "$RAM_MB" \
	--rootfs "${STORAGE}:${DISK_GB}" \
	--net0 "$NET0" \
	--unprivileged 1 \
	--features nesting=0,keyctl=0 \
	--onboot 1 >/dev/null
CREATED_CTID="$CTID"
msg_ok "Container ${CTID} created."

msg_info "Starting container…"
pct start "$CTID" >/dev/null

msg_info "Waiting for container networking…"
net_up=0
for _ in $(seq 1 30); do
	if pct exec "$CTID" -- getent hosts deb.debian.org >/dev/null 2>&1; then
		net_up=1; break
	fi
	sleep 2
done
[ "$net_up" = 1 ] || die "container ${CTID} never got network (check bridge '${BRIDGE}' / IP '${IP}')"
msg_ok "Networking up."

# ── Provision inside the container ───────────────────────────────────────────
msg_info "Provisioning inside the container (build + harden; this takes a few minutes)…"
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
curl -fsSL "${RAW_BASE}/${BRANCH}/proxmox/ct-setup.sh" -o "$TMP" \
	|| die "could not fetch proxmox/ct-setup.sh from branch '${BRANCH}'"
pct push "$CTID" "$TMP" /root/ct-setup.sh
pct exec "$CTID" -- env \
	REPO_URL="$REPO_URL" BRANCH="$BRANCH" GO_VERSION="$GO_VERSION" \
	SSH_PORT="$SSH_PORT" INTERVAL="$INTERVAL" \
	bash /root/ct-setup.sh
CREATED_CTID=""   # success — disarm the cleanup hint

# ── Done ─────────────────────────────────────────────────────────────────────
IP_ADDR="$(pct exec "$CTID" -- hostname -I 2>/dev/null | awk '{print $1}')"
PORT_ARG=""; [ "$SSH_PORT" = 22 ] || PORT_ARG=" -p ${SSH_PORT}"

echo
msg_ok "${B}valo-tui is live in CT ${CTID}.${RST}"
cat <<EOF

  ${B}Container${RST}   id ${CTID} · ${CT_HOSTNAME} · ip ${IP_ADDR:-<pending>}
  ${B}Connect${RST}     ssh ${IP_ADDR:-<container-ip>}${PORT_ARG}
                (any username — it drops straight into the read-only TUI)

  ${B}Expose it publicly as valo.blackpantha.com${RST}
    • DNS:    add an A record  valo.blackpantha.com  ->  your site's public IP
    • Router: port-forward public :${SSH_PORT}  ->  ${IP_ADDR:-<container-ip>}:${SSH_PORT}  (forward ONLY this port)
    • Then anyone runs:  ssh${PORT_ARG} valo.blackpantha.com

  ${B}Operate it (from the Proxmox host)${RST}
    pct enter ${CTID}
    systemctl status valo-fetcher valo-tui-ssh
    journalctl -u valo-tui-ssh -f

  ${B}Update / remove${RST}
    bash proxmox/update.sh      ${DIM}CTID=${CTID}${RST}    # pull main, rebuild, restart
    bash proxmox/uninstall.sh   ${DIM}CTID=${CTID}${RST}    # stop / destroy the container
EOF
