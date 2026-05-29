"""Shared widgets and rendering helpers."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.binding import Binding
from textual.widgets import DataTable, Label, Static

from ..data.models import MatchCard

LIVE = "#e87a5d"
MUTED = "#4a708b"
TEXT = "#c8d8e8"

# Navigation map: (key, label, available?)
NAV = [
    ("Circuit", [
        ("g", "global live", True),
        ("r", "regions", False),
        ("i", "international", False),
    ]),
    ("Competition", [
        ("m", "matches", True),
        ("s", "schedule", False),
        ("b", "brackets", True),
        ("t", "standings", False),
    ]),
    ("Deep Dives", [
        ("R", "records", False),
        ("w", "watchlist", False),
        ("x", "compare", False),
    ]),
]


class Sidebar(Static):
    """Static nav rail listing the spec's screen map; live screens highlighted."""

    def compose(self) -> ComposeResult:
        yield Label("◢ valo-tui", classes="brand")
        for group, items in NAV:
            yield Label(f"— {group} —", classes="group")
            for key, label, available in items:
                cls = "nav" if available else "nav-dim"
                yield Label(f"  [{key}] {label}", classes=cls)


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
