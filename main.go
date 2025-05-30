package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/naggie/dstask"
	// "github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ****
// "dstask next" list extra keybindings
// ****
type dstaskNextKeyMap struct {
	refresh        key.Binding
	note           key.Binding
	edit           key.Binding
	open           key.Binding
	start          key.Binding
	stop           key.Binding
	done           key.Binding
	setContextView key.Binding
	quit           key.Binding
}

var dstaskNextKeys = dstaskNextKeyMap{
	refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	note: key.NewBinding(
		key.WithKeys("enter", "n"),
		key.WithHelp("enter/n", dstask.CMD_NOTE),
	),
	edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", dstask.CMD_EDIT),
	),
	open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", dstask.CMD_OPEN),
	),
	start: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", dstask.CMD_START),
	),
	stop: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", dstask.CMD_STOP),
	),
	done: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", dstask.CMD_DONE),
	),
	setContextView: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "Set context..."),
	),
	quit: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc/q", "quit"),
	),
}

// ****
// "dstask context" keybindings
// ****
type setContextKeyMap struct {
	submit key.Binding
	cancel key.Binding
}

var setContextKeys = setContextKeyMap{
	submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "submit"),
	),
	cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

func (k setContextKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.submit, k.cancel}
}

func (k setContextKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.submit, k.cancel}}
}

type dstaskListItem struct {
	id          string
	title       string
	description string
}

// TODO Exact filtering for ID + Priority
func (i dstaskListItem) FilterValue() string { return i.title + "\n" + i.description }
func (i dstaskListItem) Title() string       { return i.title }
func (i dstaskListItem) Description() string { return i.description }

// TODO Status bar at the bottom
// * Display errors instead of crashing the script
// * Display command output from sync (as an example)
type dstaskErrorMsg struct{ err error }

type dstaskActiveContextMsg struct{ activeContext string }

type dstaskSetContextMsg struct{}

type dstaskNextMsg struct{ tasks []dstask.Task }

func dstaskCmdForID(cmd string, id string) tea.Cmd {
	c := exec.Command("dstask", cmd, id)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return dstaskErrorMsg{err}
	})
}

func dstaskActiveContext() tea.Msg {
	c := exec.Command("dstask", "context")
	b, err := c.Output()
	if err != nil {
		return dstaskErrorMsg{err}
	}
	// The newline from the command output complicates the rendering. Remove it.
	trimmed := strings.TrimRight(string(b), "\n")
	return dstaskActiveContextMsg{trimmed}
}

func dstaskSetContext(context string) tea.Msg {
	args := []string{"context"}
	args = append(args, strings.Fields(context)...)
	c := exec.Command("dstask", args...)
	err := c.Run()
	if err != nil {
		return dstaskErrorMsg{err}
	}
	return dstaskSetContextMsg{}
}

func dstaskNext() tea.Msg {
	c := exec.Command("dstask")
	b, err := c.Output()
	if err != nil {
		return dstaskErrorMsg{err}
	}
	var tasks []dstask.Task
	err = json.Unmarshal(b, &tasks)
	if err != nil {
		return dstaskErrorMsg{err}
	}
	return dstaskNextMsg{tasks}
}

type viewType int

const (
	viewTypeTasksNext viewType = iota
	viewTypeSetContext
)

// TODO Model and view for add/log commands (tabs/toggle between, accept input)
// Add "a" keybinding for "add" ("l" for "log"), follow setContextView example
// TODO Tabs or toggle between next, show-active, show-paused, show-open, show-resolved, show-unorganized
type model struct {
	// table table.Model
	currentView    viewType
	tasks          list.Model
	activeContext  string
	setContextForm tea.Model
	setContextHelp help.Model
	err            error
}

