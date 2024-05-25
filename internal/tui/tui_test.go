package tui

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/mmap"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

func init() {
	lipgloss.SetColorProfile(termenv.Ascii)
}

func GetProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func GetTestResult(t *testing.T) *result.Result {
	t.Helper()

	// test against bin-linux-1.21-amd64
	path := filepath.Join(GetProjectRoot(), "scripts", "bins", "bin-linux-1.21-amd64")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("failed to get absolute path of %s: %v", path, err)
	}

	f, err := mmap.Open(path)
	require.NoError(t, err)

	r, err := internal.Analyze(path, f, uint64(f.Len()), internal.Options{})
	if err != nil {
		t.Fatalf("failed to analyze %s: %v", path, err)
	}
	return r
}

func TestFullOutput(t *testing.T) {
	m := newMainModel(GetTestResult(t), 300, 100)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(300, 100))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("runtime"))
	}, teatest.WithCheckInterval(time.Millisecond*100), teatest.WithDuration(time.Second*3))

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
	}, teatest.WithCheckInterval(time.Millisecond*100), teatest.WithDuration(time.Second*1))

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
	}, teatest.WithCheckInterval(time.Millisecond*100), teatest.WithDuration(time.Second*1))

	tm.Send(tea.KeyMsg{
		Type: tea.KeyCtrlC,
	})

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*1))
}
