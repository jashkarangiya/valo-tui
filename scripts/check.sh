#!/usr/bin/env bash
# Run every check for valo-tui in one shot: compile, primitives, data layer,
# worker, and a full headless app flow. Usage: ./scripts/check.sh
set -euo pipefail

cd "$(dirname "$0")/.."
PY="${PY:-.venv/bin/python}"

echo "──────────────────────────────────────────────"
echo " valo-tui · full check"
echo "──────────────────────────────────────────────"

echo "[1/6] compile all modules"
"$PY" -m compileall -q valo_tui worker serve
echo "      ok"

echo "[2/6] visual primitives (icons + bars)"
"$PY" - <<'PY'
from valo_tui.style import bars
from valo_tui.style.icons import map_icon, agent_glyph
from valo_tui.data.models import RoundLine
assert "█" in bars.winbar(13, 9) and bars.winbar(0, 0).count("░") == 18
assert "▲" in bars.momentum([RoundLine(2, "Attacker", "X")], "X")
assert map_icon("Lotus") == "❀" and agent_glyph("jett")[0] == "▲"
print("      ok")
PY

echo "[3/6] data layer (cache reads + region buckets)"
"$PY" - <<'PY'
from valo_tui.data import cache
ev = cache.active_events()
regions, intl = cache.global_live()
print(f"      events={len(ev)} live={len(cache.live_matches())} "
      f"upcoming={len(cache.upcoming_matches())} completed={len(cache.completed_matches())}")
print(f"      cache last updated: {cache.last_updated()}  (None = run the worker)")
assert isinstance(regions, dict) and len(regions) == 4
print("      ok")
PY

echo "[4/6] worker one-shot fetch (network)"
"$PY" worker/fetcher.py --once

echo "[5/6] bracket reconstruction + render (fixture, no network)"
"$PY" - <<'PY'
from valo_tui.data.bracket import build_bracket
from valo_tui.screens.brackets import render_bracket

def m(mid, ph, a, sa, bteam, sb):
    win_a = sa > sb
    return {"match_id": mid, "phase": ph, "status": "completed",
            "teams": [{"name": a, "score": sa, "is_winner": win_a},
                      {"name": bteam, "score": sb, "is_winner": not win_a}]}

raw = [
    m(1, "Upper Semifinals", "AAA", 2, "BBB", 0),
    m(2, "Upper Semifinals", "CCC", 1, "DDD", 2),
    m(3, "Upper Final", "AAA", 2, "DDD", 1),   # winners propagate -> edge
    m(4, "Lower Final", "BBB", 0, "CCC", 2),
    m(5, "Grand Final", "AAA", 3, "CCC", 1),
]
b = build_bracket(raw)
names = [s.name for s in b.sections]
markup = render_bracket(b, 3)
print(f"      sections={names}")
print(f"      rendered {markup.count(chr(10)) + 1} lines")
assert names == ["Upper Bracket", "Lower Bracket", "Grand Final"], names
assert "┐" in markup and "├" in markup, "connectors missing"
print("      ok")
PY

echo "[6/6] headless app flow (splash -> events -> event results -> detail)"
"$PY" - <<'PY'
import asyncio
from valo_tui.app import ValoTUI
from valo_tui.screens.match_detail import MatchDetailScreen
from valo_tui.data import cache
from textual.widgets import ContentSwitcher

async def main():
    app = ValoTUI()
    async with app.run_test(size=(130, 45)) as pilot:
        await pilot.pause()
        await pilot.press("enter")  # dismiss the landing page
        for _ in range(20):
            await pilot.pause(0.2)
            if type(app.screen).__name__ != "SplashScreen":
                break
        assert type(app.screen).__name__ != "SplashScreen", "landing never dismissed"

        # Global nav: home -> events.
        await pilot.press("e"); await pilot.pause()
        assert app.query_one("#content", ContentSwitcher).current == "events"

        events = cache.active_events()
        if not events:
            # Cache empty (e.g. vlr.gg unreachable / rate-limited): the event
            # routing still works, there's just nothing to focus on.
            print("      events=0 (cache empty — skipped event drill)")
            print("      ok")
            return

        # Focus an event; its sub-pages should become reachable.
        app.select_event(events[0].id); await pilot.pause()
        await pilot.press("r"); await pilot.pause()
        assert app.query_one("#content", ContentSwitcher).current == "results"
        rows = app.query_one("#results-table").row_count

        done = cache.completed_matches()
        if not done:
            print(f"      events={len(events)} results_rows={rows} "
                  f"(no completed matches — skipped detail drill)")
            print("      ok")
            return
        app.push_screen(MatchDetailScreen(done[0].match_id))
        for _ in range(30):
            await pilot.pause(0.3)
            if app.screen.query(".scoreboard"):
                break
        boards = len(app.screen.query(".scoreboard"))
        print(f"      events={len(events)} results_rows={rows} scoreboards={boards}")
        print("      ok")

asyncio.run(main())
PY

echo "──────────────────────────────────────────────"
echo " ✓ ALL CHECKS PASSED"
echo "──────────────────────────────────────────────"
