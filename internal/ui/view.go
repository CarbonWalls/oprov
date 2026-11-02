package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// View renders the full screen.
func (a *app) View() string {
	if a.width == 0 {
		return "loading…"
	}
	base := a.frame()

	switch {
	case a.mode == modeConfirm && a.confirm != nil:
		return placeBox(base, a.confirm.view(min(a.width-8, 64)), a.width, a.height)
	case a.showHelp:
		return placeBox(base, helpOverlay(min(a.width-8, 72)), a.width, a.height)
	case a.mode == modeForm && a.form != nil:
		return placeBox(base, a.formView(), a.width, a.height)
	}
	return base
}

// frame renders the underlying application without overlays.
func (a *app) frame() string {
	var b strings.Builder

	// Top bar.
	b.WriteString(a.renderTopBar())
	b.WriteString("\n")

	// Main panes.
	mainH := a.height - 3
	if mainH < 1 {
		mainH = 1
	}
	listW := a.width * 36 / 100
	if listW < 26 {
		listW = 26
	}
	if listW > 48 {
		listW = 48
	}
	if listW > a.width/2 {
		listW = a.width / 2
	}
	detailW := a.width - listW
	if detailW < 1 {
		detailW = 1
	}

	list := a.renderList(listW, mainH)
	detail := a.renderDetail(detailW, mainH)
	joined := lipgloss.JoinHorizontal(lipgloss.Top, list, detail)
	lines := strings.Split(joined, "\n")
	for len(lines) < mainH {
		lines = append(lines, strings.Repeat(" ", a.width))
	}
	if len(lines) > mainH {
		lines = lines[:mainH]
	}
	b.WriteString(strings.Join(lines, "\n"))

	// Status bar + help footer.
	b.WriteString("\n")
	b.WriteString(a.renderStatus())
	b.WriteString("\n")
	b.WriteString(a.renderHelp())
	return b.String()
}

func (a *app) renderTopBar() string {
	left := inter("●").Render("●") + " " +
		txt("oprov").Render("oprov") +
		mute(" · opencode providers").Render(" · opencode providers")

	right := mute(shortHome(a.store.Path)).Render(shortHome(a.store.Path))
	if strings.HasSuffix(a.store.Path, ".jsonc") {
		right += "  " + faint("jsonc").Render("jsonc")
	}

	pad := a.width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return styleTopBar.Width(a.width).Render(left + strings.Repeat(" ", pad) + right)
}

func (a *app) renderStatus() string {
	msg := a.status.msg
	if a.testing {
		msg = fmt.Sprintf("testing %s …", a.testID)
	}
	if msg == "" {
		msg = fmt.Sprintf("%d provider(s) · %s", len(a.store.ProviderIDs()), shortHome(a.store.Path))
	}
	var styled string
	if a.testing {
		styled = inter(a.spinner.View()).Render(a.spinner.View()) + " " + mute(msg).Render(msg)
	} else {
		switch a.status.kind {
		case statusOK:
			styled = okT(msg).Render(msg)
		case statusErr:
			styled = errT(msg).Render(msg)
		case statusWarn:
			styled = warn(msg).Render(msg)
		default:
			styled = mute(msg).Render(msg)
		}
	}

	right := faint("? help · q quit").Render("? help · q quit")
	// styleStatus has 2-cell side padding; content area is width-4.
	avail := a.width - 4
	pad := avail - lipgloss.Width(styled) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return styleStatus.Width(a.width).Render(styled + strings.Repeat(" ", pad) + right)
}

// renderHelp draws the one-line keybinding footer.
func (a *app) renderHelp() string {
	var parts []string
	if a.focus == focusList {
		parts = []string{"a", "add", "e", "edit", "d", "del", "c", "key", "x", "clear", "t", "test", "tab", "detail", "q", "quit"}
	} else {
		parts = []string{"a", "model", "e", "edit", "d", "del", "c", "key", "x", "clear", "t", "test", "esc", "list", "q", "quit"}
	}
	var b strings.Builder
	for i := 0; i+1 < len(parts); i += 2 {
		if i > 0 {
			b.WriteString(strings.Repeat(" ", 2))
		}
		b.WriteString(keyCap(parts[i]).Render("[" + parts[i] + "]"))
		b.WriteString(faint(" " + parts[i+1]).Render(" " + parts[i+1]))
	}
	hint := b.String()
	if lipgloss.Width(hint) > a.width-4 {
		hint = lipgloss.NewStyle().MaxWidth(a.width - 4).Render(hint)
	}
	return styleHelp.Width(a.width).Render(hint)
}

// scrimColor is the flat background drawn behind a centered modal: darker
// than the modal panel so the box visibly rises, and completely uniform so
// no underlying UI content or borders leak through ("black spots").
const scrimColor = "#0b0b0b"

// placeBox draws a centered, opaque box onto a full-screen scrim. The frame
// content is intentionally not shown while a modal is open; lines are padded
// with ANSI-safe widths so escape sequences are never broken.
func placeBox(_ string, box string, screenW, screenH int) string {
	boxLines := strings.Split(strings.TrimRight(box, "\n"), "\n")
	boxW := 0
	for _, l := range boxLines {
		if w := ansi.StringWidth(l); w > boxW {
			boxW = w
		}
	}
	boxH := len(boxLines)
	if boxW > screenW {
		boxW = screenW
	}
	if boxH > screenH {
		boxH = screenH
		boxLines = boxLines[:screenH]
	}

	x := (screenW - boxW) / 2
	if x < 0 {
		x = 0
	}
	y := (screenH - boxH) / 2
	if y < 0 {
		y = 0
	}

	// Pad box lines shorter than boxW with the box background so the modal
	// stays an opaque rectangle.
	padStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgPanel))
	for i, l := range boxLines {
		if w := ansi.StringWidth(l); w < boxW {
			boxLines[i] = l + padStyle.Render(strings.Repeat(" ", boxW-w))
		}
	}

	scrim := lipgloss.NewStyle().Background(lipgloss.Color(scrimColor))
	out := make([]string, screenH)
	for i := 0; i < screenH; i++ {
		if i >= y && i < y+boxH {
			j := i - y
			line := scrim.Render(strings.Repeat(" ", x)) + boxLines[j]
			w := ansi.StringWidth(line)
			if w < screenW {
				line += scrim.Render(strings.Repeat(" ", screenW-w))
			}
			out[i] = line
			continue
		}
		out[i] = scrim.Render(strings.Repeat(" ", screenW))
	}
	return strings.Join(out, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// boxStyle is the shared modal container.
func boxStyle(w int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(border)).
		Background(lipgloss.Color(bgPanel)).
		Foreground(lipgloss.Color(textBase)).
		Padding(0, 2).
		Width(w)
}
