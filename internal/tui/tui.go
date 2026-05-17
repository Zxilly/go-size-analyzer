package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/Zxilly/go-size-analyzer/internal/result"
)

func RunTUI(r *result.Result, width, height int) error {
	model := newMainModel(r, width, height)
	_, err := tea.NewProgram(model).Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
