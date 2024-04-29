package utils

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"testing"
)

func TestFatalError(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		FatalError(errors.New("test"))
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestFatalError")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return
	} else {
		t.Fatalf("process ran with err %v, want exit status 1", err)
	}
}

func TestSyncOutputWriteLocksAndWrites(t *testing.T) {
	var buf bytes.Buffer
	syncOutput := &SyncOutput{output: &buf}
	_, err := syncOutput.Write([]byte("test"))

	assert.NoError(t, err)
	assert.Equal(t, "test", buf.String())
}

func TestSyncOutputSetOutputLocksAndSets(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	syncOutput := &SyncOutput{output: &buf1}
	syncOutput.SetOutput(&buf2)
	_, err := syncOutput.Write([]byte("test"))

	assert.NoError(t, err)
	assert.Equal(t, "", buf1.String())
	assert.Equal(t, "test", buf2.String())
}
