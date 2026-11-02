package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Style tokens lifted from opencode v2.2.5's default "opencode" TUI theme
// (packages/tui/src/theme/assets/v2/opencode.json, dark mode).

const (
	bgBase   = "#0a0a0a" // neutral.800
	bgPanel  = "#141414" // neutral.700 (raised base)
	bgHigh   = "#1e1e1e" // neutral.600 (raised high)
	border   = "#484848"
	borderD  = "#2c2c2c"
	textBase = "#eeeeee" // neutral.200
	textMute = "#808080" // neutral.400
	textFain = "#4c4c4c" // neutral.500

	accentAm   = "#9d7cd8" // accent hue (purple) .200
	interactAm = "#fab283" // interactive hue (orange) .200
	warnAm     = "#f5a742"
	errAm      = "#e06c75" // red.200
	okAm       = "#7fd88f" // green.200
)

var (
	styleBase = lipgloss.NewStyle().Background(lipgloss.Color(bgBase))

	styleTopBar = lipgloss.NewStyle().
			Background(lipgloss.Color(bgBase)).
			Foreground(lipgloss.Color(textBase))

	stylePane = func(active bool) lipgloss.Style {
		s := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(borderD)).
			Padding(0, 1)
		if active {
			s = s.BorderForeground(lipgloss.Color(border))
		}
		return s
	}

	stylePaneActive   = stylePane(true)
	stylePaneInactive = stylePane(false)

	styleStatus = lipgloss.NewStyle().
			Background(lipgloss.Color(bgPanel)).
			Foreground(lipgloss.Color(textMute)).
			Padding(0, 2)

	styleHelp = lipgloss.NewStyle().
			Background(lipgloss.Color(bgBase)).
			Foreground(lipgloss.Color(textFain)).
			Padding(0, 2)
)

// --- text helpers ----------------------------------------------------------

func txt(s string) lipgloss.Style    { return lipgloss.NewStyle().Foreground(lipgloss.Color(textBase)) }
func mute(s string) lipgloss.Style   { return lipgloss.NewStyle().Foreground(lipgloss.Color(textMute)) }
func faint(s string) lipgloss.Style  { return lipgloss.NewStyle().Foreground(lipgloss.Color(textFain)) }
func accent(s string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(accentAm)) }
func inter(s string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(interactAm))
}
func warn(s string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(warnAm)) }
func errT(s string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(errAm)) }
func okT(s string) lipgloss.Style  { return lipgloss.NewStyle().Foreground(lipgloss.Color(okAm)) }

// key style: the hint key of a keybinding, opencode-style (dim key, label).
func keyCap(k string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(textMute))
}

// openCodeFormTheme returns a huh theme that matches opencode's default
// "opencode" dark theme: near-black background, amber/orange for focused
// interactive elements, purple for emphasis, #484848 borders.
func openCodeFormTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Focused field: title + border in the interactive (orange) accent.
	t.Focused.Base = t.Focused.Base.BorderForeground(lipgloss.Color(interactAm))
	t.Focused.Title = t.Focused.Title.Foreground(lipgloss.Color(textBase)).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(lipgloss.Color(textMute))
	t.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color(errAm))
	t.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(lipgloss.Color(errAm))
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(lipgloss.Color(interactAm))
	t.Focused.Option = t.Focused.Option.Foreground(lipgloss.Color(textBase))
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(lipgloss.Color(interactAm))
	t.Focused.TextInput.CursorText = t.Focused.TextInput.CursorText.
		Foreground(lipgloss.Color(bgBase)).
		Background(lipgloss.Color(interactAm))
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(lipgloss.Color(textFain))
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(lipgloss.Color(interactAm))
	t.Focused.TextInput.Text = t.Focused.TextInput.Text.Foreground(lipgloss.Color(textBase))
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(lipgloss.Color(okAm))
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(lipgloss.Color(textFain))
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(lipgloss.Color(interactAm))
	t.Focused.FocusedButton = t.Focused.FocusedButton.
		Foreground(lipgloss.Color(bgBase)).
		Background(lipgloss.Color(interactAm))
	t.Focused.BlurredButton = t.Focused.BlurredButton.
		Foreground(lipgloss.Color(textMute)).
		Background(lipgloss.Color(bgHigh))
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(lipgloss.Color(textMute))
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(lipgloss.Color(textMute))

	// Blurred field: muted.
	t.Blurred.Base = t.Blurred.Base.BorderForeground(lipgloss.Color(border))
	t.Blurred.Title = t.Blurred.Title.Foreground(lipgloss.Color(textMute))
	t.Blurred.Description = t.Blurred.Description.Foreground(lipgloss.Color(textFain))
	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(lipgloss.Color(textFain))
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(lipgloss.Color(textMute))
	t.Blurred.TextInput.Cursor = t.Blurred.TextInput.Cursor.Foreground(lipgloss.Color(textFain))

	// Group styling.
	t.Group.Title = t.Group.Title.Foreground(lipgloss.Color(accentAm)).Bold(true)
	t.Group.Description = t.Group.Description.Foreground(lipgloss.Color(textFain))

	// Help (bottom of form): kebab hints in the muted/faint style.
	t.Help.ShortKey = t.Help.ShortKey.Foreground(lipgloss.Color(textFain))
	t.Help.ShortDesc = t.Help.ShortDesc.Foreground(lipgloss.Color(textFain))
	t.Help.FullKey = t.Help.FullKey.Foreground(lipgloss.Color(textFain))
	t.Help.FullDesc = t.Help.FullDesc.Foreground(lipgloss.Color(textMute))
	return t
}
