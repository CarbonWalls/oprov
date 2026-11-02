package ui

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/oprov/oprov/internal/config"
)

var idRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

const defaultPackage = "@opencode/ai/providers/openai-compatible"

// formKeyMap returns a keymap with a reliable "go to previous field" key.
// Terminals differ in whether they emit a distinct sequence for shift+tab
// (many send plain tab), so text inputs and confirmations also accept ↑ and
// dropdowns accept ← to move up. These keys are not used by any field kind.
func formKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	rebind := func(b key.Binding, keys ...string) key.Binding {
		return key.NewBinding(key.WithKeys(keys...), key.WithHelp(b.Help().Key, b.Help().Desc))
	}
	km.Input.Prev = rebind(km.Input.Prev, "shift+tab", "up")
	km.Text.Prev = rebind(km.Text.Prev, "shift+tab")
	km.Select.Prev = rebind(km.Select.Prev, "shift+tab", "left")
	km.Confirm.Prev = rebind(km.Confirm.Prev, "shift+tab", "up")
	return km
}

// --- provider form ---------------------------------------------------------

func (a *app) startProviderForm(editing bool) tea.Cmd {
	var id string
	var p config.ProviderConfig
	if editing {
		id = a.providerID()
		if id == "" {
			return nil
		}
		a.formPrevID = id
		p = a.store.ReadProvider(id)
	}
	a.formKind = formProvider
	a.formAdd = !editing

	nameVal := p.Name
	pkgVal := p.Package
	if pkgVal == "" {
		pkgVal = defaultPackage
	}
	urlVal := ""
	if u, ok := p.Settings["baseURL"].(string); ok {
		urlVal = u
	}
	envVal := strings.Join(p.Env, ", ")

	fields := []huh.Field{}
	if !editing {
		existing := a.store.ProviderIDs()
		fields = append(fields, huh.NewInput().
			Key("id").
			Title("Provider ID").
			Description("id used in references, e.g. acme/qwen3-coder").
			Prompt("> ").
			Validate(func(v string) error {
				v = strings.TrimSpace(v)
				if v == "" {
					return errors.New("provider id is required")
				}
				if !idRe.MatchString(v) {
					return errors.New("letters, digits, '.', '-', '_' only, no '/'")
				}
				for _, e := range existing {
					if e == v {
						return errors.New("provider with this id already exists")
					}
				}
				return nil
			}))
	}
	fields = append(fields,
		huh.NewInput().Key("name").Title("Display name").Description("optional, shown in opencode").Prompt("> ").Value(&nameVal),
		huh.NewInput().Key("package").Title("Runtime package").Description("npm package, e.g. "+defaultPackage).Prompt("> ").Value(&pkgVal),
		huh.NewInput().Key("baseURL").Title("Base URL").Description("API endpoint, e.g. https://llm.acme.example/v1").Prompt("> ").Value(&urlVal),
		huh.NewInput().Key("env").Title("Env vars").Description("comma-separated credential env var names (optional)").Prompt("> ").Value(&envVal),
	)

	a.form = huh.NewForm(huh.NewGroup(fields...)).
		WithTheme(openCodeFormTheme()).
		WithKeyMap(formKeyMap()).
		WithWidth(min(a.width-12, 64)).
		WithHeight(min(a.height-12, 20)).
		WithShowHelp(false)
	a.mode = modeForm
	return a.form.Init()
}

func (a *app) finishProviderForm() {
	f := a.form
	if a.formAdd {
		id := strings.TrimSpace(f.GetString("id"))
		pc := config.ProviderConfig{
			Name:    strings.TrimSpace(f.GetString("name")),
			Package: strings.TrimSpace(f.GetString("package")),
			Env:     splitVars(f.GetString("env")),
		}
		if u := strings.TrimSpace(f.GetString("baseURL")); u != "" {
			pc.Settings = map[string]any{"baseURL": u}
		}
		if err := a.store.AddProvider(id, pc); err != nil {
			a.toastErr(err)
			return
		}
		if !a.saveOK("added provider " + id) {
			return
		}
		ids := a.store.ProviderIDs()
		for i, e := range ids {
			if e == id {
				a.sel = i
				break
			}
		}
		a.msel = 0
		return
	}

	id := a.formPrevID
	var err error
	if err = a.store.SetProviderName(id, strings.TrimSpace(f.GetString("name"))); err == nil {
		err = a.store.SetProviderPackage(id, strings.TrimSpace(f.GetString("package")))
	}
	if err == nil {
		err = a.store.SetProviderBaseURL(id, strings.TrimSpace(f.GetString("baseURL")))
	}
	if err == nil {
		err = a.store.SetProviderEnv(id, splitVars(f.GetString("env")))
	}
	if err != nil {
		a.toastErr(err)
		return
	}
	a.saveOK("updated provider " + id)
}

