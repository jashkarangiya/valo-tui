"""Block-character bar and sparkline primitives.

All helpers return Rich-markup strings, intended for ``Static.update`` (which
parses markup), not for ``DataTable`` cells.
"""

from __future__ import annotations

from ..data.models import RoundLine

ACCENT = "#e87a5d"   # team 1 / attacker-ish
BLUE = "#5d9ce8"     # team 2
TRACK = "#22384a"    # empty track

FULL = "█"
EMPTY = "░"


def winbar(a: int | None, b: int | None, width: int = 18) -> str:
    """Proportional round-share bar; left (accent) is team A's share."""
    a, b = a or 0, b or 0
    total = a + b
    if total == 0:
        return f"[{TRACK}]{EMPTY * width}[/]"
    fill = round((a / total) * width)
    fill = max(0, min(width, fill))
    return f"[{ACCENT}]{FULL * fill}[/][{TRACK}]{EMPTY * (width - fill)}[/]"


def momentum(rounds: list[RoundLine], team1_short: str | None) -> str:
    """Round-by-round momentum: ▲ attacker win, ▼ defender win; colour = team."""
    if not rounds:
        return ""
    parts = []
    for r in rounds:
        glyph = "▲" if r.is_attack else "▼"
        colour = ACCENT if r.winner_short == team1_short else BLUE
        parts.append(f"[{colour}]{glyph}[/]")
    return "".join(parts)
