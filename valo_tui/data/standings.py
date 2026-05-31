"""Derive a standings table from a list of completed series.

vlr.gg has no standings feed wired up yet, so we reconstruct match/map records
from the event's own matches. A match's per-team ``score`` is the series score
(maps won), so we get both a W-L series record and a map differential out of
the same data. Shared by the standings, teams and overview sub-pages.
"""

from __future__ import annotations

from dataclasses import dataclass

from .models import MatchCard


@dataclass
class TeamRecord:
    team: str
    played: int = 0
    wins: int = 0
    losses: int = 0
    maps_won: int = 0
    maps_lost: int = 0

    @property
    def map_diff(self) -> int:
        return self.maps_won - self.maps_lost

    @property
    def pct(self) -> float:
        return (self.wins / self.played * 100) if self.played else 0.0


def team_records(matches: list[MatchCard]) -> list[TeamRecord]:
    """Build a sorted standings table from completed matches in ``matches``."""
    table: dict[str, TeamRecord] = {}
    for m in matches:
        if m.status != "completed":
            continue
        s1, s2 = m.team1.score, m.team2.score
        if s1 is None or s2 is None or s1 == s2:
            continue
        if m.team1.name in ("TBD", "") or m.team2.name in ("TBD", ""):
            continue
        r1 = table.setdefault(m.team1.name, TeamRecord(m.team1.name))
        r2 = table.setdefault(m.team2.name, TeamRecord(m.team2.name))
        r1.played += 1
        r2.played += 1
        r1.maps_won += s1
        r1.maps_lost += s2
        r2.maps_won += s2
        r2.maps_lost += s1
        if s1 > s2:
            r1.wins += 1
            r2.losses += 1
        else:
            r2.wins += 1
            r1.losses += 1
    return sorted(
        table.values(),
        key=lambda r: (r.wins, r.map_diff, r.pct),
        reverse=True,
    )
