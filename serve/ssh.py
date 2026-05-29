"""SSH server — spawns a fresh Textual instance per incoming connection.

Public, read-only: authentication is disabled. The host key is generated on
first boot if it does not already exist on the mounted volume.
"""

import asyncio
import sys
from pathlib import Path

import asyncssh

HOST_KEY = Path("/etc/valtui/ssh_host_key")
PORT = 2222


def ensure_host_key() -> str:
    """Create a persistent host key on first run; return its path."""
    if not HOST_KEY.exists():
        HOST_KEY.parent.mkdir(parents=True, exist_ok=True)
        key = asyncssh.generate_private_key("ssh-ed25519")
        HOST_KEY.write_bytes(key.export_private_key())
        HOST_KEY.chmod(0o600)
    return str(HOST_KEY)


class ValtuiSSHServer(asyncssh.SSHServer):
    def begin_auth(self, username: str) -> bool:
        return False  # no auth — public read-only TUI

    def password_auth_supported(self) -> bool:
        return False


async def handle_client(process: asyncssh.SSHServerProcess) -> None:
    width, height = process.get_terminal_size()[:2]
    proc = await asyncio.create_subprocess_exec(
        sys.executable, "-m", "valtui",
        stdin=process.stdin,
        stdout=process.stdout,
        stderr=process.stderr,
        env={
            "TERM": process.get_terminal_type() or "xterm-256color",
            "COLUMNS": str(width or 120),
            "LINES": str(height or 40),
        },
    )
    await proc.wait()
    process.exit(proc.returncode or 0)


async def main() -> None:
    host_key = ensure_host_key()
    await asyncssh.create_server(
        ValtuiSSHServer,
        host="0.0.0.0",
        port=PORT,
        server_host_keys=[host_key],
        process_factory=handle_client,
        allow_pty=True,
    )
    print(f"valtui ssh server listening on :{PORT}", flush=True)
    await asyncio.Future()


if __name__ == "__main__":
    asyncio.run(main())
