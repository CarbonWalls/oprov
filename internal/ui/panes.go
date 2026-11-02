package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/oprov/oprov/internal/config"
)

// renderList builds the left provider pane.
func (a *app) renderList(w, h int) string {
	active := a.focus == focusList
	ids := a.store.ProviderIDs()

	inner := strings.Builder{}

	// Header.
	title := "PROVIDERS"
	if active {
		title = inter(title).Render(title)
	} else {
		title = mute(title).Render(title)
	}
	count := faint(fmt.Sprintf("%d", len(ids))).Render(fmt.Sprintf("%d", len(ids)))
	contentW := w - 4
	head := title + strings.Repeat(" ", max(0, contentW-lipgloss.Width(title)-lipgloss.Width(count))) + count
	inner.WriteString(head + "\n")
	inner.WriteString(faint(strings.Repeat("─", contentW)).Render(strings.Repeat("─", contentW)) + "\n")

	bodyH := h - 4
	if bodyH < 1 {
		bodyH = 1
	}

	if len(ids) == 0 {
		empty := strings.Builder{}
		empty.WriteString("\n")
		empty.WriteString(center("no custom providers", contentW))
		empty.WriteString("\n")
		empty.WriteString(center("press a to add one", contentW))
		inner.WriteString(empty.String())
	} else {
		// selection-scoped scroll window
		start := a.sel - bodyH/2
		if start < 0 {
			start = 0
		}
		if start+bodyH > len(ids) {
			start = max(0, len(ids)-bodyH)
		}
		for i := start; i < len(ids) && i < start+bodyH; i++ {
			inner.WriteString(a.listItem(ids[i], i, contentW))
			if i < start+bodyH-1 && i < len(ids)-1 {
				inner.WriteString("\n")
			}
		}
	}

	style := stylePaneInactive
	if active {
		style = stylePaneActive
	}
	return style.Width(w).Height(h).Render(inner.String())
}

// listItem renders one provider row (two lines).
func (a *app) listItem(id string, idx, contentW int) string {
	p := a.store.ReadProvider(id)
	selected := idx == a.sel

	keyInfo := config.KeyInfoOf(&p)
	dot, dotColor := keyDot(keyInfo)

	models := len(p.Models)

	// line 1: name + model count
	name := p.Name
	if name == "" {
		name = id
	}
	count := ""
	if models > 0 {
		count = fmt.Sprintf("%d model(s)", models)
	}

	var l1 strings.Builder
	if selected {
		l1.WriteString(inter("❯").Render("❯") + " ")
	} else {
		l1.WriteString(strings.Repeat(" ", 2))
	}
	nameS := txt(name).Render(truncate(name, contentW-8))
	if selected {
		nameS = lipgloss.NewStyle().Foreground(lipgloss.Color(textBase)).Bold(true).Render(truncate(name, contentW-8))
	} else {
		nameS = mute(name).Render(truncate(name, contentW-8))
	}
	l1.WriteString(nameS)
	l1.WriteString(strings.Repeat(" ", max(0, contentW-2-lipgloss.Width(nameS)-lipgloss.Width(count))))
	l1.WriteString(faint(count).Render(count))

	// line 2: id + key status
	var l2 strings.Builder
	l2.WriteString(strings.Repeat(" ", 2))
	l2.WriteString(faint(id).Render(truncate(id, contentW-8)))
	dotTxt := "  " + dotColor.Render(dot)
	l2.WriteString(strings.Repeat(" ", max(0, contentW-2-lipgloss.Width(id)-lipgloss.Width(dotTxt))))
	l2.WriteString(dotTxt)

	return l1.String() + "\n" + l2.String()
}

// keyDot returns the status glyph and its colour for a provider key.
func keyDot(info config.KeyInfo) (string, lipgloss.Style) {
	switch info.Status {
	case config.KeySet:
		return "●", okT("")
	case config.KeyUnset:
		return "○", warn("")
	default:
		return "·", faint("")
	}
}

// renderDetail builds the right detail pane.
func (a *app) renderDetail(w, h int) string {
	active := a.focus == focusDetail
	style := stylePaneInactive
	if active {
		style = stylePaneActive
	}
	id := a.providerID()
	if id == "" {
		inner := strings.Builder{}
		inner.WriteString(accent("DETAIL").Render("DETAIL") + "\n")
		inner.WriteString(faint(strings.Repeat("─", w-4)).Render(strings.Repeat("─", w-4)) + "\n")
		inner.WriteString("\n")
		inner.WriteString(center("select a provider", w-4))
		return style.Width(w).Height(h).Render(inner.String())
	}
	p := a.store.ReadProvider(id)

	inner := strings.Builder{}
	title := p.Name
	if title == "" {
		title = id
	}
	if active {
		inner.WriteString(inter(title).Render(title) + "\n")
	} else {
		inner.WriteString(mute(title).Render(title) + "\n")
	}
	inner.WriteString(faint(strings.Repeat("─", w-4)).Render(strings.Repeat("─", w-4)) + "\n")

	rows := detailRows(&p, id, w-4)
	for _, r := range rows {
		inner.WriteString(r + "\n")
	}

	// Models section.
	modelIDs := a.store.ModelIDs(id)
	inner.WriteString("\n")
	modelsHead := "MODELS"
	if active {
		modelsHead = inter("MODELS").Render(modelsHead)
	} else {
		modelsHead = accent("MODELS").Render(modelsHead)
	}
	cnt := faint(fmt.Sprintf("%d", len(modelIDs))).Render(fmt.Sprintf("%d", len(modelIDs)))
	inner.WriteString(modelsHead + "  " + cnt + "\n")

	if len(modelIDs) == 0 {
		inner.WriteString(mute("no models · press a to add").Render("no models · press a to add") + "\n")
	} else {
		for i, mid := range modelIDs {
			inner.WriteString(a.modelItem(id, mid, i, w-4, active))
			if i < len(modelIDs)-1 {
				inner.WriteString("\n")
			}
		}
	}

	content := inner.String()
	// Scroll so the selected model stays visible when the detail pane is focused.
	if active {
		content = scrollToModel(content, a.msel, h-4)
	}
	return style.Width(w).Height(h).Render(content)
}

