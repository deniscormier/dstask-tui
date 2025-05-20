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
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

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
	c := exec.Command("dstask", "context", context)
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
// Add "a" keybinding for "add" ("l" for "log")
// When pressed, hide list, unhide huh.Form ()
// https://github.com/charmbracelet/huh/blob/main/examples/bubbletea/main.go#L76
// TODO Tabs or toggle between next, show-active, show-paused, show-open, show-resolved, show-unorganized
type model struct {
	// table table.Model
	currentView    viewType
	activeContext  string
	setContextForm tea.Model
	tasks          list.Model
	err            error
}

func (m model) Init() tea.Cmd {
	return tea.Batch(dstaskActiveContext, dstaskNext, m.setContextForm.Init())
}

type KeyMap struct {
	refresh        key.Binding
	note           key.Binding
	edit           key.Binding
	open           key.Binding
	start          key.Binding
	stop           key.Binding
	done           key.Binding
	setContextView key.Binding
}

var keys = KeyMap{
	refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	// TODO remove enter for now...
	// note: key.NewBinding(
	// 	key.WithKeys("enter", "n"),
	// 	key.WithHelp("enter/n", dstask.CMD_NOTE),
	// ),
	note: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", dstask.CMD_NOTE),
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
	// TODO use esc, but only in "set context" view
	// setNextView: key.NewBinding(
	// 	key.WithKeys("c"),
	// 	key.WithHelp("c", "Set context..."),
	// ),
	setContextView: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "Set context..."),
	),
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.currentView == viewTypeTasksNext && !m.tasks.SettingFilter() {
			switch {
			case key.Matches(msg, keys.refresh):
				return m, tea.Batch(dstaskActiveContext, dstaskNext)
			case key.Matches(msg, keys.note):
				i, ok := m.tasks.SelectedItem().(dstaskListItem)
				if ok {
					return m, dstaskCmdForID(dstask.CMD_NOTE, i.id)
				}
			case key.Matches(msg, keys.edit):
				i, ok := m.tasks.SelectedItem().(dstaskListItem)
				if ok {
					return m, dstaskCmdForID(dstask.CMD_EDIT, i.id)
				}
			case key.Matches(msg, keys.open):
				i, ok := m.tasks.SelectedItem().(dstaskListItem)
				if ok {
					return m, dstaskCmdForID(dstask.CMD_OPEN, i.id)
				}
			case key.Matches(msg, keys.start):
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
			case key.Matches(msg, keys.stop):
				i, ok := m.tasks.SelectedItem().(dstaskListItem)
				if ok {
					return m, dstaskCmdForID(dstask.CMD_STOP, i.id)
				}
			case key.Matches(msg, keys.done):
				i, ok := m.tasks.SelectedItem().(dstaskListItem)
				if ok {
					return m, dstaskCmdForID(dstask.CMD_DONE, i.id)
				}
			case key.Matches(msg, keys.setContextView):
				m.currentView = viewTypeSetContext
				return m, nil
			case msg.String() == "q":
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.tasks.SetSize(msg.Width-h, msg.Height-v-2)
	case dstaskActiveContextMsg:
		m.activeContext = msg.activeContext
		return m, nil
	case dstaskSetContextMsg:
		return m, dstaskActiveContext
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
	}
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

func (m model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error()
	}
	var currentViewContents string
	switch m.currentView {
	case viewTypeTasksNext:
		currentViewContents = m.tasks.View()
	case viewTypeSetContext:
		currentViewContents = m.setContextForm.View()
	default:
		return "Error: invalid view type set in app"
	}
	return docStyle.Render("Active context: " + m.activeContext + "\n\n" + currentViewContents)
}

func main() {
	m := model{}
	m.tasks = list.New(nil, list.NewDefaultDelegate(), 0, 0)
	m.tasks.Title = "dstask " + dstask.CMD_NEXT
	m.tasks.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.refresh,
			keys.note,
			keys.edit,
			keys.done,
		}
	}
	m.tasks.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.refresh,
			keys.note,
			keys.edit,
			keys.open,
			keys.start,
			keys.stop,
			keys.done,
			keys.setContextView,
		}
	}
	m.tasks.SetStatusBarItemName("task", "tasks")

	setContextForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("context").
				Title("Set context to...").
				Description("Guides tasks displayed by default").
				Placeholder("P1 +this -that project:myproject"),
		),
	)
	setContextForm.SubmitCmd = func() tea.Msg {
		return dstaskSetContext(setContextForm.GetString("context"))
	}
	m.setContextForm = setContextForm
	// TODO show help menu in form

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
