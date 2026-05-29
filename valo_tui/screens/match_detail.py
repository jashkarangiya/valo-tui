"""Match detail — series header, map vetoes, and per-map scoreboards.

Pushed on top of the dashboard when the user presses Enter on a match. The
series fetch can hit the network (read-through cache), so it runs in a worker
thread to keep the UI responsive."""

from __future__ import annotations

from rich.text import Text

from textual import work
from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import VerticalScroll
from textual.screen import Screen
from textual.widgets import Footer, Label

from ..data import cache
from ..data.models import MapScore, SeriesDetail
from .widgets import LIVE, MUTED, TEXT, VimDataTable

_COLS = [
    ("player", 16),
    ("agent", 10),
    ("acs", 5),
    ("k", 4),
    ("d", 4),
    ("a", 4),
    ("adr", 5),
    ("hs%", 5),
    ("fk", 4),
    ("fd", 4),
]


class MatchDetailScreen(Screen):
    BINDINGS = [
        Binding("escape,q", "app.pop_screen", "Back"),
        Binding("j", "scroll_down", "Down", show=False),
        Binding("k", "scroll_up", "Up", show=False),
    ]

    def __init__(self, match_id: int) -> None:
        super().__init__()
        self.match_id = match_id

    def compose(self) -> ComposeResult:
        with VerticalScroll(id="detail-scroll"):
            yield Label(f"[{MUTED}]loading match {self.match_id}…[/]", id="detail-status")
        yield Footer()

    def on_mount(self) -> None:
        self._load()

    @work(thread=True, exclusive=True)
    def _load(self) -> None:
        detail = cache.series_detail(self.match_id)
        self.app.call_from_thread(self._render_detail, detail)

    def _render_detail(self, detail: SeriesDetail | None) -> None:
        body = self.query_one("#detail-scroll", VerticalScroll)
        body.remove_children()
        if detail is None:
            body.mount(Label(f"[{MUTED}]no detail available for this match[/]"))
            return

        body.mount(Label(self._header(detail), classes="series-header"))
        if detail.vetoes:
            body.mount(Label(self._vetoes(detail), classes="veto"))

        if not detail.maps:
            body.mount(Label(f"[{MUTED}]no map data yet[/]"))
            return

        for m in detail.maps:
            title = self._map_title(m)
            body.mount(Label(title, classes="map-title"))
            body.mount(self._map_table(m))

    # ── rendering helpers ────────────────────────────────────
    def _header(self, d: SeriesDetail) -> str:
        s1 = d.team1.score if d.team1.score is not None else "–"
        s2 = d.team2.score if d.team2.score is not None else "–"
        bo = f"  [{MUTED}]{d.best_of}[/]" if d.best_of else ""
        note = f"  [{LIVE}]{d.status_note}[/]" if d.status_note else ""
        line1 = f"[bold {TEXT}]{d.team1.name}[/]  [{LIVE}]{s1} – {s2}[/]  [bold {TEXT}]{d.team2.name}[/]{bo}{note}"
        line2 = f"[{MUTED}]{d.event} · {d.phase}[/]"
        return f"{line1}\n{line2}"

    def _vetoes(self, d: SeriesDetail) -> str:
        parts = []
        for v in d.vetoes:
            verb = v.action.lower()
            if verb == "ban":
                parts.append(f"[{MUTED}]{v.team} ban {v.map}[/]")
            elif verb == "pick":
                parts.append(f"[{TEXT}]{v.team} pick {v.map}[/]")
            else:
                parts.append(f"[{MUTED}]{v.map} ({verb})[/]")
        return "veto: " + "  ·  ".join(parts)

    def _map_title(self, m: MapScore) -> str:
        if m.team1_score is not None and m.team2_score is not None:
            return f"{m.name}  [{LIVE}]{m.team1_score}–{m.team2_score}[/]"
        return f"{m.name}  [{MUTED}](all maps)[/]"

    def _map_table(self, m: MapScore) -> VimDataTable:
        table = VimDataTable(cursor_type="row", zebra_stripes=False)
        for name, width in _COLS:
            table.add_column(name, width=width)
        for p in sorted(m.players, key=lambda p: (p.acs or 0), reverse=True):
            table.add_row(
                Text(p.name, style=TEXT),
                Text(", ".join(p.agents) or "—", style=MUTED),
                _num(p.acs),
                _num(p.k),
                _num(p.d),
                _num(p.a),
                _num(p.adr),
                _pct(p.hs_pct),
                _num(p.fk),
                _num(p.fd),
            )
        # Size the table to its contents so multiple maps stack in the scroll.
        table.styles.height = len(m.players) + 1
        return table


def _num(v) -> Text:
    return Text("–" if v is None else (f"{v:.0f}" if isinstance(v, float) else str(v)))


def _pct(v) -> Text:
    return Text("–" if v is None else f"{v:.0f}%")
