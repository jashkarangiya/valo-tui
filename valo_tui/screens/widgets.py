"""Shared widgets and rendering helpers."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.binding import Binding
from textual.widgets import DataTable, Label, Static

from ..data.models import MatchCard

LIVE = "#e8674e"
ACCENT = "#e8674e"
MUTED = "#4a708b"
TEXT = "#c8d8e8"
RULE = "#1c3a52"

BRAND = "valo-tui · vct26"

# Global nav: (key, route, label). Always visible.
GLOBAL_NAV = [
    ("h", "home", "home"),
    ("e", "events", "events"),
    ("l", "live", "live"),
    ("a", "about", "about"),
]

# Event-context nav: revealed only while an event is in focus. These are the
# children of an event, never global pages.
EVENT_NAV = [
    ("o", "overview", "overview"),
    ("r", "results", "results"),
    ("f", "fixtures", "fixtures"),
    ("t", "standings", "standings"),
    ("b", "bracket", "bracket"),
    ("m", "teams", "teams"),
]


class Sidebar(Static):
    """Focusable, context-aware nav rail.

    Shows the global section always; once an event is selected the app reveals
    that event's sub-pages beneath it. Up/Down move the highlight (switching the
    page live); Enter/Right enters the content.
    """

    can_focus = True

    BINDINGS = [
        Binding("up,k", "nav(-1)", "Up", show=False),
        Binding("down,j", "nav(1)", "Down", show=False),
        Binding("enter,right,l", "enter", "Open", show=False),
    ]

    def __init__(self, active: str = "home", **kwargs) -> None:
        self._active = active
        self._focused = False
        super().__init__("", **kwargs)

    def on_mount(self) -> None:
        self.update(self._markup())

    def _nav(self) -> list[tuple[str, str, str]]:
        """The currently selectable items (global, plus event items if any)."""
        items = list(GLOBAL_NAV)
        if getattr(self.app, "current_event_id", None) is not None:
            items += EVENT_NAV
        return items

    def _index(self) -> int:
        for i, (_, route, _) in enumerate(self._nav()):
            if route == self._active:
                return i
        return 0

    @property
    def _route(self) -> str:
        return self._nav()[self._index()][1]

    def set_active(self, route: str) -> None:
        self._active = route
        self.update(self._markup())

    def rebuild(self) -> None:
        """Re-render after the event context changed."""
        self.update(self._markup())

    def action_nav(self, delta: int) -> None:
        nav = self._nav()
        self._active = nav[(self._index() + delta) % len(nav)][1]
        self.update(self._markup())
        self.app.switch_content(self._active)

    def action_enter(self) -> None:
        self.app.switch_content(self._route)
        self.app.focus_content()

    def _markup(self) -> str:
        active = self._active
        focused = self._focused

        def row(key: str, route: str, label: str) -> str:
            if route == active:
                marker = f"[{ACCENT}]›[/]" if focused else " "
                return f"{marker}[{MUTED}][{key}][/] [bold {ACCENT}]{label}[/]"
            return f" [{MUTED}][{key}][/] [{TEXT}]{label}[/]"

        lines = [f"[bold {TEXT}]{BRAND}[/]", f"[{RULE}]{'─' * 20}[/]", ""]
        lines += [row(k, r, lbl) for k, r, lbl in GLOBAL_NAV]

        if getattr(self.app, "current_event_id", None) is not None:
            name = _clip(getattr(self.app, "event_name", "") or "event", 18)
            lines += [
                "",
                f"[{MUTED}]── event ──[/]",
                f"[bold {ACCENT}]{name}[/]",
                "",
            ]
            lines += [row(k, r, lbl) for k, r, lbl in EVENT_NAV]

        lines += [
            "",
            f"[{RULE}]{'─' * 20}[/]",
            "",
            f"[{MUTED}]↑↓    navigate[/]",
            f"[{MUTED}]enter open[/]",
            f"[{MUTED}]esc   back here[/]",
            f"[{MUTED}]q     quit[/]",
        ]
        return "\n".join(lines)

    def on_focus(self) -> None:
        self._focused = True
        self.update(self._markup())

    def on_blur(self) -> None:
        self._focused = False
        self.update(self._markup())


class VimDataTable(DataTable):
    """DataTable with vim-style j/k bound to its built-in cursor actions."""

    BINDINGS = [
        Binding("j", "cursor_down", "Down", show=False),
        Binding("k", "cursor_up", "Up", show=False),
    ]


class LiveDot(Static):
    """A live indicator that pulses ● ↔ ○ on a slow interval."""

    DEFAULT_CSS = "LiveDot { width: auto; color: #e87a5d; }"

    def __init__(self, label: str = "LIVE", **kwargs) -> None:
        self._label = label
        self._on = True
        super().__init__(self._markup(), **kwargs)

    def _markup(self) -> str:
        dot = "●" if self._on else "○"
        return f"[{LIVE}]{dot} {self._label}[/]" if self._label else f"[{LIVE}]{dot}[/]"

    def on_mount(self) -> None:
        self.set_interval(0.7, self._tick)

    def _tick(self) -> None:
        self._on = not self._on
        self.update(self._markup())


class SkeletonRow(Static):
    """A shimmering placeholder row shown while real data is unavailable."""

    DEFAULT_CSS = "SkeletonRow { width: 1fr; height: 1; color: #22384a; }"
    _FRAMES = ["░░░░", "▒▒▒▒", "▓▓▓▓", "▒▒▒▒"]

    def __init__(self, cells: int = 8, **kwargs) -> None:
        self._cells = cells
        self._i = 0
        super().__init__(self._frame(), **kwargs)

    def _frame(self) -> str:
        return "   ".join([self._FRAMES[self._i]] * self._cells)

    def on_mount(self) -> None:
        self.set_interval(0.18, self._tick)

    def _tick(self) -> None:
        self._i = (self._i + 1) % len(self._FRAMES)
        self.update(self._frame())


def match_line(m: MatchCard) -> str:
    """One-line Rich-markup summary of a match for the dashboard panels."""
    if m.is_live:
        dot = f"[{LIVE}]●[/] "
        score = f"[{LIVE}]{_score(m)}[/]"
    elif m.status == "completed":
        dot = f"[{MUTED}]·[/] "
        score = f"[{MUTED}]{_score(m)}[/]"
    else:
        dot = f"[{MUTED}]○[/] "
        score = f"[{MUTED}]{m.time or 'soon'}[/]"
    t1 = _clip(m.team1.name, 12)
    t2 = _clip(m.team2.name, 12)
    return f"{dot}{t1} [{MUTED}]vs[/] {t2}  {score}"


def _score(m: MatchCard) -> str:
    s1 = m.team1.score if m.team1.score is not None else "–"
    s2 = m.team2.score if m.team2.score is not None else "–"
    return f"{s1}–{s2}"


def _clip(s: str, n: int) -> str:
    return s if len(s) <= n else s[: n - 1] + "…"
