package screens

// Messages screens emit for the root app to act on. Keeping them in the leaf
// screens package (which the app imports) avoids an import cycle.

// EnterAppMsg asks the root to dismiss the splash and enter the shell.
type EnterAppMsg struct{}

// SwitchRouteMsg asks the root to show a different route.
type SwitchRouteMsg struct{ To string }

// SelectEventMsg focuses the app on an event and opens one of its sub-pages.
type SelectEventMsg struct {
	ID  int
	Tab string
}

// CloseOverlayMsg dismisses the match-detail overlay and returns to the shell.
type CloseOverlayMsg struct{}

// OpenBracketMsg asks the root to focus the named event and open its bracket
// (emitted from the match-detail [b] binding).
type OpenBracketMsg struct{ EventName string }
