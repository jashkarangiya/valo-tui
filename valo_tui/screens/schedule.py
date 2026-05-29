"""[s] schedule — upcoming matches as bento cards."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import VerticalScroll
from textual.widgets import Label, Static

from ..data import cache
from .widgets import ACCENT, MUTED, TEXT


class ScheduleView(VerticalScroll):
    def compose(self) -> ComposeResult:
        yield Label("upcoming matches", classes="page-title")
        yield VerticalScroll(id="schedule-cards")

    def on_mount(self) -> None:
        self.load_data()

    def load_data(self) -> None:
        cards = self.query_one("#schedule-cards", VerticalScroll)
        cards.remove_children()
        matches = cache.upcoming_matches()
        if not matches:
            cards.mount(Label("nothing scheduled in the cache", classes="hint"))
            return
        for m in matches[:24]:
            when = m.time or m.date or "soon"
            title = f"[bold {TEXT}]{m.team1.name}  vs  {m.team2.name}[/]   [{ACCENT}]·[/]  [{TEXT}]{when}[/]"
            sub = f"[{MUTED}]{m.event} · {m.phase}[/]" if m.phase else f"[{MUTED}]{m.event}[/]"
            cards.mount(Static(f"{title}\n{sub}", classes="card"))
