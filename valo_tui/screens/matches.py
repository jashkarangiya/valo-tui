"""[m] matches — a sortable list of matches (Live / Soon / Done).

Pressing Enter on a row drills into the per-map scoreboards."""

from __future__ import annotations

from rich.text import Text

from textual.app import ComposeResult
from textual.containers import Vertical
from textual.widgets import Label

from ..data import cache
from ..data.models import MatchCard
from .widgets import LIVE, MUTED, TEXT, VimDataTable

# Status sort order: live first, then upcoming, then completed.
_ORDER = {"live": 0, "upcoming": 1, "completed": 2}


class MatchesView(Vertical):
    """The [m] matches screen, backed by a DataTable keyed on match_id."""

    def compose(self) -> ComposeResult:
        yield Label("all matches", classes="page-title")
        yield Label("j/k move · enter → scoreboards · r refresh", classes="hint")
        table = VimDataTable(id="matches-table", cursor_type="row", zebra_stripes=False)
        yield table

    def on_mount(self) -> None:
        table = self.query_one(VimDataTable)
        table.add_column("status", width=7, key="status")
        table.add_column("match", width=34, key="match")
        table.add_column("score", width=7, key="score")
        table.add_column("event", key="event")
        table.add_column("when", width=10, key="when")
        self.load_data()

    def load_data(self) -> None:
        table = self.query_one(VimDataTable)
        prev = table.cursor_row
        table.clear()

        matches = sorted(
            cache.live_matches() + cache.upcoming_matches() + cache.completed_matches(),
            key=lambda m: _ORDER.get(m.status, 9),
        )
        for m in matches:
            table.add_row(*self._row(m), key=str(m.match_id))

        if matches:
            table.move_cursor(row=min(prev, len(matches) - 1))

    def _row(self, m: MatchCard) -> tuple[Text, ...]:
        if m.is_live:
            status = Text("● live", style=LIVE)
            color = TEXT
        elif m.status == "completed":
            status = Text("✓ done", style=MUTED)
            color = MUTED
        else:
            status = Text("○ soon", style=MUTED)
            color = TEXT

        match = Text(f"{m.team1.name}  vs  {m.team2.name}", style=color)
        s1 = m.team1.score if m.team1.score is not None else "–"
        s2 = m.team2.score if m.team2.score is not None else "–"
        score = Text(f"{s1}–{s2}", style=LIVE if m.is_live else MUTED)
        event = Text(m.event, style=MUTED)
        when = Text(m.time or m.date or "", style=MUTED)
        return status, match, score, event, when
