# Deploy valo-tui to a Proxmox LXC

One command on the Proxmox host builds an **unprivileged** container that runs
the vlr.gg fetcher + the Wish SSH server, and nothing else. Users `ssh` in and
land straight in the read-only TUI — there is no shell to reach.

```
proxmox/
  install.sh     run on the PVE host: creates the LXC + provisions it   ← start here
  ct-setup.sh    runs inside the container: build, harden, install services
  update.sh      run on the PVE host: pull latest, rebuild, restart (keeps cache)
  uninstall.sh   run on the PVE host: stop and optionally destroy the CT
```

## Install (one-liner)

On the **Proxmox VE host**, as root:

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/jashkarangiya/valo-tui/main/proxmox/install.sh)"
```

It checks you're on a Proxmox host, prompts for the container settings (with
sensible defaults — just press Enter to accept them), creates the unprivileged
Debian 12 LXC, builds valo-tui from `main`, installs the hardened services, and
prints the connection details. Uses `whiptail` dialogs when available, plain
prompts otherwise.

> **Prefer to read before you run** (recommended): download
> [`install.sh`](install.sh), skim it, then execute it.

### Interactive

Running with no env vars walks you through every value. Accept the defaults or
choose **No** at the summary to customise:

| Prompt        | Default          | Notes                                            |
| ------------- | ---------------- | ------------------------------------------------ |
| CT ID         | next free id     | must be unused                                   |
| Hostname      | `valo-tui`       |                                                  |
| Storage       | `local-lvm`      | rootfs storage (`pvesm status`)                  |
| Bridge        | `vmbr0`          | must exist on the host                           |
| Network       | `dhcp`           | or a static `CIDR,gw=…`; optional fixed MAC      |
| CPU cores     | `1`              |                                                  |
| RAM (MB)      | `512`            |                                                  |
| Disk (GB)     | `4`              |                                                  |
| SSH/TUI port  | `22`             | `22` = bare `ssh host`; else e.g. `23234`        |
| Repo / branch | this repo / main | what to build from                               |

### Non-interactive (env vars)

Set `NONINTERACTIVE=1` (or pipe with no TTY) and every value comes from its env
var; nothing is prompted. All defaults shown in parentheses:

```bash
NONINTERACTIVE=1 \
CTID=141 \                 # (next free id)
CT_HOSTNAME=valo \         # (valo-tui)   — NB: CT_HOSTNAME, not HOSTNAME
STORAGE=local-lvm \        # (local-lvm)
TEMPLATE_STORAGE=local \   # (local)      — where CT templates live
BRIDGE=vmbr0 \             # (vmbr0)
IP=192.168.1.50/24,gw=192.168.1.1 \   # (dhcp)
MAC=BC:24:11:AB:CD:EF \    # (auto)       — handy for a DHCP reservation
DISK_GB=4 RAM_MB=512 CORES=1 \         # (4 / 512 / 1)
SSH_PORT=22 \              # (22)
REPO_URL=https://github.com/jashkarangiya/valo-tui.git \
BRANCH=main \              # (main)
GO_VERSION=1.26.3 \        # (1.26.3)     — must satisfy go.mod
INTERVAL=30s \             # (30s)        — fetcher live-refresh cadence
  bash -c "$(curl -fsSL https://raw.githubusercontent.com/jashkarangiya/valo-tui/main/proxmox/install.sh)"
```

> `CT_HOSTNAME` rather than `HOSTNAME`: bash auto-populates `$HOSTNAME` with the
> *Proxmox host's* name, so using it would silently misname the container.

## Update / redeploy

Pull the latest branch, rebuild, reinstall the (possibly updated) units, and
restart — **without** touching the cache or the SSH host key. The listen port is
read back from the running service, so you don't have to remember it.

```bash
# auto-detects the single running valo-tui container
bash proxmox/update.sh

