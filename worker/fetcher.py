"""Fetcher worker — the only writer to the SQLite cache.

Runs as a separate process. Polls vlr.gg via ``vlrdevapi`` on per-feed
cadences and upserts JSON blobs into the ``kv`` table. The TUI clients read
those blobs and never touch the network (except read-through series detail).

Usage:
    python worker/fetcher.py            # run the polling loops forever
    python worker/fetcher.py --once     # fetch every feed once, then exit
"""

from __future__ import annotations

import asyncio
import sqlite3
import sys
from datetime import datetime, timezone

import vlrdevapi as vlr

from valo_tui import config
from valo_tui.data.serialize import dumps

SCHEMA = """
CREATE TABLE IF NOT EXISTS kv (
    key        TEXT PRIMARY KEY,
    value      JSON NOT NULL,
    updated_at TEXT NOT NULL
);
"""


def init_db() -> sqlite3.Connection:
    con = sqlite3.connect(config.db_path())
    con.executescript(SCHEMA)
    con.commit()
    return con


def _now() -> str:
    return datetime.now(timezone.utc).isoformat()


def write(con: sqlite3.Connection, key: str, value: object) -> None:
    con.execute(
        "INSERT OR REPLACE INTO kv (key, value, updated_at) VALUES (?, json(?), ?)",
        (key, dumps(value), _now()),
    )
    con.commit()


# ---------------------------------------------------------------------------
# Individual fetch steps. Each is defensive: a single failure is logged and
# the previous cached value is left untouched.
# ---------------------------------------------------------------------------


def fetch_live(con: sqlite3.Connection) -> None:
    matches = vlr.matches.live()
    write(con, "matches:live", matches)
    _log(f"live: {len(matches)} match(es)")


def fetch_upcoming(con: sqlite3.Connection) -> None:
    matches = vlr.matches.upcoming(limit=80)
    write(con, "matches:upcoming", matches)
    _log(f"upcoming: {len(matches)} match(es)")


def fetch_completed(con: sqlite3.Connection) -> None:
    matches = vlr.matches.completed(limit=60)
    write(con, "matches:completed", matches)
    _log(f"completed: {len(matches)} match(es)")


def fetch_events(con: sqlite3.Connection) -> None:
    events = vlr.events.list_events(status=vlr.EventStatus.ONGOING, limit=40)
    events += vlr.events.list_events(status=vlr.EventStatus.UPCOMING, limit=40)
    write(con, "events:active", events)
    _log(f"events: {len(events)} active")


def _log(msg: str) -> None:
    print(f"[{_now()}] {msg}", flush=True)


def _safe(con: sqlite3.Connection, fn) -> None:
    try:
        fn(con)
    except Exception as exc:  # noqa: BLE001 — worker must never die on one feed
        _log(f"{fn.__name__} failed: {type(exc).__name__}: {exc}")


# ---------------------------------------------------------------------------
# Loops
# ---------------------------------------------------------------------------


async def _loop(con: sqlite3.Connection, fn, interval: int) -> None:
    while True:
        _safe(con, fn)
        await asyncio.sleep(interval)


def run_once() -> None:
    con = init_db()
    for fn in (fetch_events, fetch_live, fetch_upcoming, fetch_completed):
        _safe(con, fn)
    con.close()
    _log("one-shot fetch complete")


async def main() -> None:
    con = init_db()
    _log(f"worker started -> {config.db_path()}")
    await asyncio.gather(
        _loop(con, fetch_live, config.LIVE_INTERVAL),
        _loop(con, fetch_upcoming, config.UPCOMING_INTERVAL),
        _loop(con, fetch_completed, config.COMPLETED_INTERVAL),
        _loop(con, fetch_events, config.EVENTS_INTERVAL),
    )


if __name__ == "__main__":
    if "--once" in sys.argv:
        run_once()
    else:
        asyncio.run(main())
