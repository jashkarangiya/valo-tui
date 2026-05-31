"""[f] fixtures — what's still to come in the focused event.

Event-scoped upcoming matches as bento cards, grouped by phase so the bracket
rounds read in order."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import VerticalScroll
from textual.widgets import Label, Static

from ..data import cache
from .widgets import ACCENT, MUTED, TEXT


class FixturesView(VerticalScroll):
    can_focus = True

    def compose(self) -> ComposeResult:
        yield Label("fixtures", classes="page-title")
        yield VerticalScroll(id="fixtures-cards")

    def load_data(self) -> None:
        cards = self.query_one("#fixtures-cards", VerticalScroll)
        cards.remove_children()
        eid = getattr(self.app, "current_event_id", None)
        if eid is None:
            cards.mount(Label("select an event first", classes="hint"))
            return

        matches = [
            m for m in cache.event_match_cards(eid, self.app.event_name)
            if m.status == "upcoming"
        ]
        if not matches:
            cards.mount(Label("nothing scheduled for this event", classes="hint"))
            return

        # Group by phase, preserving first-seen order.
        phases: list[str] = []
        for m in matches:
            ph = m.phase or "scheduled"
            if ph not in phases:
                phases.append(ph)
        for ph in phases:
            cards.mount(Label(ph, classes="sched-section"))
            for m in [x for x in matches if (x.phase or "scheduled") == ph]:
                cards.mount(self._card(m))

    def _card(self, m) -> Static:
        when = m.time or m.date or "soon"
        title = (
            f"[bold {TEXT}]{m.team1.name}  vs  {m.team2.name}[/]"
            f"   [{ACCENT}]·[/]  [{TEXT}]{when}[/]"
        )
        return Static(title, classes="card")
