"""Seed an isolated demo cache with fixture data and capture SVG screenshots
of every screen, so you can *see* valo-tui without a live terminal or network.

Run:  VALO_TUI_DB=/tmp/valo_demo.db .venv/bin/python scripts/shots.py
Output: ./screenshots/*.svg
"""

from __future__ import annotations

import asyncio
import random
from pathlib import Path

from valo_tui.data import cache
from valo_tui.data.serialize import dumps

random.seed(7)
AGENTS = ["jett", "omen", "sova", "killjoy", "raze", "viper", "breach", "cypher", "neon", "fade"]


def _players(team_short: str, base: int):
    out = []
    for i in range(5):
        k = random.randint(10, 28)
        out.append({
            "name": f"{team_short.lower()}_{['atk','smk','init','sen','flex'][i]}",
            "team_short": team_short,
            "agents": [AGENTS[(base + i) % len(AGENTS)]],
            "acs": random.randint(150, 320), "k": k, "d": random.randint(8, 20),
            "a": random.randint(2, 12), "adr": round(random.uniform(120, 190), 0),
            "hs_pct": round(random.uniform(18, 42), 0),
            "fk": random.randint(0, 6), "fd": random.randint(0, 6),
        })
    return out


def _rounds(n: int, t1: str, t2: str, t1_wins: int):
    rounds, w1 = [], 0
    for i in range(n):
        if w1 < t1_wins and (random.random() < 0.55 or (n - i) <= (t1_wins - w1)):
            win, w1 = t1, w1 + 1
        else:
            win = t2
        rounds.append({
            "number": i + 1,
            "winner_side": random.choice(["Attacker", "Defender"]),
            "winner_team_short": win,
        })
    return rounds


def _map(name, t1, s1, t2, s2):
    return {
        "map_name": name,
        "teams": [{"short": t1, "score": s1}, {"short": t2, "score": s2}],
        "players": _players(t1, 0) + _players(t2, 3),
        "rounds": _rounds(s1 + s2, t1, t2, s1),
    }


def seed_series(match_id: int):
    info = {
        "match_id": match_id,
        "teams": [{"name": "NRG", "short": "NRG"}, {"name": "Sentinels", "short": "SEN"}],
        "score": [2, 1], "status_note": "final",
        "event": "VCT 2026: Americas Stage 1", "event_phase": "Playoffs · Upper Final",
        "best_of": "Bo3",
        "map_actions": [
            {"action": "ban", "team": "NRG", "map": "Pearl"},
            {"action": "ban", "team": "SEN", "map": "Icebox"},
            {"action": "pick", "team": "NRG", "map": "Lotus"},
            {"action": "pick", "team": "SEN", "map": "Bind"},
            {"action": "remaining", "team": "", "map": "Ascent"},
        ],
    }
    maps = [_map("Lotus", "NRG", 13, "SEN", 9),
            _map("Bind", "NRG", 11, "SEN", 13),
            _map("Ascent", "NRG", 13, "SEN", 7)]
    cache._store_kv(f"series:{match_id}", {"info": info, "maps": maps})


def seed_bracket(event_id: int):
    def m(mid, ph, a, sa, b, sb):
        return {"match_id": mid, "phase": ph, "status": "completed",
                "teams": [{"name": a, "score": sa, "is_winner": sa > sb},
                          {"name": b, "score": sb, "is_winner": sb > sa}]}
    raw = [
        m(101, "Upper Quarterfinals", "NRG", 2, "KRÜ", 0),
        m(102, "Upper Quarterfinals", "LOUD", 1, "MIBR", 2),
        m(103, "Upper Quarterfinals", "Sentinels", 2, "100T", 1),
        m(104, "Upper Quarterfinals", "LEVIATÁN", 0, "G2", 2),
        m(105, "Upper Semifinals", "NRG", 2, "MIBR", 1),
        m(106, "Upper Semifinals", "Sentinels", 0, "G2", 2),
        m(107, "Upper Final", "NRG", 2, "G2", 1),
        m(108, "Lower Round 1", "KRÜ", 2, "LOUD", 0),
        m(109, "Lower Round 1", "100T", 1, "LEVIATÁN", 2),
        m(110, "Lower Round 2", "MIBR", 2, "KRÜ", 1),
        m(111, "Lower Round 2", "Sentinels", 2, "LEVIATÁN", 0),
        m(112, "Lower Round 3", "MIBR", 1, "Sentinels", 2),
        m(113, "Lower Final", "G2", 2, "Sentinels", 1),
        m(114, "Grand Final", "NRG", 3, "G2", 2),
    ]
    cache._store_kv(f"event:matches:{event_id}", raw)


