#!/usr/bin/env bash
#
# valo-tui — Cloudflare Tunnel setup.
#
# Run this INSIDE the valo-tui LXC (after install.sh has built the box):
#
#   pct push <ctid> proxmox/cloudflare.sh /root/cloudflare.sh
#   pct enter <ctid>
#   TUNNEL_HOSTNAME=valo.black-pantha.com bash /root/cloudflare.sh
#
# It installs cloudflared, creates a named tunnel, and routes ONLY
# ssh://localhost:<wish-port> through it — so the tunnel can reach exactly one
# thing: the read-only TUI. There are NO open inbound ports on your router and
# your home IP stays hidden; Cloudflare's edge is the only way in, and it only
# ever lands on Wish.
#
# Prereqs (one-time, on your side):
#   • A domain whose nameservers point at Cloudflare (free plan is fine).
#   • You'll complete a browser login when prompted (cloudflared prints a URL).
#
# Everything is overridable via env; see the defaults below.
set -euo pipefail

TUNNEL_NAME="${TUNNEL_NAME:-valo-tui}"
TUNNEL_HOSTNAME="${TUNNEL_HOSTNAME:-}"     # e.g. valo.black-pantha.com  (REQUIRED)
# Wish listen port. Default: read it back from the running service, else 23234.
WISH_PORT="${WISH_PORT:-}"
CF_DIR=/etc/cloudflared
CF_USER=cloudflared

