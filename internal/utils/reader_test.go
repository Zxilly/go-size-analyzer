package utils

import (
	"bytes"
	"io"
	"testing"
)

func TestReaderAtAdapter_Read(t *testing.T) {
	data := []byte("Hello, World!")
	buffer := bytes.NewReader(data)
	reader := NewReaderAtAdapter(buffer)

	// Test reading the entire data
	readData := make([]byte, len(data))
	n, err := reader.Read(readData)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("unexpected number of bytes read: got %d, want %d", n, len(data))
	}
	if !bytes.Equal(readData, data) {
		t.Errorf("unexpected data read: got %s, want %s", readData, data)
	}

	// Test reading beyond the data
	readData = make([]byte, 10)
	n, err = reader.Read(readData)
	if err != io.EOF {
		t.Errorf("unexpected error: got %v, want %v", err, io.EOF)
	}
	if n != 0 {
		t.Errorf("unexpected number of bytes read: got %d, want %d", n, 0)
	}
}