def seed_lists():
    def match(mid, a, sa, b, sb, event, phase, status, time=None):
        return {"match_id": mid, "team1": {"name": a, "score": sa},
                "team2": {"name": b, "score": sb}, "event": event,
                "event_phase": phase, "status": status, "time": time}
    live = [
        match(201, "NRG", 1, "Sentinels", 0, "VCT 2026: Americas Stage 1", "Playoffs", "live"),
        match(202, "Fnatic", 0, "Team Liquid", 1, "VCT 2026: EMEA Stage 1", "Playoffs", "live"),
    ]
    upcoming = [
        match(208, "Paper Rex", None, "EDG", None, "VCT Masters 2026", "Playoffs", "upcoming", "in 1h"),
        match(209, "Sentinels", None, "FNATIC", None, "Esports World Cup 2026", "Group A", "upcoming", "in 3h"),
        match(203, "DRX", None, "Gen.G", None, "VCT 2026: Pacific Stage 1", "Week 3", "upcoming", "in 2h"),
        match(204, "EDG", None, "Bilibili", None, "VCT 2026: China Stage 1", "Week 3", "upcoming", "in 5h"),
        match(205, "100T", None, "LOUD", None, "VCT 2026: Americas Stage 1", "Week 3", "upcoming", "in 1d"),
    ]
    completed = [
        match(206, "G2", 2, "MIBR", 0, "VCT 2026: Americas Stage 1", "Playoffs", "completed"),
        match(207, "Paper Rex", 2, "T1", 1, "VCT Masters 2026", "Group Stage", "completed"),
    ]
    events = [
        {"id": 301, "name": "VCT 2026: Americas Stage 1", "status": "ongoing", "region": "United States"},
        {"id": 302, "name": "VCT Masters 2026", "status": "ongoing", "region": "International"},
    ]
    cache._store_kv("matches:live", live)
    cache._store_kv("matches:upcoming", upcoming)
    cache._store_kv("matches:completed", completed)
    cache._store_kv("events:active", events)


async def main():
    out = Path("screenshots")
    out.mkdir(exist_ok=True)
    seed_lists()
    seed_series(5000)
    seed_bracket(7000)

    from valo_tui.app import ValoTUI
    from valo_tui.screens.brackets import BracketsScreen, BracketWidget
    from valo_tui.screens.match_detail import MatchDetailScreen
    from textual.widgets import DataTable

    app = ValoTUI()
    async with app.run_test(size=(132, 46)) as pilot:
        # splash (capture before it fades)
        await pilot.pause(0.2)
        app.save_screenshot("01_landing.svg", path=str(out))
        # dismiss the landing page
        await pilot.press("enter")
        for _ in range(20):
            await pilot.pause(0.2)
            if type(app.screen).__name__ != "SplashScreen":
                break
        # content pages
        await pilot.press("g"); await pilot.pause()
        app.save_screenshot("02_live.svg", path=str(out))
        await pilot.press("m"); await pilot.pause()
        app.save_screenshot("03_matches.svg", path=str(out))
        await pilot.press("t"); await pilot.pause()
        app.save_screenshot("04_standings.svg", path=str(out))
        await pilot.press("s"); await pilot.pause()
        app.save_screenshot("05_schedule.svg", path=str(out))
        await pilot.press("a"); await pilot.pause()
        app.save_screenshot("06_about.svg", path=str(out))
        # match detail v2
        await pilot.press("m"); await pilot.pause()
        app.push_screen(MatchDetailScreen(5000))
        for _ in range(20):
            await pilot.pause(0.3)
            if app.screen.query(DataTable):
                break
        app.save_screenshot("07_match_detail.svg", path=str(out))
        await pilot.press("escape"); await pilot.pause()
        # brackets — plain sleeps so we don't block on screen idle
        app.push_screen(BracketsScreen(7000))
        for _ in range(20):
            await asyncio.sleep(0.3)
            if app.screen.query(BracketWidget):
                break
        await asyncio.sleep(0.3)
        app.save_screenshot("08_brackets.svg", path=str(out))

    print("saved screenshots:")
    for f in sorted(out.glob("*.svg")):
        print(" ", f)


if __name__ == "__main__":
    asyncio.run(main())