func (m model) Init() tea.Cmd {
	return tea.Batch(dstaskActiveContext, dstaskNext, m.setContextForm.Init())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.currentView {
		case viewTypeTasksNext:
			if !m.tasks.SettingFilter() {
				switch {
				case key.Matches(msg, dstaskNextKeys.refresh):
					return m, tea.Batch(dstaskActiveContext, dstaskNext)
				case key.Matches(msg, dstaskNextKeys.note):
					i, ok := m.tasks.SelectedItem().(dstaskListItem)
					if ok {
						return m, dstaskCmdForID(dstask.CMD_NOTE, i.id)
					}
				case key.Matches(msg, dstaskNextKeys.edit):
					i, ok := m.tasks.SelectedItem().(dstaskListItem)
					if ok {
						return m, dstaskCmdForID(dstask.CMD_EDIT, i.id)
					}
				case key.Matches(msg, dstaskNextKeys.open):
					i, ok := m.tasks.SelectedItem().(dstaskListItem)
					if ok {
						return m, dstaskCmdForID(dstask.CMD_OPEN, i.id)
					}
				case key.Matches(msg, dstaskNextKeys.start):
					i, ok := m.tasks.SelectedItem().(dstaskListItem)
					if ok {
						// TODO status messages (example here)
						// return m, tea.Sequence(
						// 	dstaskCmdForID(dstask.CMD_START, i.id),
						// 	m.listModel.NewStatusMessage("Hi there!"))
						// Change status mesage lifetime default of 1 second
						// m.listModel.StatusMessageLifetime()
						return m, dstaskCmdForID(dstask.CMD_START, i.id)
					}
				case key.Matches(msg, dstaskNextKeys.stop):
					i, ok := m.tasks.SelectedItem().(dstaskListItem)
					if ok {
						return m, dstaskCmdForID(dstask.CMD_STOP, i.id)
					}
				case key.Matches(msg, dstaskNextKeys.done):
					i, ok := m.tasks.SelectedItem().(dstaskListItem)
					if ok {
						return m, dstaskCmdForID(dstask.CMD_DONE, i.id)
					}
				case key.Matches(msg, dstaskNextKeys.setContextView):
					m.currentView = viewTypeSetContext
					return m, nil
				case key.Matches(msg, dstaskNextKeys.quit):
					return m, tea.Quit
				}
			}
			var cmd tea.Cmd
			m.tasks, cmd = m.tasks.Update(msg)
			return m, cmd

		case viewTypeSetContext:
			if msg.String() == "esc" {
				m.currentView = viewTypeTasksNext
				return m, nil
			}
			var cmd tea.Cmd
			m.setContextForm, cmd = m.setContextForm.Update(msg)
			if f, ok := m.setContextForm.(*huh.Form); ok {
				m.setContextForm = f
			}
			return m, cmd
		} // switch m.currentView
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.tasks.SetSize(msg.Width-h, msg.Height-v-2)
	case dstaskActiveContextMsg:
		m.activeContext = msg.activeContext
		return m, nil
	case dstaskSetContextMsg:
		m.currentView = viewTypeTasksNext
		m.setContextForm = newSetContextForm()
		cmd := m.setContextForm.Init()
		return m, tea.Batch(dstaskActiveContext, dstaskNext, cmd)
	case dstaskNextMsg:
		var taskItems []list.Item
		for _, task := range msg.tasks {
			title := task.Summary
			if task.Status != dstask.STATUS_PENDING {
				title += " %" + task.Status
			}
			description := fmt.Sprintf("#%d %s", task.ID, task.Priority)
			tags := strings.Join(task.Tags, " +")
			if tags != "" {
				description += " +" + tags
			}
			if task.Project != "" {
				description += " project:" + task.Project
			}
			notes := strings.TrimSpace(task.Notes)
			noteLines := strings.Split(notes, "\n")
			lastNote := noteLines[len(noteLines)-1]
			if len(lastNote) > 0 {
				description += " / " + lastNote
			}
			taskItems = append(taskItems, dstaskListItem{
				// title:       title,
				// description: task.LongSummary(),
				id:          strconv.Itoa(task.ID),
				title:       title,
				description: description,
			})
		}
		return m, m.tasks.SetItems(taskItems)
	case dstaskErrorMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		} else {
			return m, dstaskNext
		}
	} // switch msg := msg.(type)
	var cmd tea.Cmd
	m.tasks, cmd = m.tasks.Update(msg)
	batch := tea.BatchMsg{cmd}

	m.setContextForm, cmd = m.setContextForm.Update(msg)
	if f, ok := m.setContextForm.(*huh.Form); ok {
		m.setContextForm = f
		batch = append(batch, cmd)
	}
	return m, tea.Batch(batch...)
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

func (m model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error()
	}
	currentViewContents := "Active context: " + m.activeContext + "\n\n"
	switch m.currentView {
	case viewTypeTasksNext:
		currentViewContents += m.tasks.View()
	case viewTypeSetContext:
		currentViewContents += m.setContextForm.View() + "\n\n" + m.setContextHelp.View(setContextKeys)
	default:
		return "Error: invalid view type set in app"
	}
	return docStyle.Render(currentViewContents)
}

func newSetContextForm() *huh.Form {
	input := huh.NewInput().
		Title("Set context to...").
		Description(`Provide "dstask context" args for default filtering of tasks`).
		Placeholder("P1 +this -that project:myproject")
	setContextForm := huh.NewForm(huh.NewGroup(input))
	setContextForm.SubmitCmd = func() tea.Msg {
		return dstaskSetContext(input.GetValue().(string))
	}
	// Create our own help text
	setContextForm.WithShowHelp(false)
	return setContextForm
}

func initialModel() model {
	tasks := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	tasks.DisableQuitKeybindings()
	tasks.SetStatusBarItemName("task", "tasks")
	tasks.Title = "dstask " + dstask.CMD_NEXT
	tasks.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			dstaskNextKeys.refresh,
			dstaskNextKeys.note,
			dstaskNextKeys.edit,
			dstaskNextKeys.done,
			dstaskNextKeys.quit,
		}
	}
	tasks.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			dstaskNextKeys.refresh,
			dstaskNextKeys.note,
			dstaskNextKeys.edit,
			dstaskNextKeys.open,
			dstaskNextKeys.start,
			dstaskNextKeys.stop,
			dstaskNextKeys.done,
			dstaskNextKeys.setContextView,
			dstaskNextKeys.quit,
		}
	}

	setContextHelp := help.New()
	setContextHelp.ShortHelpView([]key.Binding{
		setContextKeys.submit,
		setContextKeys.cancel,
	})

	return model{
		tasks:          tasks,
		setContextForm: newSetContextForm(),
		setContextHelp: setContextHelp,
	}
}

func main() {
	if _, err := tea.NewProgram(initialModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
