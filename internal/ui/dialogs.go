package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/oprov/oprov/internal/config"
)

// confirmDlg is a lightweight modal confirm with y/n keys.
type confirmDlg struct {
	title       string
	message     string
	okLabel     string
	cancelLabel string
	danger      bool
	onOK        func(*app)
}

func (a *app) startConfirm(dlg *confirmDlg) tea.Cmd {
	a.confirm = dlg
	a.mode = modeConfirm
	return nil
}

func (a *app) updateConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter", " ":
		dlg := a.confirm
		a.confirm = nil
		a.mode = modeMain
		if dlg != nil && dlg.onOK != nil {
			dlg.onOK(a)
		}
	case "n", "N", "esc", "ctrl+c":
		a.confirm = nil
		a.mode = modeMain
	}
	return a, nil
}

func (d *confirmDlg) view(width int) string {
	if width < 40 {
		width = 40
	}
	inner := strings.Builder{}
	title := accent(d.title).Render(d.title)
	if d.danger {
		title = errT(d.title).Render(d.title)
	}
	inner.WriteString(title + "\n\n")
	for _, line := range strings.Split(d.message, "\n") {
		inner.WriteString(mute(line).Render(line) + "\n")
	}
	inner.WriteString("\n")

	okLabel := d.okLabel
	if okLabel == "" {
		okLabel = "confirm"
	}
	if d.danger {
		inner.WriteString(errT("[y]").Render("[y]") + " " + errT(okLabel).Render(okLabel))
	} else {
		inner.WriteString(inter("[y]").Render("[y]") + " " + txt(okLabel).Render(okLabel))
	}
	cancelLabel := d.cancelLabel
	if cancelLabel == "" {
		cancelLabel = "cancel"
	}
	inner.WriteString("      " + keyCap("[n]").Render("[n]") + " " + mute(cancelLabel).Render(cancelLabel))

	return boxStyle(width).Render(inner.String())
}

// --- concrete confirm dialogs ---------------------------------------------

func delProviderConfirm(a *app) *confirmDlg {
	id := a.providerID()
	p := a.store.ReadProvider(id)
	name := p.Name
	if name == "" {
		name = id
	}
	msg := fmt.Sprintf("Remove provider %q from the config?\nThis removes %d model(s) from the file.", name, len(p.Models))
	return &confirmDlg{
		title:   "Delete provider",
		message: msg,
		okLabel: "delete",
		danger:  true,
		onOK: func(a *app) {
			if a.store.DeleteProvider(id) && a.saveOK("deleted provider "+id) {
				if a.sel > 0 {
					a.sel--
				}
				a.msel = 0
			}
		},
	}
}

func delModelConfirm(a *app) *confirmDlg {
	pid := a.providerID()
	mids := a.modelIDs()
	if len(mids) == 0 {
		return nil
	}
	mid := mids[a.msel]
	m := a.store.ReadModel(pid, mid)
	name := m.Name
	if name == "" {
		name = mid
	}
	msg := fmt.Sprintf("Remove model %q from provider %q?", name, pid)
	return &confirmDlg{
		title:   "Delete model",
		message: msg,
		okLabel: "delete",
		danger:  true,
		onOK: func(a *app) {
			if a.store.DeleteModel(pid, mid) && a.saveOK("deleted model "+mid) {
				if a.msel > 0 {
					a.msel--
				}
			}
		},
	}
}

func clearKeyConfirm(a *app) *confirmDlg {
	pid := a.providerID()
	p := a.store.ReadProvider(pid)
	info := config.KeyInfoOf(&p)
	msg := fmt.Sprintf("Remove the API key for %q?\n", pid)
	if info.Path != "" {
		msg += fmt.Sprintf("The key file at %s will be deleted too.", shortHome(info.Path))
	} else {
		msg += "The apiKey setting will be removed from the config."
	}
	return &confirmDlg{
		title:   "Clear API key",
		message: msg,
		okLabel: "clear",
		danger:  true,
		onOK: func(a *app) {
			if info.Path != "" {
				config.RemoveKeyFile(info.Path)
			}
			if err := a.store.SetProviderAPIKey(pid, ""); err != nil {
				a.status.set("error: "+err.Error(), statusErr)
				return
			}
			a.saveOK("removed API key for " + pid)
		},
	}
}

// saveOK persists the document and reports the outcome.
func (a *app) saveOK(msg string) bool {
	if err := a.store.Save(); err != nil {
		a.status.set("save failed: "+err.Error(), statusErr)
		return false
	}
	a.status.set(msg, statusOK)
	return true
}

func helpOverlay(width int) string {
	var b strings.Builder
	b.WriteString(accent("KEYBINDS").Render("KEYBINDS") + "\n\n")
	groups := [][]string{
		{"↑/↓ · j/k", "move selection"},
		{"g / G", "first / last"},
		{"a", "add provider (list) · add model (detail)"},
		{"e · enter", "edit provider · edit model"},
		{"d", "delete provider · delete model"},
		{"c", "set / change API key"},
		{"x", "clear API key"},
		{"t", "test connection (GET /models)"},
		{"tab · enter", "focus detail pane"},
		{"esc · h", "back to provider list"},
		{"?", "toggle this help"},
		{"q · ctrl+c", "quit"},
	}
	for _, g := range groups {
		k := inter(g[0]).Render(g[0])
		d := mute(g[1]).Render(g[1])
		pad := width - 6 - lipgloss.Width(k) - lipgloss.Width(d)
		if pad < 2 {
			pad = 2
		}
		b.WriteString(k + strings.Repeat(" ", pad) + d + "\n")
	}
	return boxStyle(width).Render(b.String())
}
