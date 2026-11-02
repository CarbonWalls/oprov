package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/oprov/oprov/internal/config"
)

type focusPane int

const (
	focusList focusPane = iota
	focusDetail
)

type mode int

const (
	modeMain mode = iota
	modeForm
	modeConfirm
)

// tickMsg fires to expire status messages.
type tickMsg struct{}

// testMsg carries the result of an async connection check.
type testMsg struct {
	id  string
	res config.TestResult
}

type formKind int

const (
	formProvider formKind = iota
	formModel
	formKey
)

// app is the root Bubble Tea model.
type app struct {
	store *config.Store

	width  int
	height int

	mode     mode
	focus    focusPane
	showHelp bool

	sel  int // selected provider index
	msel int // selected model index in the detail pane

	status statusBar

	form       *huh.Form
	formKind   formKind
	formPrevID string     // provider id under edit (for its key form)
	formAdd    bool       // provider/model form is in "add" mode
	formModel  string     // model id under edit ("" when adding)
	postForm   func(*app) // deferred action after a form finishes

	confirm *confirmDlg

	testing   bool
	testID    string
	spinner   spinner.Model
	spinnerOn bool
}

type statusKind int

const (
	statusInfo statusKind = iota
	statusOK
	statusErr
	statusWarn
)

type statusBar struct {
	msg  string
	kind statusKind
	at   time.Time
}

func (s *statusBar) set(msg string, k statusKind) {
	s.msg = msg
	s.kind = k
	s.at = time.Now()
}

func (s *statusBar) expired() bool {
	return s.msg != "" && time.Since(s.at) > 6*time.Second
}

func newApp(st *config.Store) *app {
	return &app{
		store: st,
	}
}

// New creates the TUI application for the given config store.
func New(st *config.Store) tea.Model {
	return newApp(st)
}

func (a *app) Init() tea.Cmd {
	a.status.set("ready", statusInfo)
	return tea.Batch(
		tea.SetWindowTitle("oprov · opencode providers"),
		waitTick(),
	)
}

func waitTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		return a, nil

	case tea.KeyMsg:
		return a.updateKey(m)

	case tickMsg:
		if a.status.expired() {
			a.status.msg = ""
		}
		return a, waitTick()

	case spinner.TickMsg:
		if a.testing && a.spinnerOn {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(m)
			return a, cmd
		}
		return a, nil

	case testMsg:
		a.testing = false
		a.spinnerOn = false
		if m.res.OK {
			a.status.set(fmt.Sprintf("test %s · %s", m.id, m.res.Message), statusOK)
		} else {
			a.status.set(fmt.Sprintf("test %s · %s", m.id, m.res.Message), statusErr)
		}
		return a, nil
	}

	if a.mode == modeForm && a.form != nil {
		return a.updateForm(msg)
	}
	return a, nil
}

func (a *app) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// While a form is open, every keystroke belongs to the form.
	if a.mode == modeForm && a.form != nil {
		return a.updateForm(msg)
	}
	if a.mode == modeConfirm {
		return a.updateConfirmKey(msg)
	}
	if a.showHelp {
		switch key {
		case "?", "esc", "q":
			a.showHelp = false
		}
		return a, nil
	}

	switch key {
	case "ctrl+c":
		return a, tea.Quit
	case "?":
		a.showHelp = !a.showHelp
		return a, nil
	}

	switch a.focus {
	case focusList:
		return a.updateListKey(key)
	case focusDetail:
		return a.updateDetailKey(key)
	}
	return a, nil
}

func (a *app) updateListKey(key string) (tea.Model, tea.Cmd) {
	n := len(a.store.ProviderIDs())
	switch key {
	case "j", "down":
		if n > 0 {
			a.sel = (a.sel + 1) % n
		}
	case "k", "up":
		if n > 0 {
			a.sel = (a.sel - 1 + n) % n
		}
	case "g":
		a.sel = 0
	case "G":
		a.sel = n - 1
	case "l", "enter", "tab":
		if n > 0 {
			a.focus = focusDetail
			a.msel = 0
		}
	case "a":
		return a, a.startProviderForm(false)
	case "e":
		if n > 0 {
			return a, a.startProviderForm(true)
		}
	case "d":
		if n > 0 {
			return a, a.startConfirm(delProviderConfirm(a))
		}
	case "c":
		if n > 0 {
			return a, a.startKeyForm()
		}
	case "x":
		if n > 0 {
			return a, a.startConfirm(clearKeyConfirm(a))
		}
	case "t":
		if n > 0 {
			return a, a.startTest()
		}
	case "q":
		return a, tea.Quit
	}
	return a, nil
}

func (a *app) updateDetailKey(key string) (tea.Model, tea.Cmd) {
	n := len(a.modelIDs())
	switch key {
	case "j", "down":
		if n > 0 {
			a.msel = (a.msel + 1) % n
		}
	case "k", "up":
		if n > 0 {
			a.msel = (a.msel - 1 + n) % n
		}
	case "g":
		a.msel = 0
	case "G":
		a.msel = n - 1
	case "h", "esc", "tab":
		a.focus = focusList
		a.msel = 0
	case "a":
		return a, a.startModelForm("")
	case "e", "enter":
		if n > 0 {
			return a, a.startModelForm(a.modelIDs()[a.msel])
		}
	case "d":
		if n > 0 {
			return a, a.startConfirm(delModelConfirm(a))
		}
	case "c":
		return a, a.startKeyForm()
	case "x":
		return a, a.startConfirm(clearKeyConfirm(a))
	case "t":
		return a, a.startTest()
	case "q":
		return a, tea.Quit
	}
	return a, nil
}

// --- helpers ---------------------------------------------------------------

func (a *app) providerID() string {
	ids := a.store.ProviderIDs()
	if len(ids) == 0 {
		return ""
	}
	if a.sel < 0 || a.sel >= len(ids) {
		a.sel = 0
	}
	return ids[a.sel]
}

func (a *app) modelIDs() []string {
	return a.store.ModelIDs(a.providerID())
}

func (a *app) startTest() tea.Cmd {
	id := a.providerID()
	if id == "" {
		return nil
	}
	a.testing = true
	a.spinnerOn = true
	a.testID = id
	a.spinner = spinner.New(spinner.WithSpinner(spinner.Dot))
	a.spinner.Style = inter("")
	a.status.set(fmt.Sprintf("testing %s …", id), statusInfo)
	return tea.Batch(
		a.testCmd(id),
		a.spinner.Tick,
	)
}

// testCmd performs the connection check off-thread.
func (a *app) testCmd(id string) tea.Cmd {
	p := a.store.ReadProvider(id)
	return func() tea.Msg {
		res := config.TestConnection(&p, 12*time.Second)
		return testMsg{id: id, res: res}
	}
}

func shortHome(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(os.PathSeparator)) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}
