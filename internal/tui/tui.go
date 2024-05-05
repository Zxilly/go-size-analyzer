package tui

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	tea "github.com/charmbracelet/bubbletea"
	"log/slog"
	"os"
)

func RunTUI(result *result.Result) {
	model := newViewModel(result)
	_, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		slog.Error(fmt.Sprintf("TUI error: %v", err))
		os.Exit(1)
	}
}
