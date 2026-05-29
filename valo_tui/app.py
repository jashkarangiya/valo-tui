"""Main ValoTUI app shell: sidebar + content switcher, single-key routing."""

from __future__ import annotations

from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal
from textual.widgets import ContentSwitcher, Footer, Header

from .data import cache
from .screens.brackets import BracketsScreen
from .screens.global_live import GlobalLiveView
from .screens.match_detail import MatchDetailScreen
from .screens.matches import MatchesView
from .screens.splash import SplashScreen
from .screens.widgets import Sidebar, VimDataTable


class ValoTUI(App):
    TITLE = "valo-tui"
    SUB_TITLE = "Valorant esports in your terminal"
    CSS_PATH = "styles.tcss"

    BINDINGS = [
        Binding("g", "show('global')", "Global"),
        Binding("m", "show('matches')", "Matches"),
        Binding("b", "brackets", "Brackets"),
        Binding("r", "refresh", "Refresh"),
        Binding("q", "quit", "Quit"),
    ]

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        with Horizontal(id="body"):
            yield Sidebar()
            with ContentSwitcher(initial="global", id="content"):
                yield GlobalLiveView(id="global")
                yield MatchesView(id="matches")
        yield Footer()

    def on_mount(self) -> None:
        self._sync_subtitle()
        self.push_screen(SplashScreen())

    # ── routing ──────────────────────────────────────────────
    def action_show(self, view: str) -> None:
        switcher = self.query_one("#content", ContentSwitcher)
        switcher.current = view
        self._reload(view)

    def action_brackets(self) -> None:
        self.push_screen(BracketsScreen())

    def action_refresh(self) -> None:
        switcher = self.query_one("#content", ContentSwitcher)
        self._reload(switcher.current)
        self._sync_subtitle()
        self.notify("refreshed from cache", timeout=2)

    def _reload(self, view: str | None) -> None:
        if view == "global":
            self.query_one(GlobalLiveView).load_data()
        elif view == "matches":
            self.query_one(MatchesView).load_data()

    def _sync_subtitle(self) -> None:
        ts = cache.last_updated()
        self.sub_title = f"cache · {ts} UTC" if ts else "cache empty — run the worker"

    # ── drill-down ───────────────────────────────────────────
    def on_data_table_row_selected(self, event: VimDataTable.RowSelected) -> None:
        if event.row_key.value is None:
            return
        try:
            match_id = int(event.row_key.value)
        except (TypeError, ValueError):
            return
        self.push_screen(MatchDetailScreen(match_id))
