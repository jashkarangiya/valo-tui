"""Match detail v3 — a tactical Valorant broadcast view.

Big ASCII team logos + centered score; series momentum bars; per-map round
momentum, plus scoreboards grouped by agent role and coloured by team so you
can tell the two sides apart at a glance. Brackets are reachable from here
(press b) only when the match's event actually has a playoff bracket."""

from __future__ import annotations

import pyfiglet

from textual import work
from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal, VerticalScroll
from textual.screen import Screen
from textual.widgets import Footer, Label, Static

from ..data import cache
from ..data.models import MapScore, SeriesDetail
from ..style import bars
from ..style.icons import agent_glyph, agent_role, map_icon
from .widgets import MUTED, SkeletonRow

TEAM1 = "#e8674e"   # left team (accent)
TEAM2 = "#5d9ce8"   # right team (blue)
TEXT = "#c8d8e8"

_BRACKET_PHASES = (
    "playoff", "bracket", "upper", "lower", "grand final",
    "quarterfinal", "semifinal", "final",
)
_ROLE_ORDER = ("duelist", "initiator", "controller", "sentinel", "flex")


def _logo(tag: str) -> str:
    try:
        return pyfiglet.figlet_format(tag[:4], font="ansi_shadow").rstrip("\n")
    except Exception:
        return tag


