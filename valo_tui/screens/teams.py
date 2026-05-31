"""[m] teams — the rosters competing in the focused event, with their running
series record. Derived from the event's own matches."""

from __future__ import annotations

from rich.text import Text

from textual.app import ComposeResult
from textual.containers import Vertical
from textual.widgets import Label

from ..data import cache
from ..data.standings import team_records
from .widgets import ACCENT, MUTED, TEXT, VimDataTable


class TeamsView(Vertical):
    def compose(self) -> ComposeResult:
        yield Label("teams", classes="page-title")
        yield Label("rosters in this event", classes="hint")
        yield VimDataTable(id="teams-table", cursor_type="row", zebra_stripes=False)

    def on_mount(self) -> None:
        t = self.query_one(VimDataTable)
        t.add_column("team", width=24)
        t.add_column("W-L", width=8)
        t.add_column("maps", width=8)
        t.add_column("diff", width=6)
        self.load_data()

    def load_data(self) -> None:
        t = self.query_one(VimDataTable)
        t.clear()
        eid = getattr(self.app, "current_event_id", None)
        if eid is None:
            t.add_row(Text("select an event first", style=MUTED), Text(""), Text(""), Text(""))
            return

        matches = cache.event_match_cards(eid, self.app.event_name)
        # Every team that appears, even if it hasn't played a decided match yet.
        names: list[str] = []
        for m in matches:
            for side in (m.team1.name, m.team2.name):
                if side and side != "TBD" and side not in names:
                    names.append(side)

        records = {r.team: r for r in team_records(matches)}
        if not names:
            t.add_row(Text("no teams cached for this event", style=MUTED),
                      Text(""), Text(""), Text(""))
            return

        # Teams with a record first (by standing), then the rest alphabetically.
        ranked = [r.team for r in team_records(matches)]
        rest = sorted(n for n in names if n not in records)
        for name in ranked + rest:
            r = records.get(name)
            if r:
                wl = f"{r.wins}-{r.losses}"
                maps = f"{r.maps_won}-{r.maps_lost}"
                diff = f"{r.map_diff:+d}"
                style = ACCENT if r.map_diff > 0 else MUTED
            else:
                wl, maps, diff, style = "0-0", "0-0", "—", MUTED
            t.add_row(
                Text(name, style=TEXT),
                Text(wl, style=TEXT),
                Text(maps, style=MUTED),
                Text(diff, style=style),
            )
