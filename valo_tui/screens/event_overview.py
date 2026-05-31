"""[o] overview — the event landing page.

The parent of all the event sub-pages: it tells you what the event *is*
(region, format, status, dates) and where things stand (phases seen, how many
matches are done / live / upcoming) before you drill into the children.
"""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import VerticalScroll
from textual.widgets import Label, Static

from ..data import cache
from ..data.models import MatchCard
from .widgets import ACCENT, LIVE, MUTED, TEXT


class EventOverviewView(VerticalScroll):
    can_focus = True

    def compose(self) -> ComposeResult:
        yield Label("overview", classes="page-title")
        yield Static(id="ov-banner", classes="card")
        yield Static(id="ov-progress")
        yield Static(id="ov-nav")

    def load_data(self) -> None:
        eid = getattr(self.app, "current_event_id", None)
        if eid is None:
            self.query_one("#ov-banner", Static).update(
                f"[{MUTED}]select an event from [/][{TEXT}]e events[/]"
            )
            self.query_one("#ov-progress", Static).update("")
            self.query_one("#ov-nav", Static).update("")
            return

        event = cache.event_by_id(eid)
        name = event.name if event else "event"
        matches = cache.event_match_cards(eid, name)
        self._render_banner(event, name)
        self._render_progress(matches)
        self._render_nav()

    def _render_banner(self, event, name: str) -> None:
        region = (event.region if event else None) or "—"
        status = (event.status if event else None) or "—"
        dates = ""
        if event and event.start and event.end:
            dates = f"{event.start} – {event.end}"
        elif event and event.start:
            dates = event.start
        prize = (event.prize if event else None) or ""

        lines = [f"[bold {TEXT}]{name}[/]"]
        meta = f"[{MUTED}]region [/][{TEXT}]{region}[/]   [{MUTED}]status [/]"
        meta += f"[{LIVE}]{status}[/]" if status.lower().startswith("ongo") else f"[{TEXT}]{status}[/]"
        lines.append(meta)
        if dates:
            lines.append(f"[{MUTED}]dates  [/][{TEXT}]{dates}[/]")
        if prize:
            lines.append(f"[{MUTED}]prize  [/][{TEXT}]{prize}[/]")
        self.query_one("#ov-banner", Static).update("\n".join(lines))

    def _render_progress(self, matches: list[MatchCard]) -> None:
        live = sum(1 for m in matches if m.status == "live")
        done = sum(1 for m in matches if m.status == "completed")
        soon = sum(1 for m in matches if m.status == "upcoming")

        # Distinct phases, in the order they first appear, with their completion.
        phases: list[str] = []
        for m in matches:
            if m.phase and m.phase not in phases:
                phases.append(m.phase)

        lines = [
            f"[bold {ACCENT}]progress[/]",
            f"[{MUTED}]matches  [/][{TEXT}]{done} done[/][{MUTED}] · [/]"
            f"[{LIVE}]{live} live[/][{MUTED}] · [/][{TEXT}]{soon} upcoming[/]",
        ]
        if phases:
            lines.append("")
            for ph in phases[:8]:
                in_phase = [m for m in matches if m.phase == ph]
                left = sum(1 for m in in_phase if m.status != "completed")
                if any(m.status == "live" for m in in_phase):
                    state = f"[{LIVE}]live[/]"
                elif left == 0:
                    state = f"[{MUTED}]complete[/]"
                else:
                    state = f"[{TEXT}]{left} left[/]"
                lines.append(f"[{TEXT}]{ph:<22}[/] {state}")
        else:
            lines.append(f"[{MUTED}]no matches cached for this event yet[/]")
        self.query_one("#ov-progress", Static).update("\n".join(lines))

    def _render_nav(self) -> None:
        nav = (
            f"\n[bold {ACCENT}]quick nav[/]\n"
            f"[{MUTED}]· [/][{TEXT}]r[/][{MUTED}]  results    completed & live series[/]\n"
            f"[{MUTED}]· [/][{TEXT}]f[/][{MUTED}]  fixtures   what's still to come[/]\n"
            f"[{MUTED}]· [/][{TEXT}]t[/][{MUTED}]  standings  group tables[/]\n"
            f"[{MUTED}]· [/][{TEXT}]b[/][{MUTED}]  bracket    playoff tree[/]\n"
            f"[{MUTED}]· [/][{TEXT}]m[/][{MUTED}]  teams      rosters in this event[/]"
        )
        self.query_one("#ov-nav", Static).update(nav)
