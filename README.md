# dstask-tui

A terminal UI for managing tasks. Wraps https://github.com/naggie/dstask using
the https://github.com/charmbracelet/bubbletea TUI framework

Install: `go install github.com/deniscormier/dstask-tui@latest`

# Future Ideas

Finalize repo and tool rename to `dstui`

A command that scans for URLs and presents them in a multi-select for selecting which ones to open
* Get file name
* Make use of https://github.com/mvdan/xurls (command line tool)

# Developer docs

Run: `go run main.go`

https://github.com/naggie/dstask

TUI framework repositories

* https://github.com/charmbracelet/bubbletea (framework and example usage)
    * https://github.com/charmbracelet/bubbletea/blob/main/examples
* https://github.com/charmbracelet/bubbles (components)
* https://github.com/charmbracelet/lipgloss (styling)
* https://github.com/charmbracelet/huh (forms)
    * https://github.com/charmbracelet/huh/blob/main/examples
