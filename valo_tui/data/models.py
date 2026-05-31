"""UI-facing dataclasses.

These are deliberately flat and decoupled from ``vlrdevapi``'s internal
models: the worker stores raw JSON, and ``cache.py`` maps that JSON into the
shapes below so the screens never touch the upstream library.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


def _i(v: Any) -> int | None:
    try:
        return int(v) if v is not None else None
    except (TypeError, ValueError):
        return None


@dataclass
class TeamSide:
    name: str
    score: int | None = None
    country: str | None = None
    short: str | None = None

    @classmethod
    def from_raw(cls, d: dict | None) -> "TeamSide":
        d = d or {}
        return cls(
            name=d.get("name") or "TBD",
            score=_i(d.get("score")),
            country=d.get("country"),
            short=d.get("short") or d.get("tag"),
        )


@dataclass
class MatchCard:
    """A single match row shared by the live, upcoming and completed views."""

    match_id: int
    team1: TeamSide
    team2: TeamSide
    event: str
    phase: str
    status: str  # "upcoming" | "live" | "completed"
    time: str | None = None
    date: str | None = None

    @property
    def is_live(self) -> bool:
        return self.status == "live"

    @classmethod
    def from_raw(cls, d: dict) -> "MatchCard":
        return cls(
            match_id=_i(d.get("match_id")) or 0,
            team1=TeamSide.from_raw(d.get("team1")),
            team2=TeamSide.from_raw(d.get("team2")),
            event=d.get("event") or "",
            phase=d.get("event_phase") or d.get("phase") or "",
            status=d.get("status") or "upcoming",
            time=d.get("time"),
            date=d.get("date"),
        )

    @classmethod
    def from_event_raw(cls, d: dict, event_name: str = "") -> "MatchCard":
        """Map a ``vlr.events.matches`` row (``teams`` list, not team1/team2)
        into the shared card shape so event sub-pages reuse the same widgets."""
        teams = d.get("teams") or []
        t1 = teams[0] if len(teams) > 0 and teams[0] else {}
        t2 = teams[1] if len(teams) > 1 and teams[1] else {}
        return cls(
            match_id=_i(d.get("match_id")) or 0,
            team1=TeamSide.from_raw(t1),
            team2=TeamSide.from_raw(t2),
            event=event_name or d.get("event") or "",
            phase=d.get("phase") or d.get("event_phase") or "",
            status=_norm_status(d),
            time=d.get("time"),
            date=d.get("date"),
        )


def _norm_status(d: dict) -> str:
    """Normalise an event-match status into upcoming | live | completed."""
    raw = (d.get("status") or "").lower()
    if "live" in raw:
        return "live"
    if "complet" in raw or "final" in raw:
        return "completed"
    if "upcom" in raw or "tbd" in raw or "soon" in raw or "sched" in raw:
        return "upcoming"
    # No usable status string: infer from the data. A decided winner means the
    # series is over; otherwise assume it hasn't been played yet.
    teams = d.get("teams") or []
    if any((t or {}).get("is_winner") for t in teams):
        return "completed"
    return "upcoming"


@dataclass
class EventCard:
    id: int
    name: str
    status: str  # "upcoming" | "ongoing" | "completed"
    region: str | None = None
    prize: str | None = None
    start: str | None = None
    end: str | None = None

    @classmethod
    def from_raw(cls, d: dict) -> "EventCard":
        return cls(
            id=_i(d.get("id")) or 0,
            name=d.get("name") or "",
            status=d.get("status") or "ongoing",
            region=d.get("region"),
            prize=d.get("prize"),
            start=d.get("start_text") or d.get("start_date"),
            end=d.get("end_text") or d.get("end_date"),
        )


@dataclass
class PlayerLine:
    name: str
    agents: list[str] = field(default_factory=list)
    acs: int | None = None
    k: int | None = None
    d: int | None = None
    a: int | None = None
    adr: float | None = None
    hs_pct: float | None = None
    fk: int | None = None
    fd: int | None = None
    team_short: str | None = None

    @classmethod
    def from_raw(cls, d: dict) -> "PlayerLine":
        return cls(
            name=d.get("name") or "?",
            agents=list(d.get("agents") or []),
            acs=_i(d.get("acs")),
            k=_i(d.get("k")),
            d=_i(d.get("d")),
            a=_i(d.get("a")),
            adr=d.get("adr"),
            hs_pct=d.get("hs_pct"),
            fk=_i(d.get("fk")),
            fd=_i(d.get("fd")),
            team_short=d.get("team_short"),
        )


@dataclass
class RoundLine:
    number: int
    side: str | None = None  # "Attacker" | "Defender"
    winner_short: str | None = None

    @property
    def is_attack(self) -> bool:
        return (self.side or "").lower().startswith("attack")

    @classmethod
    def from_raw(cls, d: dict) -> "RoundLine":
        return cls(
            number=_i(d.get("number")) or 0,
            side=d.get("winner_side"),
            winner_short=d.get("winner_team_short"),
        )


@dataclass
class MapScore:
    name: str
    players: list[PlayerLine] = field(default_factory=list)
    team1_short: str | None = None
    team1_score: int | None = None
    team2_short: str | None = None
    team2_score: int | None = None
    rounds: list[RoundLine] = field(default_factory=list)

    @property
    def is_aggregate(self) -> bool:
        return self.name.lower() == "all"

    @property
    def has_score(self) -> bool:
        # Upcoming maps come back as 0–0 with placeholder rosters; treat only a
        # non-zero score as "real".
        return (self.team1_score or 0) + (self.team2_score or 0) > 0

    @property
    def state(self) -> str:
        """One of 'completed' | 'live' | 'pending' for rendering decisions."""
        if self.rounds:
            return "completed"
        if self.has_score:
            return "live"
        return "pending"

    @classmethod
    def from_raw(cls, d: dict) -> "MapScore":
        teams = d.get("teams") or []
        t1 = teams[0] if len(teams) > 0 and teams[0] else {}
        t2 = teams[1] if len(teams) > 1 and teams[1] else {}
        return cls(
            name=d.get("map_name") or "?",
            players=[PlayerLine.from_raw(p) for p in (d.get("players") or [])],
            team1_short=t1.get("short") or t1.get("name"),
            team1_score=_i(t1.get("score")),
            team2_short=t2.get("short") or t2.get("name"),
            team2_score=_i(t2.get("score")),
            rounds=[RoundLine.from_raw(r) for r in (d.get("rounds") or [])],
        )


@dataclass
class Veto:
    action: str  # pick | ban | remaining
    team: str
    map: str


@dataclass
class SeriesDetail:
    match_id: int
    team1: TeamSide
    team2: TeamSide
    event: str
    phase: str
    best_of: str | None = None
    status_note: str | None = None
    remaining: str | None = None
    patch: str | None = None
    vetoes: list[Veto] = field(default_factory=list)
    maps: list[MapScore] = field(default_factory=list)

    @property
    def is_live(self) -> bool:
        return "live" in (self.status_note or "").lower()

    @property
    def is_completed(self) -> bool:
        if self.is_live:
            return False
        return (self.team1.score or 0) > 0 or (self.team2.score or 0) > 0

    def pick_label(self, map_name: str) -> str | None:
        """Which team picked a given map (or 'decider'), from the veto data."""
        for v in self.vetoes:
            if v.map.lower() != map_name.lower():
                continue
            if v.action.lower() == "pick":
                return f"{v.team} pick"
            if v.action.lower() in ("remaining", "decider"):
                return "decider"
        return None

    @classmethod
    def from_raw(cls, info: dict, maps: list[dict] | None) -> "SeriesDetail":
        teams = info.get("teams") or []
        t1 = TeamSide.from_raw(teams[0] if len(teams) > 0 else {})
        t2 = TeamSide.from_raw(teams[1] if len(teams) > 1 else {})
        score = info.get("score") or [None, None]
        t1.score = _i(score[0]) if len(score) > 0 else None
        t2.score = _i(score[1]) if len(score) > 1 else None

        vetoes: list[Veto] = []
        for v in (info.get("map_actions") or []):
            vetoes.append(Veto(v.get("action", ""), v.get("team", ""), v.get("map", "")))

        return cls(
            match_id=_i(info.get("match_id")) or 0,
            team1=t1,
            team2=t2,
            event=info.get("event") or "",
            phase=info.get("event_phase") or "",
            best_of=info.get("best_of"),
            status_note=info.get("status_note"),
            remaining=info.get("remaining"),
            patch=info.get("patch"),
            vetoes=vetoes,
            maps=[MapScore.from_raw(m) for m in (maps or [])],
        )
