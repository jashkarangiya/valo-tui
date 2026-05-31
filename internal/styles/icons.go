package styles

import (
	"image/color"
	"strings"
)

// Single-character semantic icons for maps and agent roles, ported from
// style/icons.py. Deliberately tiny — one glyph + colour.

var mapIcons = map[string]string{
	"Ascent": "◆", "Bind": "◈", "Haven": "▲", "Lotus": "❀",
	"Sunset": "☀", "Split": "║", "Icebox": "❄", "Pearl": "○",
	"Breeze": "≈", "Fracture": "⚡", "Abyss": "▽", "Corrode": "◙",
}

// MapIcon returns the glyph for a map name, or "·" if unknown.
func MapIcon(name string) string {
	if g, ok := mapIcons[titleCase(strings.TrimSpace(name))]; ok {
		return g
	}
	return "·"
}

// titleCase upper-cases the first rune and lower-cases the rest (map names are
// single words), matching Python str.title() for this use.
func titleCase(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(strings.ToLower(s))
	r[0] = []rune(strings.ToUpper(string(r[0])))[0]
	return string(r)
}

type roleStyle struct {
	glyph  string
	colour color.Color
}

var roleGlyph = map[string]roleStyle{
	"duelist":    {"▲", RoleDuelist},
	"controller": {"◆", RoleController},
	"initiator":  {"◈", RoleInitiator},
	"sentinel":   {"●", RoleSentinel},
}

var agentRole = func() map[string]string {
	m := map[string]string{}
	for role, agents := range map[string]string{
		"duelist":    "jett raze phoenix reyna yoru neon iso waylay",
		"controller": "brimstone omen viper astra harbor clove",
		"initiator":  "sova breach skye kayo fade gekko tejo",
		"sentinel":   "killjoy cypher sage chamber deadlock vyse",
	} {
		for _, a := range strings.Fields(agents) {
			m[a] = role
		}
	}
	return m
}()

// AgentRole returns an agent's role, or "" if unknown.
func AgentRole(name string) string {
	return agentRole[strings.ToLower(strings.TrimSpace(name))]
}

// AgentGlyph returns (glyph, colour) for an agent, defaulting to a muted dot.
func AgentGlyph(name string) (string, color.Color) {
	if rs, ok := roleGlyph[AgentRole(name)]; ok {
		return rs.glyph, rs.colour
	}
	return "·", Muted
}
