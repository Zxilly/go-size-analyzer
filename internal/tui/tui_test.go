//go:build !js && !wasm

package tui

import (
	"bytes"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/Zxilly/go-size-analyzer/internal/test"
)

func sendKey(tm *teatest.TestModel, code rune) {
	tm.Send(tea.KeyPressMsg{Code: code})
}

func sendKeyN(tm *teatest.TestModel, code rune, n int) {
	for range n {
		sendKey(tm, code)
	}
}

func TestFullOutput(t *testing.T) {
	m := newMainModel(test.GetTestResult(t), 300, 100)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(300, 100),
		teatest.WithProgramOptions(tea.WithColorProfile(colorprofile.Ascii)),
	)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("runtime"))
	}, teatest.WithCheckInterval(time.Millisecond*200), teatest.WithDuration(time.Second*10))

	// test scroll
	tm.Send(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	tm.Send(tea.MouseWheelMsg{Button: tea.MouseWheelUp})

	// test detail
	sendKey(tm, tea.KeyTab)
	sendKey(tm, tea.KeyDown)
	tm.Send(tea.MouseWheelMsg{Button: tea.MouseWheelUp})

	// switch back
	sendKey(tm, tea.KeyTab)
	sendKey(tm, tea.KeyEnter)

	// resize window
	tm.Send(tea.WindowSizeMsg{
		Width:  200,
		Height: 100,
	})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("proc.go"))
	}, teatest.WithCheckInterval(time.Millisecond*100), teatest.WithDuration(time.Second*3))

	// enter proc.go
	sendKeyN(tm, tea.KeyDown, 4)
	sendKey(tm, tea.KeyEnter)

	// list all
	sendKeyN(tm, tea.KeyDown, 20)

	// back to runtime
	sendKey(tm, tea.KeyBackspace)
	sendKeyN(tm, tea.KeyDown, 20)

	// back to root
	sendKey(tm, tea.KeyBackspace)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte(".gopclntab"))
	}, teatest.WithCheckInterval(time.Millisecond*100), teatest.WithDuration(time.Second*3))

	tm.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
}
