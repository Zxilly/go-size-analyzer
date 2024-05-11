package utils

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFatalError(t *testing.T) {
	m := &mock.Mock{}
	m.On("os.Exit", 1).Return()
	exitFunc = func(code int) {
		m.MethodCalled("os.Exit", code)
	}

	FatalError(nil)
	assert.True(t, m.AssertNotCalled(t, "os.Exit", 1))

	FatalError(assert.AnError)
	assert.True(t, m.AssertCalled(t, "os.Exit", 1))
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