// --- model form ------------------------------------------------------------

func (a *app) startModelForm(mid string) tea.Cmd {
	pid := a.providerID()
	if pid == "" {
		return nil
	}
	a.formKind = formModel
	a.formPrevID = pid
	a.formModel = mid

	m := a.store.ReadModel(pid, mid)

	contextStr, outputStr, costInStr, costOutStr := "", "", "", ""
	if m.Limit != nil {
		if m.Limit.Context != nil {
			contextStr = strconv.Itoa(*m.Limit.Context)
		}
		if m.Limit.Output != nil {
			outputStr = strconv.Itoa(*m.Limit.Output)
		}
	}
	if m.Cost != nil {
		if m.Cost.Input != nil {
			costInStr = fmt.Sprintf("%g", *m.Cost.Input)
		}
		if m.Cost.Output != nil {
			costOutStr = fmt.Sprintf("%g", *m.Cost.Output)
		}
	}
	tools := true
	if m.Capabilities != nil && m.Capabilities.Tools != nil {
		tools = *m.Capabilities.Tools
	}
	reasoning := false
	if m.Capabilities != nil && m.Capabilities.Reasoning != nil && m.Capabilities.Reasoning.Supported != nil {
		reasoning = *m.Capabilities.Reasoning.Supported
	}
	disabled := m.Disabled

	fields := []huh.Field{}
	if mid == "" {
		existing := a.store.ModelIDs(pid)
		fields = append(fields, huh.NewInput().
			Key("mid").
			Title("Model ID").
			Description("key in the models map, e.g. qwen3-coder (may contain '/')").
			Prompt("> ").
			Validate(func(v string) error {
				v = strings.TrimSpace(v)
				if v == "" {
					return errors.New("model id is required")
				}
				for _, e := range existing {
					if e == v {
						return errors.New("model already exists")
					}
				}
				return nil
			}))
	}
	fields = append(fields,
		huh.NewInput().Key("modelID").Title("Model API id").Description("id sent to the provider; blank = model id").Prompt("> ").Value(&m.ModelID),
		huh.NewInput().Key("name").Title("Display name").Description("optional").Prompt("> ").Value(&m.Name),
		huh.NewInput().Key("context").Title("Context limit").Description("tokens, e.g. 200000 (optional)").Prompt("> ").Value(&contextStr).Validate(optionalInt),
		huh.NewInput().Key("output").Title("Output limit").Description("tokens, e.g. 32768 (optional)").Prompt("> ").Value(&outputStr).Validate(optionalInt),
		huh.NewInput().Key("costIn").Title("Cost · input").Description("USD per 1M tokens (optional)").Prompt("> ").Value(&costInStr).Validate(optionalFloat),
		huh.NewInput().Key("costOut").Title("Cost · output").Description("USD per 1M tokens (optional)").Prompt("> ").Value(&costOutStr).Validate(optionalFloat),
		huh.NewConfirm().Key("tools").Title("Tool support").Value(&tools),
		huh.NewConfirm().Key("reasoning").Title("Reasoning support").Value(&reasoning),
		huh.NewConfirm().Key("disabled").Title("Disabled").Description("hide this model from /models").Value(&disabled),
	)

	a.form = huh.NewForm(huh.NewGroup(fields...)).
		WithTheme(openCodeFormTheme()).
		WithKeyMap(formKeyMap()).
		WithWidth(min(a.width-12, 64)).
		WithHeight(min(a.height-8, 24)).
		WithShowHelp(false)
	a.mode = modeForm
	return a.form.Init()
}

