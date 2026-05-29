"""Runtime configuration shared by the worker, cache, and TUI.

The single source of truth here is the SQLite cache location. In Docker the
``valtui-data`` volume is mounted at ``/var/lib/valtui`` and ``VALTUI_DB``
points there; locally we fall back to the user cache directory so nothing
needs root.
"""

from __future__ import annotations

import os
from pathlib import Path

# Polling cadences (seconds) for the fetcher worker.
LIVE_INTERVAL = 30
UPCOMING_INTERVAL = 5 * 60
COMPLETED_INTERVAL = 10 * 60
EVENTS_INTERVAL = 15 * 60

# How long an on-demand series detail stays fresh before a read-through refetch.
DETAIL_TTL = 60

# The four Tier-1 regional leagues, used to bucket the global-live dashboard.
REGIONS = ("Americas", "EMEA", "Pacific", "China")


def db_path() -> Path:
    """Resolve the cache database path, creating parent dirs as needed."""
    raw = os.environ.get("VALTUI_DB")
    path = Path(raw) if raw else Path.home() / ".cache" / "valtui" / "cache.db"
    path.parent.mkdir(parents=True, exist_ok=True)
    return path
