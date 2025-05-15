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
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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

type dstaskNextMsg struct{ tasks []dstask.Task }

func dstaskCmdForID(cmd string, id string) tea.Cmd {
	c := exec.Command("dstask", cmd, id)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return dstaskErrorMsg{err}
	})
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

// TODO Model and view for context command (Always see current context)
// TODO Model and view for add/log commands (tabs/toggle between, aceept input)
// TODO Tabs or toggle between next, show-active, show-paused, show-open, show-resolved, show-unorganized
type model struct {
	// table table.Model
	listModel list.Model
	err       error
}

func (m model) Init() tea.Cmd {
	return dstaskNext
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return m, dstaskNext
		case "n", "enter":
			i, ok := m.listModel.SelectedItem().(dstaskListItem)
			if ok {
				return m, dstaskCmdForID("note", i.id)
			}
		case "e":
			i, ok := m.listModel.SelectedItem().(dstaskListItem)
			if ok {
				return m, dstaskCmdForID("edit", i.id)
			}
		case "o":
			i, ok := m.listModel.SelectedItem().(dstaskListItem)
			if ok {
				return m, dstaskCmdForID("open", i.id)
			}
		case "d":
			i, ok := m.listModel.SelectedItem().(dstaskListItem)
			if ok {
				return m, dstaskCmdForID("done", i.id)
			}
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.listModel.SetSize(msg.Width-h, msg.Height-v)
	case dstaskNextMsg:
		var taskItems []list.Item
		for _, task := range msg.tasks {
			// TODO Indicator for started/stopped
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
				title:       task.Summary,
				description: description,
			})
		}
		return m, m.listModel.SetItems(taskItems)
	case dstaskErrorMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		} else {
			return m, dstaskNext
		}
	}
	var cmd tea.Cmd
	m.listModel, cmd = m.listModel.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error() + "\n"
	} else {
		return m.listModel.View()
	}
	// return "Press 'e' to open your EDITOR.\nPress 'a' to toggle the altscreen\nPress 'q' to quit.\n"
}

func main() {
	m := model{}
	m.listModel = list.New(nil, list.NewDefaultDelegate(), 0, 0)
	m.listModel.Title = "dstask next"
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
