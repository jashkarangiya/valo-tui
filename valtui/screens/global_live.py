"""Global-live dashboard — a Bento grid of the four regional leagues with
international events pinned across the top. Shows Live → Next → Recent."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import Grid, Vertical
from textual.widgets import Static

from .. import config
from ..data import cache
from .widgets import LIVE, MUTED, TEXT, match_line

MAX_PER_SLOT = 4


class RegionPanel(Static):
    """One league's compartment in the Bento grid."""

    def __init__(self, league: str) -> None:
        super().__init__(id=f"region-{league.lower()}")
        self.league = league

    def render_data(self, slots: dict) -> None:
        live, nxt, recent = slots["live"], slots["next"], slots["recent"]
        self.set_class(bool(live), "live")

        lines = [f"[bold {TEXT}]{self.league}[/]"]
        if live:
            lines.append(f"[{LIVE}]── live ──[/]")
            lines += [match_line(m) for m in live[:MAX_PER_SLOT]]
        if nxt:
            lines.append(f"[{MUTED}]── next ──[/]")
            lines += [match_line(m) for m in nxt[:MAX_PER_SLOT]]
        if recent:
            lines.append(f"[{MUTED}]── recent ──[/]")
            lines += [match_line(m) for m in recent[:MAX_PER_SLOT]]
        if not (live or nxt or recent):
            lines.append(f"[{MUTED}]no matches tracked[/]")
        self.update("\n".join(lines))


class GlobalLiveView(Vertical):
    """The [g] global live screen."""

    def compose(self) -> ComposeResult:
        yield Static(id="intl-bar")
        with Grid(id="regions-grid"):
            for region in config.REGIONS:
                yield RegionPanel(region)

    def on_mount(self) -> None:
        self.load_data()

    def load_data(self) -> None:
        regions, intl = cache.global_live()
        self._render_intl(intl)
        for region in config.REGIONS:
            panel = self.query_one(f"#region-{region.lower()}", RegionPanel)
            panel.render_data(regions[region])

    def _render_intl(self, intl: list) -> None:
        bar = self.query_one("#intl-bar", Static)
        if not intl:
            bar.update(f"[{MUTED}]★ international · no active international events[/]")
            return
        lines = [f"[bold {LIVE}]★ international[/]"]
        lines += [match_line(m) for m in intl[:5]]
        bar.update("\n".join(lines))
