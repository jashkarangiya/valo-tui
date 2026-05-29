"""Splash screen — ASCII logo, version, cache freshness, then fades to the
dashboard. Pushed over the main screen at startup and dismisses itself."""

from __future__ import annotations

import pyfiglet

from textual.app import ComposeResult
from textual.containers import Center, Middle
from textual.screen import Screen
from textual.widgets import Static

from .. import __version__
from ..data import cache

ART = pyfiglet.figlet_format("valo-tui", font="slant")


class SplashScreen(Screen):
    CSS = """
    SplashScreen { background: #0a1822; align: center middle; }
    #logo { color: #e87a5d; text-style: bold; width: auto; }
    #tag  { color: #4a708b; width: auto; margin-top: 1; }
    #freshness { color: #4a708b; width: auto; margin-top: 1; }
    """

    def compose(self) -> ComposeResult:
        with Middle():
            with Center():
                yield Static(ART, id="logo")
            with Center():
                yield Static("valorant esports in your terminal", id="tag")
            with Center():
                yield Static("", id="freshness")

    def on_mount(self) -> None:
        ts = cache.last_updated()
        fresh = f"cache · {ts} UTC" if ts else "cache empty — start the worker"
        self.query_one("#freshness", Static).update(f"v{__version__}  ·  {fresh}")
        self.set_timer(1.3, self._fade_out)

    def _fade_out(self) -> None:
        self.styles.animate("opacity", 0.0, duration=0.4, on_complete=self.dismiss)
