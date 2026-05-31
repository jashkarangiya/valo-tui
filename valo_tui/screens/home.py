"""[h] home — the landing dashboard.

A light front door to the app: what's live right now, how many events are
tracked, and where to go next. Everything here is read from the cache; the
real work happens once you pick an event from [e] events.
"""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import VerticalScroll
from textual.widgets import Label, Static

from ..data import cache
from .widgets import ACCENT, LIVE, MUTED, TEXT, match_line


class HomeView(VerticalScroll):
    can_focus = True

    def compose(self) -> ComposeResult:
        yield Label("home", classes="page-title")
        yield Static(id="home-hero")
        yield Static(id="home-live")
        yield Static(id="home-nav")

    def on_mount(self) -> None:
        self.load_data()

    def load_data(self) -> None:
        live = cache.live_matches()
        events = cache.active_events()
        ongoing = [e for e in events if (e.status or "").lower().startswith("ongo")]
        ts = cache.last_updated()

        hero = (
            f"[bold {TEXT}]valorant esports, in your terminal[/]\n"
            f"[{MUTED}]tracking [/][{TEXT}]{len(events)}[/][{MUTED}] events "
            f"· [/][{TEXT}]{len(ongoing)}[/][{MUTED}] ongoing "
            f"· [/][{LIVE}]{len(live)} live[/]"
        )
        if ts:
            hero += f"\n[{MUTED}]cache · {ts} UTC[/]"
        self.query_one("#home-hero", Static).update(hero)

        if live:
            lines = [f"[bold {LIVE}]● live now[/]"]
            lines += [match_line(m) for m in live[:6]]
        else:
            lines = [f"[bold {MUTED}]live now[/]", f"[{MUTED}]nothing live right now[/]"]
        self.query_one("#home-live", Static).update("\n".join(lines))

        nav = (
            f"[bold {ACCENT}]where to[/]\n"
            f"[{MUTED}]· [/][{TEXT}]e[/][{MUTED}]  events    [/][{TEXT}]"
            f"pick a tournament to open its results, standings & bracket[/]\n"
            f"[{MUTED}]· [/][{TEXT}]l[/][{MUTED}]  live      [/][{TEXT}]"
            f"all live matches across every region[/]\n"
            f"[{MUTED}]· [/][{TEXT}]a[/][{MUTED}]  about     [/][{TEXT}]"
            f"what this tool can do and the key bindings[/]"
        )
        self.query_one("#home-nav", Static).update(nav)
