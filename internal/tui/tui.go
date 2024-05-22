package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"

	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func RunTUI(r *result.Result) {
	w, h, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		utils.FatalError(fmt.Errorf("failed to get terminal size: %w", err))
	}

	model := newMainModel(r, w, h)
	_, err = tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		utils.FatalError(fmt.Errorf("TUI error: %w", err))
	}
}
