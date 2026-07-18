// tui-type is a minimalist, local-only typing test for the terminal.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nick/tui-type/internal/app"
	"github.com/nick/tui-type/internal/stats"
)

func main() {
	path, err := stats.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tui-type:", err)
		os.Exit(1)
	}
	store, err := stats.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tui-type: reading stats:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(app.New(store), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui-type:", err)
		os.Exit(1)
	}
}
