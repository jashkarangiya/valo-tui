# Connect to valo-tui over a Cloudflare Tunnel

The server is served over SSH (Charm's Wish) but exposed through a **Cloudflare
Tunnel** — so there are no open ports on the host's router and its IP stays
hidden. Because the tunnel carries SSH (not HTTP), each viewer needs Cloudflare's
`cloudflared` client to reach it. These one-liners install it and write an
`ssh valo` shortcut, so connecting becomes a single command afterwards.

```
connect/
  install.sh     macOS / Linux / WSL  — curl … | sh
  install.ps1    Windows (PowerShell) — irm … | iex
```

## For viewers — one command

**macOS / Linux / WSL**

```bash
curl -fsSL https://valo.black-pantha.com/install.sh | sh
```

**Windows (PowerShell)**

```powershell
irm https://valo.black-pantha.com/install.ps1 | iex
```

That's the whole thing. The command installs `cloudflared` (per-user, no
admin/root) the first time, writes one block to your `~/.ssh/config`, and drops
you **straight into the TUI**. Run the exact same command any time to reconnect —
or, once it's set up, the short `ssh valo` also works. Nothing else is touched;
the host key is accepted automatically on first connect.

> Set `VALO_NO_CONNECT=1` if you want it to set up the `ssh valo` shortcut
> *without* launching immediately.

> **Why is an install needed at all?** Cloudflare's free Tunnel proxies SSH only
> if the *client* runs `cloudflared access ssh` as an ssh ProxyCommand. The
> script just wires that up for you. The server end is a **read-only TUI** — no
> shell, no exec, no file access — so this connection can only ever draw a
> scoreboard. It cannot reach a shell or your machine.

## For the host (one-time setup)

1. **Have a domain on Cloudflare.** Free plan is fine; its nameservers must point
   at Cloudflare.

2. **Deploy valo-tui** to the LXC with the [`proxmox/`](../proxmox/) installer (or
   any host running `valo-tui-ssh`).

3. **Set up the tunnel inside the container:**

   ```bash
   pct push <ctid> proxmox/cloudflare.sh /root/cloudflare.sh
   pct enter <ctid>
   TUNNEL_HOSTNAME=valo.black-pantha.com bash /root/cloudflare.sh
   ```

   It installs `cloudflared`, prompts a one-time browser login, creates a named
   tunnel, routes **only** `ssh://localhost:<wish-port>` through it, points
   `valo.black-pantha.com` at the tunnel (a proxied CNAME), and installs a hardened
   `cloudflared.service`. See [`proxmox/cloudflare.sh`](../proxmox/cloudflare.sh).

4. **Publish the install scripts** at your domain and replace
   `valo.black-pantha.com` in both scripts (or set the `VALO_HOST` default) so the one
   embedded host matches your tunnel. Any static host works — e.g. a Cloudflare
   Pages site, an R2 bucket, or a GitHub raw URL:

   ```bash
   # GitHub raw works without hosting anything yourself:
   curl -fsSL https://raw.githubusercontent.com/jashkarangiya/valo-tui/main/connect/install.sh | VALO_HOST=valo.black-pantha.com sh
   ```

## Security model

- **No open inbound ports** on the host's router; the home/public IP is hidden
  behind Cloudflare's edge, which also absorbs DDoS.
- `cloudflared` runs **inside the LXC** and its config routes exactly one
  service — `ssh://localhost:<wish-port>` (the Wish TUI). It cannot reach the
  Proxmox host's SSH or anything else; a catch-all `404` refuses every other
  hostname.
- The thing on the far end is Wish wired with only the Bubble Tea middleware:
  **no shell, no `ssh host <cmd>`, no SFTP, no port-forwarding** — it can only
  render the read-only dashboard. Full model in
  [`proxmox/README.md`](../proxmox/README.md#security-model--why-ssh-in-leaks-nothing).

## Connect without the script

If a viewer already has `cloudflared`, no install is needed:

```bash
ssh -o ProxyCommand='cloudflared access ssh --hostname %h' valo.black-pantha.com
```

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| `ssh valo` hangs then fails | The tunnel isn't up: on the host, `systemctl status cloudflared` and `journalctl -u cloudflared -f`. |
| `cloudflared: command not found` in ProxyCommand | Re-run the install script; it pins an absolute path to cloudflared in the ssh config. |
| `Host valo` not recognised | Confirm the managed block exists in `~/.ssh/config` and that `VALO_HOST` was set to your real domain. |
| Garbled / no colors | Use a real terminal (Windows Terminal, iTerm, etc.); the TUI needs a PTY, which the config requests. |
| Hostname won't resolve | In the Cloudflare dashboard, confirm the proxied CNAME `valo → <tunnel-id>.cfargotunnel.com` exists. |