func (a *app) finishModelForm() {
	f := a.form
	pid := a.formPrevID
	key := a.formModel
	if key == "" {
		key = strings.TrimSpace(f.GetString("mid"))
	}
	// Preserve advanced fields (variants, compatibility, media, …) on edit.
	m := a.store.ReadModel(pid, key)

	m.ModelID = strings.TrimSpace(f.GetString("modelID"))
	m.Name = strings.TrimSpace(f.GetString("name"))

	if s := strings.TrimSpace(f.GetString("context")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			if m.Limit == nil {
				m.Limit = &config.Limit{}
			}
			m.Limit.Context = &n
		}
	}
	if s := strings.TrimSpace(f.GetString("output")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			if m.Limit == nil {
				m.Limit = &config.Limit{}
			}
			m.Limit.Output = &n
		}
	}
	if s := strings.TrimSpace(f.GetString("costIn")); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			if m.Cost == nil {
				m.Cost = &config.Cost{}
			}
			m.Cost.Input = &v
		}
	}
	if s := strings.TrimSpace(f.GetString("costOut")); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			if m.Cost == nil {
				m.Cost = &config.Cost{}
			}
			m.Cost.Output = &v
		}
	}

	tools := f.GetBool("tools")
	reasoning := f.GetBool("reasoning")
	disabled := f.GetBool("disabled")
	if m.Capabilities == nil {
		m.Capabilities = &config.Capabilities{}
	}
	m.Capabilities.Tools = &tools
	m.Capabilities.Reasoning = &config.Reasoning{Supported: &reasoning}
	m.Disabled = disabled

	if err := a.store.AddModel(pid, key, m); err != nil {
		a.toastErr(err)
		return
	}
	if !a.saveOK("saved model " + key) {
		return
	}
	for i, e := range a.store.ModelIDs(pid) {
		if e == key {
			a.msel = i
			break
		}
	}
}

// --- key form --------------------------------------------------------------

func (a *app) startKeyForm() tea.Cmd {
	pid := a.providerID()
	if pid == "" {
		return nil
	}
	a.formKind = formKey
	a.formPrevID = pid

	source := "file"
	envVar := config.SuggestEnvVar(pid)
	var keyVal string // never prefilled with secrets

	fields := []huh.Field{
		huh.NewSelect[string]().
			Key("source").
			Title("Storage").
			Options(
				huh.NewOption("Protected file (recommended)", "file"),
				huh.NewOption("Environment variable {env:…}", "env"),
				huh.NewOption("Inline in config", "inline"),
			).
			Value(&source),
		huh.NewInput().
			Key("key").
			Title("API key").
			Description("leave empty to keep the current key").
			Prompt("> ").
			EchoMode(huh.EchoModePassword).
			Value(&keyVal),
		huh.NewInput().
			Key("envVar").
			Title("Environment variable").
			Description("used when storage is env; opencode reads it at runtime").
			Prompt("> ").
			Value(&envVar),
	}

	a.form = huh.NewForm(huh.NewGroup(fields...)).
		WithTheme(openCodeFormTheme()).
		WithKeyMap(formKeyMap()).
		WithWidth(min(a.width-12, 60)).
		WithHeight(min(a.height-10, 18)).
		WithShowHelp(false)
	a.mode = modeForm
	return a.form.Init()
}

