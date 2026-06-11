# Deploy valo-tui to a Proxmox LXC

One command on the Proxmox host builds an **unprivileged** container that runs
the fetcher + the SSH TUI server, and nothing else. Users `ssh` in to see the
TUI; there is no shell to reach.

## Install

On the Proxmox VE host:

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/jashkarangiya/valo-tui/main/proxmox/valo-tui.sh)"
```

Prefer to read before you run (recommended): download
[`valo-tui.sh`](valo-tui.sh), skim it, then execute. Everything is overridable
with env vars — defaults in parentheses:

```bash
CTID=141 \              # (next free id)
HOSTNAME=valo \         # (valo-tui)
STORAGE=local-lvm \     # rootfs storage (local-lvm)
BRIDGE=vmbr0 \          # (vmbr0)
IP=192.168.1.50/24,gw=192.168.1.1 \   # (dhcp)
DISK_GB=4 RAM_MB=512 CORES=1 \
SSH_PORT=22 \           # 22 = bare `ssh host`; or e.g. 23234
  bash valo-tui.sh
```

It creates the LXC, then runs [`install.sh`](install.sh) inside to build from
GitHub and start the services. Updating later: `pct enter <ctid>` then
`bash /root/install.sh` re-pulls and rebuilds.

## Why "ssh in" leaks nothing

This is the whole point of the design, in layers:

1. **No real SSH daemon in the container.** `install.sh` purges
   `openssh-server`. The only thing bound to SSH is **Wish**, and Wish is
   configured with *only* the Bubble Tea middleware — so a connection can do
   exactly one thing: render the TUI. No shell, no `ssh host <cmd>` exec, no
   SFTP subsystem, no port-forwarding (all denied by default). You manage the
   box from the Proxmox host with `pct enter`, never over SSH.
2. **The TUI is read-only.** It reads a local SQLite cache and draws Valorant
   data. It opens no files you pick, runs no commands, and quits on `q`. There
   is no input path that reaches the filesystem or shell.
3. **Locked-down service user.** Both daemons run as `valo` — a system account
   with `/usr/sbin/nologin`, no password, no sudo, owning only its cache dir.
4. **systemd sandbox.** `ProtectSystem=strict`, read-only everywhere except the
   cache dir, `NoNewPrivileges`, restricted syscalls/address-families, dropped
   capabilities (the SSH server keeps only `CAP_NET_BIND_SERVICE` for `:22`; the
   fetcher keeps none).
5. **Unprivileged LXC.** Container root maps to an unprivileged host UID, so even
   a full in-container compromise is contained by the kernel — it is not host
   root.

Net effect: an SSH client reaches a process that can only paint a read-only
dashboard, running as a no-login user, in a strict sandbox, in an unprivileged
container, with no shell anywhere in the path.

## Exposing it as `valo.blackpantha.com`

The container gets a LAN IP on your bridge. To reach it from the internet:

- **DNS:** point `valo.blackpantha.com` (A record) at your site's public IP.
- **Port forward** on your router/firewall: public `:22` → `<container-ip>:22`
  (or `:23234` if you set `SSH_PORT=23234`). Forward *only* that port.
- Then anyone runs `ssh valo.blackpantha.com`.

Because the container has its own IP and no admin sshd, this never touches the
Proxmox host's SSH or any other container.

## Operating it

```bash
pct enter <ctid>                              # get a host-side shell in the CT
systemctl status valo-fetcher valo-tui-ssh    # service health
journalctl -u valo-fetcher -f                 # watch the fetcher
journalctl -u valo-tui-ssh -n 50              # recent connections
```

The fetcher cadence and SSH port live in the unit files
(`/etc/systemd/system/valo-*.service`); edit + `systemctl daemon-reload` +
`restart` to change them.
