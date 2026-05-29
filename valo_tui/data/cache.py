"""Read-side access to the SQLite cache.

Everything the TUI renders comes from here. The polling lists (live /
upcoming / completed / events) are populated by the fetcher worker and read
purely; per-series detail is loaded via a read-through cache (fetched on
demand and stored with a short TTL) because the worker cannot pre-fetch every
match a user might drill into.
"""

from __future__ import annotations

import json
import sqlite3
from datetime import datetime, timezone

from .. import config
from .bracket import Bracket, build_bracket
from .models import EventCard, MatchCard, SeriesDetail

# ---------------------------------------------------------------------------
# Region classification
# ---------------------------------------------------------------------------

_REGION_KEYWORDS: dict[str, tuple[str, ...]] = {
    "Americas": ("americas", "north america", "latam", "brazil", "united states", "n.a"),
    "EMEA": ("emea", "europe", "middle east", "türkiye", "turkey", "mena"),
    "Pacific": ("pacific", "korea", "japan", "asia", "oceania", "south asia"),
    "China": ("china", "chinese", "中国", " cn "),
}

_INTERNATIONAL = ("masters", "champions", "valorant champions", "vct international")


def classify_region(*parts: str | None) -> str | None:
    """Best-effort bucket of an event/match into one of the four leagues."""
    text = " ".join(p for p in parts if p).lower()
    for region, keywords in _REGION_KEYWORDS.items():
        if any(kw in text for kw in keywords):
            return region
    return None


def is_international(name: str | None) -> bool:
    text = (name or "").lower()
    if "challengers" in text:  # regional tier, not international
        return False
    return any(kw in text for kw in _INTERNATIONAL)


# ---------------------------------------------------------------------------
# Low-level kv access
# ---------------------------------------------------------------------------


def _connect() -> sqlite3.Connection:
    con = sqlite3.connect(config.db_path(), timeout=5)
    con.row_factory = sqlite3.Row
    return con


def get_raw(key: str) -> tuple[object | None, str | None]:
    """Return ``(value, updated_at_iso)`` for a kv key, or ``(None, None)``."""
    try:
        con = _connect()
    except sqlite3.Error:
        return None, None
    try:
        row = con.execute(
            "SELECT value, updated_at FROM kv WHERE key = ?", (key,)
        ).fetchone()
    except sqlite3.Error:
        return None, None
    finally:
        con.close()
    if not row:
        return None, None
    try:
        return json.loads(row["value"]), row["updated_at"]
    except (json.JSONDecodeError, TypeError):
        return None, row["updated_at"]


def _get_list(key: str) -> list[dict]:
    value, _ = get_raw(key)
    return value if isinstance(value, list) else []


# ---------------------------------------------------------------------------
# Typed accessors used by the screens
# ---------------------------------------------------------------------------


def live_matches() -> list[MatchCard]:
    return [MatchCard.from_raw(d) for d in _get_list("matches:live")]


def upcoming_matches() -> list[MatchCard]:
    return [MatchCard.from_raw(d) for d in _get_list("matches:upcoming")]


def completed_matches() -> list[MatchCard]:
    return [MatchCard.from_raw(d) for d in _get_list("matches:completed")]


def active_events() -> list[EventCard]:
    return [EventCard.from_raw(d) for d in _get_list("events:active")]


def last_updated() -> str | None:
    """Most recent write across the polling keys, as a short HH:MM:SS UTC."""
    stamps = []
    for key in ("matches:live", "matches:upcoming", "events:active"):
        _, ts = get_raw(key)
        if ts:
            stamps.append(ts)
    if not stamps:
        return None
    try:
        latest = max(datetime.fromisoformat(s) for s in stamps)
        return latest.strftime("%H:%M:%S")
    except ValueError:
        return None


# ---------------------------------------------------------------------------
# Global-live dashboard assembly
# ---------------------------------------------------------------------------


def global_live() -> tuple[dict[str, dict[str, list[MatchCard]]], list[MatchCard]]:
    """Build the (regions, international) structure for the global dashboard.

    ``regions`` maps each league name to ``{"live", "next", "recent"}`` lists.
    ``international`` is the flat list of matches at international events.
    """
    buckets: dict[str, dict[str, list[MatchCard]]] = {
        r: {"live": [], "next": [], "recent": []} for r in config.REGIONS
    }
    international: list[MatchCard] = []

    groups = (
        ("live", live_matches()),
        ("next", upcoming_matches()),
        ("recent", completed_matches()),
    )
    for slot, matches in groups:
        for m in matches:
            if is_international(m.event):
                if slot != "recent" or len(international) < 6:
                    international.append(m)
                continue
            region = classify_region(m.event, m.phase)
            if region:
                buckets[region][slot].append(m)
    return buckets, international


