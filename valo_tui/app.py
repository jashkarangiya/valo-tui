"""Main ValoTUI app shell: framed sidebar + content switcher, single-key
page routing, landing page on startup."""

from __future__ import annotations

from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal
from textual.widgets import ContentSwitcher

from .screens.about import AboutView
from .screens.brackets import BracketsScreen
from .screens.global_live import GlobalLiveView
from .screens.match_detail import MatchDetailScreen
from .screens.matches import MatchesView
from .screens.schedule import ScheduleView
from .screens.splash import SplashScreen
from .screens.standings import StandingsView
from .screens.widgets import Sidebar, VimDataTable

# route -> view id in the ContentSwitcher (brackets is a pushed screen).
ROUTES = {"live", "matches", "standings", "schedule", "about"}


class ValoTUI(App):
    TITLE = "valo-tui"
    CSS_PATH = "styles.tcss"

    BINDINGS = [
        Binding("g", "show('live')", "live"),
        Binding("m", "show('matches')", "matches"),
        Binding("t", "show('standings')", "standings"),
        Binding("s", "show('schedule')", "schedule"),
        Binding("b", "show('brackets')", "brackets"),
        Binding("a", "show('about')", "about"),
        Binding("r", "refresh", "refresh"),
        Binding("q", "quit", "quit"),
    ]

    def compose(self) -> ComposeResult:
        with Horizontal(id="frame"):
            yield Sidebar(active="live")
            with ContentSwitcher(initial="live", id="content"):
                yield GlobalLiveView(id="live")
                yield MatchesView(id="matches")
                yield StandingsView(id="standings")
                yield ScheduleView(id="schedule")
                yield AboutView(id="about")

    def on_mount(self) -> None:
        self.push_screen(SplashScreen())

    # ── routing ──────────────────────────────────────────────
    def action_show(self, route: str) -> None:
        if route == "brackets":
            self.push_screen(BracketsScreen())
            return
        if route not in ROUTES:
            return
        self.query_one("#content", ContentSwitcher).current = route
        self.query_one(Sidebar).set_active(route)
        self._reload(route)

    def action_refresh(self) -> None:
        self._reload(self.query_one("#content", ContentSwitcher).current)
        self.notify("refreshed from cache", timeout=2)

    def _reload(self, route: str | None) -> None:
        view = {
            "live": GlobalLiveView,
            "matches": MatchesView,
            "standings": StandingsView,
            "schedule": ScheduleView,
            "about": AboutView,
        }.get(route or "")
        if view is not None:
            self.query_one(view).load_data()

    # ── drill-down ───────────────────────────────────────────
    def on_data_table_row_selected(self, event: VimDataTable.RowSelected) -> None:
        if event.row_key.value is None:
            return
        try:
            match_id = int(event.row_key.value)
        except (TypeError, ValueError):
            return
        self.push_screen(MatchDetailScreen(match_id))
