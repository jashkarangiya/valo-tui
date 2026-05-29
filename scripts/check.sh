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

echo "[5/6] bracket reconstruction (Champions 2024, event 2097)"
"$PY" - <<'PY'
from valo_tui.data import cache
from valo_tui.screens.brackets import render_bracket
b = cache.bracket(2097)
assert b.has_data, "no bracket data for event 2097"
names = [s.name for s in b.sections]
markup = render_bracket(b, None)
print(f"      sections={names}")
print(f"      rendered {markup.count(chr(10)) + 1} lines")
assert "Grand Final" in names and "┐" in markup
print("      ok")
PY

echo "[6/6] headless app flow (splash -> matches -> detail)"
"$PY" - <<'PY'
import asyncio
from valo_tui.app import ValoTUI
from valo_tui.screens.match_detail import MatchDetailScreen
from valo_tui.data import cache
from textual.widgets import DataTable

async def main():
    app = ValoTUI()
    async with app.run_test(size=(130, 45)) as pilot:
        await pilot.pause()
        for _ in range(20):
            await pilot.pause(0.2)
            if type(app.screen).__name__ != "SplashScreen":
                break
        assert type(app.screen).__name__ != "SplashScreen", "splash never dismissed"
        await pilot.press("m"); await pilot.pause()
        rows = app.query_one("#matches-table").row_count
        assert rows > 0, "no matches in table"
        app.push_screen(MatchDetailScreen(cache.completed_matches()[0].match_id))
        for _ in range(30):
            await pilot.pause(0.3)
            if app.screen.query(DataTable):
                break
        tables = len(app.screen.query(DataTable))
        moms = len(app.screen.query(".momentum"))
        assert tables > 0, "detail rendered no scoreboards"
        print(f"      matches={rows} scoreboards={tables} momentum={moms}")
        print("      ok")

asyncio.run(main())
PY

echo "──────────────────────────────────────────────"
echo " ✓ ALL CHECKS PASSED"
echo "──────────────────────────────────────────────"
