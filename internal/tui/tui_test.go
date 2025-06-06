//go:build !js && !wasm

package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"

	"github.com/Zxilly/go-size-analyzer/internal/test"
)

func init() {
	lipgloss.SetColorProfile(termenv.Ascii)
}

func TestFullOutput(t *testing.T) {
	m := newMainModel(test.GetTestResult(t), 300, 100)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(300, 100))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("runtime"))
	}, teatest.WithCheckInterval(time.Millisecond*200), teatest.WithDuration(time.Second*10))

	// test scroll
	tm.Send(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
	})

	tm.Send(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
	})

	// test detail
	tm.Send(tea.KeyMsg{
		Type: tea.KeyTab,
	})

	tm.Send(tea.KeyMsg{
		Type: tea.KeyDown,
	})
	tm.Send(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
	})

	// switch back
	tm.Send(tea.KeyMsg{
		Type: tea.KeyTab,
	})

	tm.Send(tea.KeyMsg{
		Type: tea.KeyEnter,
	})

	// resize window
	tm.Send(tea.WindowSizeMsg{
		Width:  200,
		Height: 100,
	})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("proc.go"))
	}, teatest.WithCheckInterval(time.Millisecond*100), teatest.WithDuration(time.Second*3))

	// enter proc.go
	for range 4 {
		tm.Send(tea.KeyMsg{
			Type: tea.KeyDown,
		})
	}
	tm.Send(tea.KeyMsg{
		Type: tea.KeyEnter,
	})

	// list all
	for range 20 {
		tm.Send(tea.KeyMsg{
			Type: tea.KeyDown,
		})
	}

	// back to runtime
	tm.Send(tea.KeyMsg{
		Type: tea.KeyBackspace,
	})

	for range 20 {
		tm.Send(tea.KeyMsg{
			Type: tea.KeyDown,
		})
	}

	// back to root
	tm.Send(tea.KeyMsg{
		Type: tea.KeyBackspace,
	})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte(".gopclntab"))
	}, teatest.WithCheckInterval(time.Millisecond*100), teatest.WithDuration(time.Second*3))

	tm.Send(tea.KeyMsg{
		Type: tea.KeyCtrlC,
	})

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
}
