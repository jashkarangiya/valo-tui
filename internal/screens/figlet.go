package screens

import "strings"

// bigDigits is the figlet ansi_shadow digit set, matching the splash banner
// font so the score hero is visually consistent with the branding.
var bigDigits = map[rune][]string{
	'0': {" ██████╗ ", "██╔═████╗", "██║██╔██║", "████╔╝██║", "╚██████╔╝", " ╚═════╝ "},
	'1': {" ██╗", "███║", "╚██║", " ██║", " ██║", " ╚═╝"},
	'2': {"██████╗ ", "╚════██╗", " █████╔╝", "██╔═══╝ ", "███████╗", "╚══════╝"},
	'3': {"██████╗ ", "╚════██╗", " █████╔╝", " ╚═══██╗", "██████╔╝", "╚═════╝ "},
	'4': {"██╗  ██╗", "██║  ██║", "███████║", "╚════██║", "     ██║", "     ╚═╝"},
	'5': {"███████╗", "██╔════╝", "███████╗", "╚════██║", "███████║", "╚══════╝"},
	'6': {" ██████╗ ", "██╔════╝ ", "███████╗ ", "██╔═══██╗", "╚██████╔╝", " ╚═════╝ "},
	'7': {"███████╗", "╚════██║", "    ██╔╝", "   ██╔╝ ", "   ██║  ", "   ╚═╝  "},
	'8': {" █████╗ ", "██╔══██╗", "╚█████╔╝", "██╔══██╗", "╚█████╔╝", " ╚════╝ "},
	'9': {" █████╗ ", "██╔══██╗", "╚██████║", " ╚═══██║", " █████╔╝", " ╚════╝ "},
	'-': {"      ", "      ", "█████╗", "╚════╝", "      ", "      "},
}

const bigHeight = 6

// big renders a short numeric string as 6-row block art.
func big(text string) string {
	rows := make([]string, bigHeight)
	for _, r := range text {
		glyph, ok := bigDigits[r]
		if !ok {
			glyph = bigDigits['0']
		}
		for i := 0; i < bigHeight; i++ {
			if rows[i] != "" {
				rows[i] += " "
			}
			rows[i] += glyph[i]
		}
	}
	return strings.Join(rows, "\n")
}