# or target one explicitly / pick a branch
CTID=141 BRANCH=main bash proxmox/update.sh
```

(`update.sh` just re-runs `ct-setup.sh` inside the container, which is
idempotent: Go is only re-downloaded if its version changed, and
`/var/lib/valo-tui` is never cleared.)

## Uninstall / remove

```bash
bash proxmox/uninstall.sh                  # auto-detect, stop, then ask to destroy
CTID=141 bash proxmox/uninstall.sh         # target a specific CT
CTID=141 PURGE=1 bash proxmox/uninstall.sh # non-interactive: stop AND destroy
```

Without `PURGE=1` it stops the container and asks before destroying. Destroying
removes the rootfs — the cache and host key go with it.

## Exposing it as `valo.blackpantha.com`

The container gets a LAN IP on your bridge. To reach it from the internet:

- **DNS:** point `valo.blackpantha.com` (A record) at your site's public IP.
- **Port-forward** on your router/firewall: public `:22` → `<container-ip>:22`
  (or `:23234` if you set `SSH_PORT=23234`). Forward **only** that port.
- Then anyone runs `ssh valo.blackpantha.com`.

Because the container has its own IP and no admin sshd, this never touches the
Proxmox host's SSH or any other container.

## Operating it

```bash
pct enter <ctid>                              # host-side shell into the CT
systemctl status valo-fetcher valo-tui-ssh    # service health
journalctl -u valo-fetcher -f                 # watch the fetcher poll vlr.gg
journalctl -u valo-tui-ssh -n 50              # recent SSH connections
```

The fetcher cadence and listen port live in `/etc/systemd/system/valo-*.service`
(re-rendered by `ct-setup.sh`); edit + `systemctl daemon-reload` + `restart` to
change them by hand, or just re-run `update.sh` with `SSH_PORT=` / `INTERVAL=`.

## Security model — why "ssh in" leaks nothing

Defence in depth, from the inside out:

1. **No real SSH daemon in the container.** `ct-setup.sh` purges
   `openssh-server`. The only thing bound to SSH is **Wish**, wired with *only*
   the Bubble Tea middleware — so a connection can do exactly one thing: render
   the TUI. No shell, no `ssh host <cmd>` exec, no SFTP subsystem, no
   port-forwarding (all denied by default). You manage the box from the Proxmox
   host with `pct enter`, never over SSH.
2. **The TUI is read-only.** It reads a local SQLite cache and draws Valorant
   data. It opens no files you pick, runs no commands, and quits on `q`. There is
   no input path that reaches the filesystem or a shell.
3. **Locked-down service user.** Both daemons run as `valo` — a system account
   with `/usr/sbin/nologin`, no password, no sudo, owning only its cache dir.
4. **systemd sandbox** (`deploy/valo-*.service`):
   - `NoNewPrivileges=true`, `PrivateTmp=true`
   - `ProtectSystem=strict` + `ReadWritePaths=/var/lib/valo-tui` (the *only*
     writable path), `ProtectHome=true`
   - `CapabilityBoundingSet=` empty — **both** daemons hold **zero** capabilities.
     Binding `:22` as the unprivileged user is allowed by lowering
     `net.ipv4.ip_unprivileged_port_start` (namespaced to this container), which
     avoids the ambient capability that otherwise breaks systemd's mount
     namespace inside an unprivileged LXC
   - `RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6` (HTTPS scrape + TCP
     listener + local DNS, nothing exotic)
   - `SystemCallFilter=@system-service` minus `@privileged @resources`, plus
     `ProtectKernel*` / `ProtectControlGroups` / `LockPersonality` — all chosen
     to work *inside* an unprivileged LXC. Both units drop `ProtectProc` /
     `ProcSubset`, whose fresh hidepid `/proc` mount such kernels refuse for any
     sandboxed unit (status=226/NAMESPACE). `ct-setup.sh` verifies both services
     reach `active`, falls back to a relaxed profile for any unit the kernel
     still refuses the namespace for, and dumps the journal if even that fails.
5. **Unprivileged LXC.** Container root maps to an unprivileged host UID, so even
   a full in-container compromise is contained by the kernel — it is not host
   root.

Net effect: an SSH client reaches a process that can only paint a read-only
dashboard, running as a no-login user, in a strict sandbox, in an unprivileged
container, with no shell anywhere in the path.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| `must run on a Proxmox VE host ('pct' not found)` | Run on the PVE host, not inside a container or your laptop. |
| `CT <id> already exists` | Pick another `CTID`, or remove the old CT with `uninstall.sh`. |
| `storage '…' not found` | Run `pvesm status`; pass `STORAGE=<name>` of a storage that holds container rootfs. |
| `bridge '…' not found` | Run `ip link`; pass `BRIDGE=<name>` (often `vmbr0`). |
| `container never got network` | Check the bridge has DHCP / your static `IP=` is correct and the `gw=` is reachable. |
| Build fails on `go build` | Usually transient network fetching modules. Re-run `update.sh`; confirm `GO_VERSION` satisfies `go.mod`. |
| A service isn't `active` | `pct enter <ctid>` then `journalctl -u valo-tui-ssh -n 50`. `ct-setup.sh` already prints this on failure. |
| Can't connect over SSH | Confirm the port (`SSH_PORT`), that your router forwards **only** that port to the container IP, and that DNS resolves. The first connection shows the TUI host key fingerprint — that's expected. |
| Want a different port later | `SSH_PORT=23234 bash proxmox/update.sh` re-renders the unit and restarts. |
