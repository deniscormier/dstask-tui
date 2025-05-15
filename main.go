package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/naggie/dstask"
	// "github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type dstaskListItem struct {
	title       string
	description string
}

func (i dstaskListItem) FilterValue() string { return i.title }
func (i dstaskListItem) Title() string       { return i.title }
func (i dstaskListItem) Description() string { return i.description }

type dstaskErrorMsg struct{ err error }

type dstaskNextMsg struct{ tasks []dstask.Task }

// func openEditor() tea.Cmd {
// 	editor := os.Getenv("EDITOR")
// 	if editor == "" {
// 		editor = "vim"
// 	}
// 	c := exec.Command(editor) //nolint:gosec
// 	return tea.ExecProcess(c, func(err error) tea.Msg {
// 		return editorFinishedMsg{err}
// 	})
// }

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

	// return tea.ExecProcess(c, func(err error) tea.Msg {
	// 	return editorFinishedMsg{err}
	// })
}

type model struct {
	// altscreenActive bool
	// table table.Model
	// contents string
	listModel list.Model
	err       error
}

func (m model) Init() tea.Cmd {
	// return nil
	return dstaskNext
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// case "a":
		// 	m.altscreenActive = !m.altscreenActive
		// 	cmd := tea.EnterAltScreen
		// 	if !m.altscreenActive {
		// 		cmd = tea.ExitAltScreen
		// 	}
		// 	return m, cmd
		case "r":
			// return m, openEditor()
			return m, dstaskNext
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.listModel.SetSize(msg.Width-h, msg.Height-v)
	case dstaskNextMsg:
		var taskItems []list.Item
		for _, task := range msg.tasks {
			title := fmt.Sprintf("%-2d | %s", task.ID, task.Priority)
			tags := strings.Join(task.Tags, " +")
			if tags != "" {
				title = title + " | +" + tags
			}
			if task.Project != "" {
				title = title + " | project:" + task.Project
			}
			taskItems = append(taskItems, dstaskListItem{
				title:       title,
				description: task.LongSummary(),
			})
		}
		return m, m.listModel.SetItems(taskItems)
		// m.contents = string(msg.contents)
	case dstaskErrorMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
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
	m.listModel.Title = "dstask"
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
