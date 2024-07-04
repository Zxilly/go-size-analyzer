package utils

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderAtAdapter_Read(t *testing.T) {
	data := []byte("Hello, World!")
	buffer := bytes.NewReader(data)
	reader := NewReaderAtAdapter(buffer)

	// Test reading the entire data
	readData := make([]byte, len(data))
	n, err := reader.Read(readData)
	require.NoError(t, err)
	assert.Len(t, data, n)
	assert.Equal(t, data, readData)

	// Test reading beyond the data
	readData = make([]byte, 10)
	n, err = reader.Read(readData)
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, n, 0)
}