# ---------------------------------------------------------------------------
# Read-through series detail (the one place the TUI may hit the network)
# ---------------------------------------------------------------------------


def _detail_fresh(updated_at: str | None) -> bool:
    if not updated_at:
        return False
    try:
        age = datetime.now(timezone.utc) - datetime.fromisoformat(updated_at).replace(
            tzinfo=timezone.utc
        )
    except ValueError:
        return False
    return age.total_seconds() < config.DETAIL_TTL


def series_detail(match_id: int) -> SeriesDetail | None:
    """Return per-map scoreboards + vetoes for a match, caching the result."""
    key = f"series:{match_id}"
    value, updated_at = get_raw(key)
    if isinstance(value, dict) and _detail_fresh(updated_at):
        return SeriesDetail.from_raw(value.get("info", {}), value.get("maps", []))

    payload = _fetch_series(match_id)
    if payload is None:
        # Fall back to whatever stale copy we have rather than nothing.
        if isinstance(value, dict):
            return SeriesDetail.from_raw(value.get("info", {}), value.get("maps", []))
        return None

    _store_series(key, payload)
    return SeriesDetail.from_raw(payload.get("info", {}), payload.get("maps", []))


def _fetch_series(match_id: int) -> dict | None:
    try:
        import vlrdevapi as vlr

        from .serialize import to_jsonable
    except ImportError:
        return None
    try:
        info = vlr.series.info(match_id)
        if info is None:
            return None
        maps = vlr.series.matches(match_id) or []
    except Exception:
        return None
    return {
        "info": to_jsonable(info),
        "maps": [to_jsonable(m) for m in maps],
    }


def _store_series(key: str, payload: dict) -> None:
    _store_kv(key, payload)


def _store_kv(key: str, payload: object) -> None:
    try:
        con = _connect()
        # Self-initialise: the read-through cache may run before the worker
        # has ever created the schema.
        con.execute(
            "CREATE TABLE IF NOT EXISTS kv "
            "(key TEXT PRIMARY KEY, value JSON NOT NULL, updated_at TEXT NOT NULL)"
        )
        con.execute(
            "INSERT OR REPLACE INTO kv (key, value, updated_at) VALUES (?, json(?), ?)",
            (key, json.dumps(payload), datetime.now(timezone.utc).isoformat()),
        )
        con.commit()
        con.close()
    except sqlite3.Error:
        pass


# ---------------------------------------------------------------------------
# Event matches + bracket (read-through, like series detail)
# ---------------------------------------------------------------------------


def event_matches(event_id: int) -> list[dict]:
    """All matches for an event as raw dicts, read-through cached."""
    key = f"event:matches:{event_id}"
    value, updated_at = get_raw(key)
    fresh = updated_at and _detail_fresh_ttl(updated_at, config.BRACKET_TTL)
    if isinstance(value, list) and fresh:
        return value

    fetched = _fetch_event_matches(event_id)
    if fetched is None:
        return value if isinstance(value, list) else []
    _store_kv(key, fetched)
    return fetched


def bracket(event_id: int) -> Bracket:
    return build_bracket(event_matches(event_id))


def _detail_fresh_ttl(updated_at: str | None, ttl: int) -> bool:
    if not updated_at:
        return False
    try:
        age = datetime.now(timezone.utc) - datetime.fromisoformat(updated_at).replace(
            tzinfo=timezone.utc
        )
    except ValueError:
        return False
    return age.total_seconds() < ttl


def _fetch_event_matches(event_id: int) -> list[dict] | None:
    try:
        import vlrdevapi as vlr

        from .serialize import to_jsonable
    except ImportError:
        return None
    try:
        matches = vlr.events.matches(event_id) or []
    except Exception:
        return None
    # An empty result is almost always a transient/rate-limit error; signal
    # failure so the caller keeps any previously cached (good) copy.
    if not matches:
        return None
    return [to_jsonable(m) for m in matches]
