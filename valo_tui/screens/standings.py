"""[t] standings — a points table.

Standings aren't fetched yet, so for now we derive a simple win/loss table
from the completed matches already in the cache (UI-first; real per-event
standings via vlr.events.standings can replace this later)."""

from __future__ import annotations

from dataclasses import dataclass

from rich.text import Text

from textual.app import ComposeResult
from textual.containers import Vertical
from textual.widgets import Label

from ..data import cache
from .widgets import ACCENT, MUTED, TEXT, VimDataTable


@dataclass
class _Row:
    team: str
    played: int = 0
    wins: int = 0
    losses: int = 0

    @property
    def pct(self) -> float:
        return (self.wins / self.played * 100) if self.played else 0.0


class StandingsView(Vertical):
    def compose(self) -> ComposeResult:
        yield Label("standings", classes="page-title")
        yield Label("derived from completed matches in cache", classes="hint")
        yield VimDataTable(id="standings-table", cursor_type="row", zebra_stripes=False)

    def on_mount(self) -> None:
        t = self.query_one(VimDataTable)
        t.add_column("#", width=4)
        t.add_column("team", width=20)
        t.add_column("P", width=4)
        t.add_column("W", width=4)
        t.add_column("L", width=4)
        t.add_column("win%", width=6)
        self.load_data()

    def load_data(self) -> None:
        t = self.query_one(VimDataTable)
        t.clear()
        table: dict[str, _Row] = {}
        for m in cache.completed_matches():
            s1 = m.team1.score if m.team1.score is not None else -1
            s2 = m.team2.score if m.team2.score is not None else -1
            if s1 == s2:
                continue
            t1 = table.setdefault(m.team1.name, _Row(m.team1.name))
            t2 = table.setdefault(m.team2.name, _Row(m.team2.name))
            t1.played += 1
            t2.played += 1
            if s1 > s2:
                t1.wins += 1
                t2.losses += 1
            else:
                t2.wins += 1
                t1.losses += 1

        rows = sorted(table.values(), key=lambda r: (r.wins, r.pct), reverse=True)
        for i, r in enumerate(rows[:16], start=1):
            t.add_row(
                Text(str(i), style=MUTED),
                Text(r.team, style=TEXT),
                Text(str(r.played), style=MUTED),
                Text(str(r.wins), style=TEXT),
                Text(str(r.losses), style=MUTED),
                Text(f"{r.pct:.0f}%", style=ACCENT),
            )
        if not rows:
            t.add_row(Text(""), Text("no completed matches in cache", style=MUTED),
                      Text(""), Text(""), Text(""), Text(""))