func (a *app) finishKeyForm() {
	f := a.form
	pid := a.formPrevID
	secret := strings.TrimSpace(f.GetString("key"))
	if secret == "" {
		a.status.set("key for "+pid+" left unchanged", statusInfo)
		return
	}
	switch f.GetString("source") {
	case "file":
		path := config.DefaultKeyFile(a.store.Path, pid)
		if err := config.WriteFileKey(path, secret); err != nil {
			a.toastErr(err)
			return
		}
		if err := a.store.SetProviderAPIKey(pid, "{file:"+path+"}"); err != nil {
			a.toastErr(err)
			return
		}
		a.saveOK(fmt.Sprintf("key saved to %s", shortHome(path)))

	case "env":
		v := strings.TrimSpace(f.GetString("envVar"))
		if v == "" {
			v = config.SuggestEnvVar(pid)
		}
		if err := a.store.SetProviderAPIKey(pid, "{env:"+v+"}"); err != nil {
			a.toastErr(err)
			return
		}
		p := a.store.ReadProvider(pid)
		if !contains(p.Env, v) {
			if err := a.store.SetProviderEnv(pid, append(p.Env, v)); err != nil {
				a.toastErr(err)
				return
			}
		}
		a.saveOK(fmt.Sprintf("stored as {env:%s}", v))
		sc := config.ShellExportsFor(pid, v, secret)
		if sc != nil {
			a.postForm = func(a *app) {
				a.confirm = &confirmDlg{
					title:   "Shell environment",
					message: fmt.Sprintf("%s is not exported in %s yet.\nAdd this line so opencode finds the key:\n\n  %s", v, shortHome(sc.File), sc.Line),
					okLabel: "append",
					onOK: func(a *app) {
						if err := config.ApplyShellExport(sc); err != nil {
							a.toastErr(err)
							return
						}
						a.status.set(fmt.Sprintf("exported %s in %s", v, shortHome(sc.File)), statusOK)
					},
				}
				a.mode = modeConfirm
			}
		}

	case "inline":
		if err := a.store.SetProviderAPIKey(pid, secret); err != nil {
			a.toastErr(err)
			return
		}
		if err := a.store.Save(); err != nil {
			a.toastErr(err)
			return
		}
		a.status.set(fmt.Sprintf("stored inline · key is in plaintext in %s", shortHome(a.store.Path)), statusWarn)
	}
}

// --- form lifecycle --------------------------------------------------------

func (a *app) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	// esc closes the form from any field.
	if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
		a.cancelForm()
		return a, nil
	}
	var cmd tea.Cmd
	fm, cmd := a.form.Update(msg)
	a.form = fm.(*huh.Form)
	switch a.form.State {
	case huh.StateCompleted:
		a.finishForm()
	case huh.StateAborted:
		a.cancelForm()
	}
	return a, cmd
}

func (a *app) finishForm() {
	switch a.formKind {
	case formProvider:
		a.finishProviderForm()
	case formModel:
		a.finishModelForm()
	case formKey:
		a.finishKeyForm()
	}
	a.form = nil
	a.mode = modeMain
	if a.postForm != nil {
		fn := a.postForm
		a.postForm = nil
		fn(a)
	}
}

func (a *app) cancelForm() {
	a.form = nil
	a.mode = modeMain
	a.status.set("cancelled", statusInfo)
}

func (a *app) formTitle() string {
	switch a.formKind {
	case formProvider:
		if a.formAdd {
			return "ADD PROVIDER"
		}
		return "EDIT PROVIDER · " + a.formPrevID
	case formModel:
		if a.formModel == "" {
			return "ADD MODEL · " + a.formPrevID
		}
		return "EDIT MODEL · " + a.formModel
	case formKey:
		return "API KEY · " + a.formPrevID
	}
	return "FORM"
}

func (a *app) formView() string {
	title := accent(a.formTitle()).Render(a.formTitle())
	hr := faint(strings.Repeat("─", 44)).Render(strings.Repeat("─", 44))
	footer := keyCap("tab").Render("[tab] next") + "  " +
		keyCap("↑").Render("[↑/←] up") + "  " +
		keyCap("esc").Render("[esc] close")
	body := title + "\n" + hr + "\n\n" + a.form.View() + "\n\n" + faint(footer).Render(footer)
	return boxStyle(min(a.width-8, 72)).Render(body)
}

// --- helpers ---------------------------------------------------------------

func splitVars(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

func optionalInt(v string) error {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if _, err := strconv.Atoi(v); err != nil {
		return errors.New("must be an integer")
	}
	return nil
}

func optionalFloat(v string) error {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if _, err := strconv.ParseFloat(v, 64); err != nil {
		return errors.New("must be a number")
	}
	return nil
}

func (a *app) toastErr(err error) {
	a.status.set("error: "+err.Error(), statusErr)
}
