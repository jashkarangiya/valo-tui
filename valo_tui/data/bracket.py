"""Reconstruct a double-elimination bracket from a flat list of event matches.

vlr.gg exposes no explicit bracket structure, only matches tagged with a
``phase`` like "Upper Quarterfinals" or "Lower Final". We bucket those into
upper / lower / grand-final sections, order the rounds within each section,
and recover the tree edges by team identity (a match's winner reappears as a
participant in the next round).
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field

# Phase keyword → rank within a section (lower renders further left).
_ROUND_RANK = (
    ("quarter", 1),
    ("semi", 2),
    ("round 1", 1),
    ("round 2", 2),
    ("round 3", 3),
    ("round 4", 4),
    ("round 5", 5),
    ("final", 9),  # "Upper Final" / "Lower Final"
)


def _round_rank(phase: str) -> int:
    p = phase.lower()
    m = re.search(r"round\s*(\d+)", p)
    if m:
        return int(m.group(1))
    for kw, rank in _ROUND_RANK:
        if kw in p:
            return rank
    return 0


def _section(phase: str) -> str | None:
    """'upper' | 'lower' | 'final', or None if not a bracket phase."""
    p = phase.lower()
    if "grand final" in p:
        return "final"
    if "upper" in p:
        return "upper"
    if "lower" in p:
        return "lower"
    return None


@dataclass
class BracketSlot:
    name: str
    score: int | None = None
    winner: bool = False


@dataclass
class BracketMatch:
    match_id: int
    top: BracketSlot
    bottom: BracketSlot
    status: str = ""

    @property
    def winner_name(self) -> str | None:
        if self.top.winner:
            return self.top.name
        if self.bottom.winner:
            return self.bottom.name
        return None


@dataclass
class BracketColumn:
    title: str
    matches: list[BracketMatch] = field(default_factory=list)


@dataclass
class BracketSection:
    name: str  # "Upper" | "Lower" | "Final"
    columns: list[BracketColumn] = field(default_factory=list)


@dataclass
class Bracket:
    sections: list[BracketSection] = field(default_factory=list)

    @property
    def has_data(self) -> bool:
        return any(c.matches for s in self.sections for c in s.columns)


def _slot(team: dict) -> BracketSlot:
    return BracketSlot(
        name=team.get("name") or "TBD",
        score=team.get("score"),
        winner=bool(team.get("is_winner")),
    )


def _to_match(raw: dict) -> BracketMatch | None:
    teams = raw.get("teams") or []
    if len(teams) < 2:
        return None
    return BracketMatch(
        match_id=int(raw.get("match_id") or 0),
        top=_slot(teams[0] or {}),
        bottom=_slot(teams[1] or {}),
        status=raw.get("status") or "",
    )


_TITLES = {"upper": "Upper Bracket", "lower": "Lower Bracket", "final": "Grand Final"}
_ORDER = ("upper", "lower", "final")


def build_bracket(raw_matches: list[dict]) -> Bracket:
    """Group/order bracket matches into sections of left-to-right columns."""
    # section -> rank -> (phase, [matches])
    buckets: dict[str, dict[int, list[tuple[str, dict]]]] = {}
    for raw in raw_matches:
        phase = raw.get("phase") or ""
        section = _section(phase)
        if section is None:
            continue
        rank = _round_rank(phase)
        buckets.setdefault(section, {}).setdefault(rank, []).append((phase, raw))

    bracket = Bracket()
    for section in _ORDER:
        ranks = buckets.get(section)
        if not ranks:
            continue
        sec = BracketSection(name=_TITLES[section])
        for rank in sorted(ranks):
            entries = ranks[rank]
            title = entries[0][0]  # phase name of this column
            col = BracketColumn(title=title)
            for _, raw in entries:
                m = _to_match(raw)
                if m is not None:
                    col.matches.append(m)
            if col.matches:
                sec.columns.append(col)
        if sec.columns:
            bracket.sections.append(sec)
    return bracket
