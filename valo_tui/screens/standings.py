"""[t] standings — the focused event's stage table.

Event-scoped: a W-L / map-record / differential matrix derived from the
event's completed matches (real per-event standings via vlr.events.standings
can replace this later)."""

from __future__ import annotations

from rich.text import Text

from textual.app import ComposeResult
from textual.containers import Vertical
from textual.widgets import Label

from ..data import cache
from ..data.standings import team_records
from .widgets import ACCENT, MUTED, TEXT, VimDataTable


class StandingsView(Vertical):
    def compose(self) -> ComposeResult:
        yield Label("standings", classes="page-title")
        yield Label("derived from this event's completed matches", classes="hint")
        yield VimDataTable(id="standings-table", cursor_type="row", zebra_stripes=False)

    def on_mount(self) -> None:
        t = self.query_one(VimDataTable)
        t.add_column("#", width=4)
        t.add_column("team", width=22)
        t.add_column("W-L", width=7)
        t.add_column("maps", width=8)
        t.add_column("diff", width=6)
        self.load_data()

    def load_data(self) -> None:
        t = self.query_one(VimDataTable)
        t.clear()
        eid = getattr(self.app, "current_event_id", None)
        if eid is None:
            t.add_row(Text(""), Text("select an event first", style=MUTED),
                      Text(""), Text(""), Text(""))
            return

        rows = team_records(cache.event_match_cards(eid, self.app.event_name))
        for i, r in enumerate(rows[:24], start=1):
            t.add_row(
                Text(str(i), style=MUTED),
                Text(r.team, style=TEXT),
                Text(f"{r.wins}-{r.losses}", style=TEXT),
                Text(f"{r.maps_won}-{r.maps_lost}", style=MUTED),
                Text(f"{r.map_diff:+d}", style=ACCENT if r.map_diff > 0 else MUTED),
            )
        if not rows:
            t.add_row(Text(""), Text("no completed matches for this event yet", style=MUTED),
                      Text(""), Text(""), Text(""))