class MatchDetailScreen(Screen):
    BINDINGS = [
        Binding("escape,q", "app.pop_screen", "Back"),
        Binding("b", "open_bracket", "Bracket"),
        Binding("j,down", "scroll_down", "Down", show=False),
        Binding("k,up", "scroll_up", "Up", show=False),
    ]

    def __init__(self, match_id: int) -> None:
        super().__init__()
        self.match_id = match_id
        self._detail: SeriesDetail | None = None

    def compose(self) -> ComposeResult:
        with VerticalScroll(id="detail-scroll"):
            yield SkeletonRow(cells=6)
            yield Label(f"[{MUTED}]loading match {self.match_id}…[/]")
        yield Footer()

    def on_mount(self) -> None:
        self._load()

    @work(thread=True, exclusive=True)
    def _load(self) -> None:
        detail = cache.series_detail(self.match_id)
        self.app.call_from_thread(self._render_detail, detail)

    # ── contextual brackets ──────────────────────────────────
    def _has_bracket(self) -> bool:
        if self._detail is None:
            return False
        text = f"{self._detail.phase}".lower()
        return any(k in text for k in _BRACKET_PHASES)

    def _event_id(self) -> int | None:
        if self._detail is None:
            return None
        name = self._detail.event.strip().lower()
        for e in cache.active_events():
            if e.name.strip().lower() == name:
                return e.id
        return None

    def action_open_bracket(self) -> None:
        if not self._has_bracket():
            self.app.notify("no bracket for this event", timeout=2)
            return
        eid = self._event_id()
        if eid is None:
            self.app.notify("bracket unavailable (event not in cache)", timeout=2)
            return
        from .brackets import BracketsScreen

        self.app.push_screen(BracketsScreen(eid))

    # ── render ───────────────────────────────────────────────
    def _render_detail(self, detail: SeriesDetail | None) -> None:
        self._detail = detail
        body = self.query_one("#detail-scroll", VerticalScroll)
        body.remove_children()
        if detail is None:
            body.mount(Label(f"[{MUTED}]no detail available for this match[/]"))
            return

        self._mount_header(body, detail)
        maps = [m for m in detail.maps if not m.is_aggregate]
        self._mount_series_momentum(body, detail, maps)
        if not maps:
            body.mount(Label(f"[{MUTED}]no map data yet[/]"))
            return
        for m in maps:
            self._mount_map(body, detail, m)

    def _mount_header(self, body: VerticalScroll, d: SeriesDetail) -> None:
        t1 = (d.team1.short or d.team1.name[:4]).upper()
        t2 = (d.team2.short or d.team2.name[:4]).upper()
        s1 = d.team1.score if d.team1.score is not None else "–"
        s2 = d.team2.score if d.team2.score is not None else "–"
        if d.is_live:
            status = f"[{TEAM1}]● live[/]"
        elif d.is_completed:
            status = f"[{MUTED}]✓ final[/]"
        else:
            status = f"[{MUTED}]○ {d.remaining or 'upcoming'}[/]"
        bo = f" · {d.best_of}" if d.best_of else ""
        center = (
            f"\n\n[bold {TEAM1}]{s1}[/] [{MUTED}]–[/] [bold {TEAM2}]{s2}[/]\n"
            f"[{MUTED}]{bo.strip(' ·')}[/]   {status}"
        )
        body.mount(
            Horizontal(
                Static(_logo(t1), classes="logo-l"),
                Static(center, classes="score-center"),
                Static(_logo(t2), classes="logo-r"),
                id="match-header",
            )
        )
        sub = f"[{MUTED}]{d.event} · {d.phase}[/]"
        if self._has_bracket():
            sub += f"    [{TEAM1}][b][/] [{TEXT}]bracket[/]"
        body.mount(Label(sub, classes="detail-sub"))
        intel = self._intel(d)
        if intel:
            body.mount(Label(intel, classes="intel"))

    def _intel(self, d: SeriesDetail) -> str | None:
        players = [p for m in d.maps for p in m.players]
        if not players:
            return None
        top = max(players, key=lambda p: (p.acs or 0))
        fk = max(players, key=lambda p: (p.fk or 0))
        return (
            f"[{MUTED}]intel:  top acs [/][{TEXT}]{top.name} {top.acs or 0}[/]"
            f"[{MUTED}]   ·   most FK [/][{TEXT}]{fk.name} {fk.fk or 0}[/]"
        )

    def _mount_series_momentum(self, body, d: SeriesDetail, maps: list[MapScore]) -> None:
        played = [m for m in maps if m.has_score]
        if not played:
            return
        body.mount(Label(f"[{TEAM1}]series momentum[/]", classes="map-title"))
        for m in played:
            bar = bars.winbar(m.team1_score, m.team2_score)
            pick = d.pick_label(m.name)
            pick_txt = f"   [{MUTED}]{pick}[/]" if pick else ""
            line = (
                f"{map_icon(m.name)} [{TEXT}]{m.name:<9}[/] {bar}  "
                f"[{TEXT}]{m.team1_score}–{m.team2_score}[/]{pick_txt}"
            )
            body.mount(Label(line, classes="mom-line"))

    def _mount_map(self, body: VerticalScroll, d: SeriesDetail, m: MapScore) -> None:
        body.mount(Label(self._map_title(d, m), classes="map-title"))
        if m.state == "pending":
            for _ in range(3):
                body.mount(SkeletonRow(cells=5))
            return
        if m.rounds:
            body.mount(Label(f"  rounds  {bars.momentum(m.rounds, m.team1_short)}",
                             classes="momentum"))
        body.mount(Static(self._scoreboard(m), classes="scoreboard"))

    def _map_title(self, d: SeriesDetail, m: MapScore) -> str:
        score = (f"[{TEXT}]{m.team1_score}–{m.team2_score}[/]"
                 if m.has_score else f"[{MUTED}]TBD[/]")
        pick = d.pick_label(m.name)
        pick_txt = f"   [{MUTED}]{pick}[/]" if pick else ""
        return f"{map_icon(m.name)} [bold {TEXT}]{m.name}[/]   {score}{pick_txt}"

    def _scoreboard(self, m: MapScore) -> str:
        shorts = [(m.team1_short, TEAM1), (m.team2_short, TEAM2)]
        lines = []
        hdr = f"[{MUTED}]{'':<15}{'acs':>4} {'k':>3} {'d':>3} {'a':>3} {'adr':>5} {'hs':>4}[/]"
        for short, colour in shorts:
            team_players = [p for p in m.players if (p.team_short or "") == short]
            if not team_players:
                continue
            lines.append(f"[bold {colour}]{short or '?'}[/]")
            lines.append(hdr)
            by_role: dict[str, list] = {}
            for p in team_players:
                by_role.setdefault(agent_role(p.agents[0] if p.agents else None) or "flex", []).append(p)
            for role in _ROLE_ORDER:
                ps = by_role.get(role)
                if not ps:
                    continue
                lines.append(f"  [{MUTED}]{role}s[/]")
                for p in sorted(ps, key=lambda x: (x.acs or 0), reverse=True):
                    lines.append(self._player_line(p, colour))
            lines.append("")
        return "\n".join(lines)

    def _player_line(self, p, colour: str) -> str:
        glyph, gcol = agent_glyph(p.agents[0] if p.agents else None)
        agent = (p.agents[0] if p.agents else "—")[:8]
        name = p.name[:12]
        return (
            f"    [{colour}]{name:<12}[/] [{gcol}]{glyph}[/] [{MUTED}]{agent:<8}[/]"
            f" {_n(p.acs):>4} {_n(p.k):>3} {_n(p.d):>3} {_n(p.a):>3}"
            f" {_f(p.adr):>5} {_pct(p.hs_pct):>4}"
        )


def _n(v) -> str:
    return "–" if v is None else str(v)


def _f(v) -> str:
    return "–" if v is None else f"{v:.0f}"


def _pct(v) -> str:
    return "–" if v is None else f"{v:.0f}%"
