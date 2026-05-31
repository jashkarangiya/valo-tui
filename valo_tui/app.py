"""Main ValoTUI app shell: framed context-aware sidebar + content switcher.

The information architecture is *event-first*. The global nav is just
home / events / live / about; selecting an event from the events list focuses
the whole app on that tournament and reveals its sub-pages
(overview / results / fixtures / standings / bracket / teams). Matches,
standings and schedule are therefore children of an event, never global pages.
"""

from __future__ import annotations

from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal
from textual.widgets import ContentSwitcher

from .data import cache
from .data.models import EventCard
from .screens.about import AboutView
from .screens.brackets import BracketView
from .screens.event_overview import EventOverviewView
from .screens.events import EventsView
from .screens.fixtures import FixturesView
from .screens.global_live import GlobalLiveView
from .screens.home import HomeView
from .screens.match_detail import MatchDetailScreen
from .screens.results import ResultsView
from .screens.splash import SplashScreen
from .screens.standings import StandingsView
from .screens.teams import TeamsView
from .screens.widgets import Sidebar, VimDataTable

# Routes that are always available.
GLOBAL_ROUTES = {"home", "events", "live", "about"}
# Routes that only make sense once an event is in focus.
EVENT_ROUTES = {"overview", "results", "fixtures", "standings", "bracket", "teams"}
ALL_ROUTES = GLOBAL_ROUTES | EVENT_ROUTES


class ValoTUI(App):
    TITLE = "valo-tui"
    CSS_PATH = "styles.tcss"

    BINDINGS = [
        # global nav
        Binding("h", "show('home')", "home"),
        Binding("e", "show('events')", "events"),
        Binding("l", "show('live')", "live"),
        Binding("a", "show('about')", "about"),
        # event context nav (no-ops until an event is selected)
        Binding("o", "show('overview')", "overview"),
        Binding("r", "show('results')", "results"),
        Binding("f", "show('fixtures')", "fixtures"),
        Binding("t", "show('standings')", "standings"),
        Binding("b", "show('bracket')", "bracket"),
        Binding("m", "show('teams')", "teams"),
        # app
        Binding("escape", "focus_nav", "nav"),
        Binding("ctrl+r", "refresh", "refresh"),
        Binding("q", "quit", "quit"),
    ]

    def __init__(self) -> None:
        super().__init__()
        # None implies global scope; set to an event id to focus the app.
        self.current_event_id: int | None = None
        self.current_event: EventCard | None = None

    def compose(self) -> ComposeResult:
        with Horizontal(id="frame"):
            yield Sidebar(active="home")
            with ContentSwitcher(initial="home", id="content"):
                yield HomeView(id="home")
                yield EventsView(id="events")
                yield GlobalLiveView(id="live")
                yield AboutView(id="about")
                yield EventOverviewView(id="overview")
                yield ResultsView(id="results")
                yield FixturesView(id="fixtures")
                yield StandingsView(id="standings")
                yield BracketView(id="bracket")
                yield TeamsView(id="teams")

    def on_mount(self) -> None:
        # After the landing page is dismissed, focus the nav so arrow keys work.
        self.push_screen(SplashScreen(), callback=lambda _: self.action_focus_nav())

    # ── routing ──────────────────────────────────────────────
    def action_show(self, route: str) -> None:
        """Jump straight to a page (letter keys), then focus its content."""
        if route in EVENT_ROUTES and self.current_event_id is None:
            self.notify("select an event first  ·  [e] events", timeout=2)
            return
        if route not in ALL_ROUTES:
            return
        self.switch_content(route)
        self.query_one(Sidebar).set_active(route)
        self.focus_content()

    def switch_content(self, route: str) -> None:
        """Switch the visible page without moving focus (used by nav arrows)."""
        if route not in ALL_ROUTES:
            return
        if route in EVENT_ROUTES and self.current_event_id is None:
            return
        self.query_one("#content", ContentSwitcher).current = route
        self._reload(route)

    def select_event(self, event_id: int, tab: str = "overview") -> None:
        """Focus the app on a single event and open one of its sub-pages."""
        self.current_event_id = event_id
        self.current_event = cache.event_by_id(event_id)
        sidebar = self.query_one(Sidebar)
        sidebar.rebuild()
        sidebar.set_active(tab)
        self.switch_content(tab)
        self.focus_content()

    def clear_event(self) -> None:
        """Drop event focus and return to the global events list."""
        self.current_event_id = None
        self.current_event = None
        self.query_one(Sidebar).rebuild()
        self.action_show("events")

    @property
    def event_name(self) -> str:
        return self.current_event.name if self.current_event else ""

    def focus_content(self) -> None:
        cs = self.query_one("#content", ContentSwitcher)
        if not cs.current:
            return
        view = cs.get_child_by_id(cs.current)
        try:
            view.query_one(VimDataTable).focus()
        except Exception:
            if getattr(view, "can_focus", False):
                view.focus()

    def action_focus_nav(self) -> None:
        self.query_one(Sidebar).focus()

    def action_refresh(self) -> None:
        self._reload(self.query_one("#content", ContentSwitcher).current)
        self.notify("refreshed from cache", timeout=2)

    def _reload(self, route: str | None) -> None:
        cs = self.query_one("#content", ContentSwitcher)
        if not route:
            return
        try:
            view = cs.get_child_by_id(route)
        except Exception:
            return
        if hasattr(view, "load_data"):
            view.load_data()

    # ── drill-down ───────────────────────────────────────────
    def on_data_table_row_selected(self, event: VimDataTable.RowSelected) -> None:
        if event.row_key.value is None:
            return
        route = self.query_one("#content", ContentSwitcher).current
        # The events list keys rows by event id; everything else keys by match id.
        if route == "events":
            try:
                self.select_event(int(event.row_key.value))
            except (TypeError, ValueError):
                pass
            return
        if route == "teams":
            return
        try:
            match_id = int(event.row_key.value)
        except (TypeError, ValueError):
            return
        self.push_screen(MatchDetailScreen(match_id))
