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
import json
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


def _has_cached(con: sqlite3.Connection, key: str) -> bool:
    row = con.execute("SELECT value FROM kv WHERE key = ?", (key,)).fetchone()
    if not row:
        return False
    try:
        return bool(json.loads(row[0]))
    except (json.JSONDecodeError, TypeError):
        return False


def write_unless_empty(con: sqlite3.Connection, key: str, value: list) -> bool:
    """Write a list feed, but never clobber good cached data with an empty
    result (an empty fetch almost always means a transient API/rate-limit
    error for these feeds). Returns True if written."""
    if not value and _has_cached(con, key):
        return False
    write(con, key, value)
    return True


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
    kept = not write_unless_empty(con, "matches:upcoming", matches)
    _log(f"upcoming: {len(matches)} match(es)" + (" (empty — kept cached)" if kept else ""))


def fetch_completed(con: sqlite3.Connection) -> None:
    matches = vlr.matches.completed(limit=60)
    kept = not write_unless_empty(con, "matches:completed", matches)
    _log(f"completed: {len(matches)} match(es)" + (" (empty — kept cached)" if kept else ""))


def fetch_events(con: sqlite3.Connection) -> None:
    events = vlr.events.list_events(status=vlr.EventStatus.ONGOING, limit=40)
    events += vlr.events.list_events(status=vlr.EventStatus.UPCOMING, limit=40)
    kept = not write_unless_empty(con, "events:active", events)
    _log(f"events: {len(events)} active" + (" (empty — kept cached)" if kept else ""))


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