func detailRows(p *config.ProviderConfig, id string, width int) []string {
	var rows []string
	baseURL, _ := p.Settings["baseURL"].(string)
	info := config.KeyInfoOf(p)
	dot, dotColor := keyDot(info)

	rows = append(rows, rowLine("id", id, width))
	if p.Name != "" {
		rows = append(rows, rowLine("name", p.Name, width))
	}
	if p.Package != "" {
		rows = append(rows, rowLine("package", p.Package, width))
	}
	if baseURL != "" {
		rows = append(rows, rowLine("base url", baseURL, width))
	}

	// key row
	keyTxt := dotColor.Render(dot) + " "
	switch info.Status {
	case config.KeySet:
		keyTxt += okT(info.Source).Render(info.Source)
	case config.KeyUnset:
		keyTxt += warn(info.Source + " (not set)").Render(info.Source + "  (unset)")
	default:
		keyTxt += faint("no key · press c to add").Render("no key · press c to add")
	}
	rows = append(rows, rowLine("api key", keyTxt, width))

	if len(p.Env) > 0 {
		rows = append(rows, rowLine("env", strings.Join(p.Env, ", "), width))
	}
	if len(p.Headers) > 0 {
		rows = append(rows, rowLine("headers", fmt.Sprintf("%d header(s)", len(p.Headers)), width))
	}
	return rows
}

func rowLine(label, value string, width int) string {
	lbl := faint(label).Render(padTo(label, 10))
	v := txt(value).Render(truncate(value, width-12))
	return lbl + v
}

func padTo(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// modelItem renders one model row inside the detail pane.
func (a *app) modelItem(pid, mid string, idx, width int, active bool) string {
	m := a.store.ReadModel(pid, mid)
	selected := active && idx == a.msel

	name := m.Name
	if name == "" {
		name = mid
	}
	var b strings.Builder
	if selected {
		b.WriteString(inter("❯").Render("❯") + " ")
	} else {
		b.WriteString(strings.Repeat(" ", 2))
	}
	var nameS string
	if selected {
		nameS = lipgloss.NewStyle().Foreground(lipgloss.Color(textBase)).Bold(true).Render(truncate(name, width-16))
	} else {
		nameS = txt(name).Render(truncate(name, width-16))
	}
	b.WriteString(nameS)
	// context + cost summary right-aligned
	right := modelMeta(m)
	b.WriteString(strings.Repeat(" ", max(0, width-2-lipgloss.Width(nameS)-lipgloss.Width(right))))
	b.WriteString(faint(right).Render(right))
	return b.String()
}

func modelMeta(m config.ModelConfig) string {
	parts := []string{}
	if m.Limit != nil && m.Limit.Context != nil {
		parts = append(parts, humanNum(*m.Limit.Context))
	}
	if m.Cost != nil && m.Cost.Input != nil {
		parts = append(parts, fmt.Sprintf("$%.2f/M", *m.Cost.Input))
	}
	return strings.Join(parts, " · ")
}

func humanNum(n int) string {
	switch {
	case n >= 1_000_000:
		if n%1_000_000 == 0 {
			return fmt.Sprintf("%dM", n/1_000_000)
		}
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		if n%1_000 == 0 {
			return fmt.Sprintf("%dK", n/1_000)
		}
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// scrollToModel trims the detail content so the selected model row (relative
// to the start of the MODELS section) stays visible within h lines.
func scrollToModel(content string, modelIndex, h int) string {
	lines := strings.Split(content, "\n")
	total := len(lines)
	if total <= h {
		return content
	}
	start := -1
	for i, l := range lines {
		if strings.Contains(l, "MODELS") {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return content
	}
	target := start + modelIndex
	if target < start {
		target = start
	}
	top := 0
	if target >= h {
		top = target - h + 1
	}
	if top+h > total {
		top = total - h
	}
	if top < 0 {
		top = 0
	}
	return strings.Join(lines[top:top+h], "\n")
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w <= n {
		return s
	}
	// trim by cells
	out := s
	for lipgloss.Width(out) > n {
		r := []rune(out)
		if len(r) == 0 {
			break
		}
		out = string(r[:len(r)-1])
	}
	if lipgloss.Width(out) > n-1 {
		out = out[:len(out)-1]
	}
	return out + "…"
}

func center(text string, w int) string {
	if w < 1 {
		w = 1
	}
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(text)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
