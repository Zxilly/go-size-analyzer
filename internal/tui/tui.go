package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Zxilly/go-size-analyzer/internal/result"
)

func RunTUI(r *result.Result, width, height int) error {
	model := newMainModel(r, width, height)
	_, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
}
