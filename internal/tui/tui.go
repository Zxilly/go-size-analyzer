package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func RunTUI(r *result.Result, width, height int) {
	model := newMainModel(r, width, height)
	_, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		utils.FatalError(fmt.Errorf("TUI error: %w", err))
	}
}