say() { echo -e "\e[1;32m[cloudflare]\e[0m $*"; }
die() { echo -e "\e[1;31m[cloudflare] $*\e[0m" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "must run as root inside the container"
[ -n "$TUNNEL_HOSTNAME" ] || die "set TUNNEL_HOSTNAME=valo.yourdomain.com and re-run"

# ── Resolve the Wish port the TUI is actually listening on ───────────────────
if [ -z "$WISH_PORT" ]; then
	WISH_PORT="$(systemctl show -p Environment valo-tui-ssh 2>/dev/null \
		| tr ' ' '\n' | sed -n 's/^VALO_TUI_SSH_PORT=//p' | head -1)"
	WISH_PORT="${WISH_PORT:-23234}"
fi
say "Routing Cloudflare hostname ${TUNNEL_HOSTNAME} -> ssh://localhost:${WISH_PORT}"

# ── Install cloudflared (official Cloudflare apt repo) ───────────────────────
if command -v cloudflared >/dev/null 2>&1; then
	say "cloudflared already installed ($(cloudflared --version 2>/dev/null | head -1))"
else
	say "Installing cloudflared"
	export DEBIAN_FRONTEND=noninteractive
	apt-get update -qq
	apt-get install -y -qq --no-install-recommends curl ca-certificates gnupg
	install -d -m 0755 /usr/share/keyrings
	curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg \
		-o /usr/share/keyrings/cloudflare-main.gpg \
		|| die "could not fetch the cloudflare apt key"
	echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main" \
		>/etc/apt/sources.list.d/cloudflared.list
	apt-get update -qq
	apt-get install -y -qq cloudflared || die "cloudflared install failed"
fi

# ── Authenticate to your Cloudflare account (interactive, one-time) ──────────
# Saves a cert.pem under /root/.cloudflared used only to create the tunnel and
# the DNS route below; the running tunnel uses its own scoped credentials file.
if [ ! -f /root/.cloudflared/cert.pem ]; then
	say "Opening Cloudflare login — visit the URL it prints, pick your domain, authorise."
	cloudflared tunnel login || die "cloudflared login did not complete"
fi

# ── Create the named tunnel (idempotent) ────────────────────────────────────
if cloudflared tunnel list 2>/dev/null | awk '{print $2}' | grep -qx "$TUNNEL_NAME"; then
	say "Tunnel '${TUNNEL_NAME}' already exists — reusing it"
else
	say "Creating tunnel '${TUNNEL_NAME}'"
	cloudflared tunnel create "$TUNNEL_NAME" || die "could not create the tunnel"
fi
TUNNEL_ID="$(cloudflared tunnel list 2>/dev/null \
	| awk -v n="$TUNNEL_NAME" '$2==n{print $1}' | head -1)"
[ -n "$TUNNEL_ID" ] || die "could not resolve the tunnel id for '${TUNNEL_NAME}'"
say "Tunnel id ${TUNNEL_ID}"

# ── Stage credentials + config in /etc/cloudflared, owned by a system user ───
say "Writing ${CF_DIR}/config.yml"
id "$CF_USER" &>/dev/null || useradd --system --no-create-home \
	--home-dir "$CF_DIR" --shell /usr/sbin/nologin "$CF_USER"
install -d -o "$CF_USER" -g "$CF_USER" -m 0750 "$CF_DIR"
install -o "$CF_USER" -g "$CF_USER" -m 0600 \
	"/root/.cloudflared/${TUNNEL_ID}.json" "${CF_DIR}/${TUNNEL_ID}.json"

# Single ingress rule: the Wish TUI, nothing else. The catch-all 404 ensures any
# other hostname that ever resolves to this tunnel is refused, not proxied.
cat >"${CF_DIR}/config.yml" <<EOF
tunnel: ${TUNNEL_ID}
credentials-file: ${CF_DIR}/${TUNNEL_ID}.json

# Route ONLY the read-only TUI. Do not add other services here.
ingress:
  - hostname: ${TUNNEL_HOSTNAME}
    service: ssh://localhost:${WISH_PORT}
  - service: http_status:404
EOF
chown "$CF_USER:$CF_USER" "${CF_DIR}/config.yml"
chmod 0640 "${CF_DIR}/config.yml"

# ── Point the hostname at the tunnel (creates a proxied CNAME in your zone) ──
say "Routing DNS ${TUNNEL_HOSTNAME} -> ${TUNNEL_NAME}"
cloudflared tunnel route dns "$TUNNEL_NAME" "$TUNNEL_HOSTNAME" 2>/dev/null \
	|| say "DNS route already exists (or add the CNAME by hand) — continuing"

# ── Hardened systemd unit (runs the tunnel as the nologin cloudflared user) ──
say "Installing cloudflared.service"
cat >/etc/systemd/system/cloudflared.service <<EOF
[Unit]
Description=cloudflared tunnel for valo-tui (read-only TUI only)
After=network-online.target valo-tui-ssh.service
Wants=network-online.target

[Service]
User=${CF_USER}
ExecStart=/usr/bin/cloudflared --no-autoupdate --config ${CF_DIR}/config.yml tunnel run
Restart=always
RestartSec=5

# Sandbox: cloudflared needs only outbound networking + its config dir (read).
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadOnlyPaths=${CF_DIR}
CapabilityBoundingSet=
AmbientCapabilities=
RestrictAddressFamilies=AF_INET AF_INET6
SystemCallArchitectures=native
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @resources
RestrictSUIDSGID=true
RestrictRealtime=true
LockPersonality=true
UMask=0077

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now cloudflared >/dev/null 2>&1 || true

# Confirm it actually came up (and holds), mirroring ct-setup.sh's probe.
ok=0
for _ in 1 2 3 4 5; do
	systemctl is-active --quiet cloudflared && { ok=1; sleep 1; } || { ok=0; break; }
done
if [ "$ok" = 1 ]; then
	say "cloudflared is active."
else
	echo -e "\e[1;31m[cloudflare] cloudflared did not stay active — recent logs:\e[0m" >&2
	journalctl -u cloudflared -n 25 --no-pager >&2 || true
	die "tunnel failed to start; see logs above"
fi

echo
say "Done. Your tunnel is live."
cat <<EOF

  Hostname    ${TUNNEL_HOSTNAME}  ->  ssh://localhost:${WISH_PORT}  (Wish TUI only)
  Tunnel      ${TUNNEL_NAME} (${TUNNEL_ID})

  Users now connect with the one-liner (see connect/), or directly:
    ssh -o ProxyCommand='cloudflared access ssh --hostname %h' ${TUNNEL_HOSTNAME}

  Operate:
    systemctl status cloudflared
    journalctl -u cloudflared -f
EOF
