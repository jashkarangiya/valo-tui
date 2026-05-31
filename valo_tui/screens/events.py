"""[e] events — the parent hub. Browse VCT / Challenger stages and pick one to
focus the whole app on it. Enter on a row opens that event's overview."""

from __future__ import annotations

from rich.text import Text

from textual.app import ComposeResult
from textual.containers import Vertical
from textual.widgets import Label

from ..data import cache
from ..data.models import EventCard
from .widgets import ACCENT, LIVE, MUTED, TEXT, VimDataTable

# Ongoing first, then upcoming, then completed.
_ORDER = {"ongoing": 0, "upcoming": 1, "completed": 2}


def _status_cell(status: str) -> Text:
    s = (status or "").lower()
    if s.startswith("ongo"):
        return Text("● live", style=LIVE)
    if s.startswith("upcom"):
        return Text("○ soon", style=MUTED)
    return Text("✓ done", style=MUTED)


class EventsView(Vertical):
    def compose(self) -> ComposeResult:
        yield Label("events", classes="page-title")
        yield Label("enter → open event · j/k move", classes="hint")
        yield VimDataTable(id="events-table", cursor_type="row", zebra_stripes=False)

    def on_mount(self) -> None:
        t = self.query_one(VimDataTable)
        t.add_column("status", width=8, key="status")
        t.add_column("event", width=40, key="event")
        t.add_column("region", width=12, key="region")
        t.add_column("when", key="when")
        self.load_data()

    def load_data(self) -> None:
        t = self.query_one(VimDataTable)
        prev = t.cursor_row
        t.clear()
        events = sorted(
            cache.active_events(),
            key=lambda e: _ORDER.get((e.status or "").lower(), 9),
        )
        for e in events:
            t.add_row(*self._row(e), key=str(e.id))
        if events:
            t.move_cursor(row=min(prev, len(events) - 1))
        else:
            t.add_row(Text(""), Text("no events in cache — start the worker", style=MUTED),
                      Text(""), Text(""))

    def _row(self, e: EventCard) -> tuple[Text, ...]:
        when = ""
        if e.start and e.end:
            when = f"{e.start} – {e.end}"
        elif e.start:
            when = e.start
        return (
            _status_cell(e.status),
            Text(e.name, style=TEXT),
            Text(e.region or "—", style=ACCENT if e.region else MUTED),
            Text(when, style=MUTED),
        )
