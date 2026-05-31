"""[a] about — what valo-tui is, where the data comes from, and the keys."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import VerticalScroll
from textual.widgets import Label, Static

from .. import __version__
from .widgets import ACCENT, MUTED, TEXT


class AboutView(VerticalScroll):
    can_focus = True

    def compose(self) -> ComposeResult:
        yield Label("about", classes="page-title")
        body = (
            f"[bold {TEXT}]valo-tui[/]  [{MUTED}]v{__version__}[/]\n"
            f"[{TEXT}]A terminal-native tracker for global Valorant esports.[/]\n\n"
            f"[{ACCENT}]what you can do[/]\n"
            f"[{MUTED}]· browse events by region and stage[/]\n"
            f"[{MUTED}]· open an event for its overview, results, fixtures,[/]\n"
            f"[{MUTED}]  standings, bracket and teams[/]\n"
            f"[{MUTED}]· follow live matches with map scores across regions[/]\n"
            f"[{MUTED}]· drill into a match for veto, maps, agents, ACS,[/]\n"
            f"[{MUTED}]  K/D/A, ADR, HS%, FK and FD[/]\n\n"
            f"[{ACCENT}]data[/]\n"
            f"[{MUTED}]· source   [/][{TEXT}]vlr.gg (via vlrdevapi)[/]\n"
            f"[{MUTED}]· cache    [/][{TEXT}]SQLite, written by a background worker[/]\n"
            f"[{MUTED}]· the UI never blocks on the network[/]\n\n"
            f"[{ACCENT}]global nav[/]\n"
            f"[{MUTED}]· h [/][{TEXT}]home      [/][{MUTED}]landing dashboard[/]\n"
            f"[{MUTED}]· e [/][{TEXT}]events    [/][{MUTED}]pick a tournament to focus on[/]\n"
            f"[{MUTED}]· l [/][{TEXT}]live      [/][{MUTED}]global Bento dashboard[/]\n"
            f"[{MUTED}]· a [/][{TEXT}]about     [/][{MUTED}]this page[/]\n\n"
            f"[{ACCENT}]inside an event[/]\n"
            f"[{MUTED}]· o [/][{TEXT}]overview  [/][{MUTED}]· [/][{TEXT}]r [/][{MUTED}]results  [/]"
            f"[{TEXT}]· f [/][{MUTED}]fixtures[/]\n"
            f"[{MUTED}]· t [/][{TEXT}]standings [/][{MUTED}]· [/][{TEXT}]b [/][{MUTED}]bracket  [/]"
            f"[{TEXT}]· m [/][{MUTED}]teams[/]\n\n"
            f"[{ACCENT}]keys[/]\n"
            f"[{MUTED}]· ↑↓ / j k move · enter open · esc back to nav[/]\n"
            f"[{MUTED}]· ctrl+r refresh · q quit[/]\n"
        )
        yield Static(body, id="about-body")

    def load_data(self) -> None:  # for routing parity
        pass
