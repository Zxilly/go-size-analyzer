package tui

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

func RunTUI(r *result.Result) {
	model := newMainModel(r)
	_, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		utils.FatalError(fmt.Errorf("TUI error: %v", err))
	}
}
