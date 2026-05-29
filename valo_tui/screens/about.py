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
            f"[{TEXT}]A read-only terminal UI for tracking Valorant esports.[/]\n\n"
            f"[{ACCENT}]data[/]\n"
            f"[{MUTED}]· source   [/][{TEXT}]vlr.gg (via vlrdevapi)[/]\n"
            f"[{MUTED}]· cache    [/][{TEXT}]SQLite, written by a background worker[/]\n"
            f"[{MUTED}]· the UI never blocks on the network[/]\n\n"
            f"[{ACCENT}]navigation[/]\n"
            f"[{MUTED}]· g [/][{TEXT}]live      [/][{MUTED}]global Bento dashboard[/]\n"
            f"[{MUTED}]· m [/][{TEXT}]matches   [/][{MUTED}]all matches, enter to drill in[/]\n"
            f"[{MUTED}]· t [/][{TEXT}]standings [/][{MUTED}]points table[/]\n"
            f"[{MUTED}]· s [/][{TEXT}]schedule  [/][{MUTED}]upcoming matches[/]\n"
            f"[{MUTED}]· b [/][{TEXT}]brackets  [/][{MUTED}]playoff trees[/]\n"
            f"[{MUTED}]· j/k move · enter open · esc/q back[/]\n"
        )
        yield Static(body, id="about-body")

    def load_data(self) -> None:  # for routing parity
        pass
