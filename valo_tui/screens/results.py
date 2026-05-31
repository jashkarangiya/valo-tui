"""[r] results — completed and live series for the focused event.

Event-scoped (never global): the rows come from the event's own match list.
Pressing Enter on a row drills into the per-map scoreboards."""

from __future__ import annotations

from rich.text import Text

from textual.app import ComposeResult
from textual.containers import Vertical
from textual.widgets import Label

from ..data import cache
from ..data.models import MatchCard
from .widgets import LIVE, MUTED, TEXT, VimDataTable

# Live first, then most-recently completed.
_ORDER = {"live": 0, "completed": 1}


class ResultsView(Vertical):
    def compose(self) -> ComposeResult:
        yield Label("results", classes="page-title")
        yield Label("j/k move · enter → scoreboards", classes="hint")
        yield VimDataTable(id="results-table", cursor_type="row", zebra_stripes=False)

    def on_mount(self) -> None:
        t = self.query_one(VimDataTable)
        t.add_column("status", width=7, key="status")
        t.add_column("match", width=34, key="match")
        t.add_column("score", width=7, key="score")
        t.add_column("phase", width=20, key="phase")
        t.add_column("when", key="when")
        self.load_data()

    def load_data(self) -> None:
        t = self.query_one(VimDataTable)
        prev = t.cursor_row
        t.clear()
        eid = getattr(self.app, "current_event_id", None)
        if eid is None:
            t.add_row(Text("select an event first", style=MUTED),
                      Text(""), Text(""), Text(""), Text(""))
            return

        matches = [
            m for m in cache.event_match_cards(eid, self.app.event_name)
            if m.status in _ORDER
        ]
        matches.sort(key=lambda m: _ORDER.get(m.status, 9))
        for m in matches:
            t.add_row(*self._row(m), key=str(m.match_id))
        if matches:
            t.move_cursor(row=min(prev, len(matches) - 1))
        else:
            t.add_row(Text("no completed matches yet", style=MUTED),
                      Text(""), Text(""), Text(""), Text(""))

    def _row(self, m: MatchCard) -> tuple[Text, ...]:
        if m.is_live:
            status = Text("● live", style=LIVE)
        else:
            status = Text("✓ done", style=MUTED)
        match = Text(f"{m.team1.name}  vs  {m.team2.name}", style=TEXT)
        s1 = m.team1.score if m.team1.score is not None else "–"
        s2 = m.team2.score if m.team2.score is not None else "–"
        score = Text(f"{s1}–{s2}", style=LIVE if m.is_live else MUTED)
        phase = Text(m.phase, style=MUTED)
        when = Text(m.time or m.date or "", style=MUTED)
        return status, match, score, phase, when
