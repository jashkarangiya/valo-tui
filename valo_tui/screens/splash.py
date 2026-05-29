"""Landing page — ASCII logo, version, cache freshness. Press enter to enter
the app (or it auto-advances after a few seconds)."""

from __future__ import annotations

import pyfiglet

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Center, Middle
from textual.screen import Screen
from textual.widgets import Static

from .. import __version__
from ..data import cache

ART = pyfiglet.figlet_format("valo-tui", font="slant")


class SplashScreen(Screen):
    BINDINGS = [
        Binding("enter,space", "enter_app", "Enter", show=False),
        Binding("q", "app.quit", "Quit", show=False),
    ]

    CSS = """
    SplashScreen { background: #0a1822; align: center middle; }
    #logo { color: #e8674e; text-style: bold; width: auto; }
    #tag  { color: #4a708b; width: auto; margin-top: 1; }
    #freshness { color: #4a708b; width: auto; margin-top: 1; }
    #enter-hint { color: #c8d8e8; width: auto; margin-top: 2; }
    """

    def compose(self) -> ComposeResult:
        with Middle():
            with Center():
                yield Static(ART, id="logo")
            with Center():
                yield Static("valorant esports in your terminal", id="tag")
            with Center():
                yield Static("", id="freshness")
            with Center():
                yield Static("[ press enter ]", id="enter-hint")

    def on_mount(self) -> None:
        ts = cache.last_updated()
        fresh = f"cache · {ts} UTC" if ts else "cache empty — start the worker"
        self.query_one("#freshness", Static).update(f"v{__version__}  ·  {fresh}")
        # auto-advance as a fallback so it never gets stuck
        self.set_timer(6.0, self.action_enter_app)

    def action_enter_app(self) -> None:
        if self.is_running:
            self.dismiss()
