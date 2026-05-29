"""Single-character semantic icons for maps and agent roles.

Deliberately tiny — one glyph + colour, never giant art. The role colours
match the palette accents so a scoreboard reads at a glance.
"""

from __future__ import annotations

# ── Maps ─────────────────────────────────────────────────────
MAP_ICONS: dict[str, str] = {
    "Ascent": "◆",
    "Bind": "◈",
    "Haven": "▲",
    "Lotus": "❀",
    "Sunset": "☀",
    "Split": "║",
    "Icebox": "❄",
    "Pearl": "○",
    "Breeze": "≈",
    "Fracture": "⚡",
    "Abyss": "▽",
    "Corrode": "◙",
}


def map_icon(name: str | None) -> str:
    return MAP_ICONS.get((name or "").strip().title(), "·")


# ── Agent roles ──────────────────────────────────────────────
# role -> (glyph, colour)
ROLE_GLYPH: dict[str, tuple[str, str]] = {
    "duelist": ("▲", "#e0594a"),
    "controller": ("◆", "#5d9ce8"),
    "initiator": ("◈", "#e8c15d"),
    "sentinel": ("●", "#5de88a"),
}

_AGENT_ROLE: dict[str, str] = {}
for _role, _agents in {
    "duelist": "jett raze phoenix reyna yoru neon iso waylay",
    "controller": "brimstone omen viper astra harbor clove",
    "initiator": "sova breach skye kayo fade gekko tejo",
    "sentinel": "killjoy cypher sage chamber deadlock vyse",
}.items():
    for _a in _agents.split():
        _AGENT_ROLE[_a] = _role


def agent_role(name: str | None) -> str | None:
    return _AGENT_ROLE.get((name or "").strip().lower())


def agent_glyph(name: str | None) -> tuple[str, str]:
    """Return ``(glyph, colour)`` for an agent, defaulting to a muted dot."""
    return ROLE_GLYPH.get(agent_role(name) or "", ("·", "#4a708b"))
