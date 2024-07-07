package utils

import (
	"errors"
	"io"
)

type ReaderAtAdapter struct {
	readerAt io.ReaderAt
	offset   int64
}

func NewReaderAtAdapter(readerAt io.ReaderAt) *ReaderAtAdapter {
	return &ReaderAtAdapter{readerAt: readerAt}
}

func (r *ReaderAtAdapter) Read(p []byte) (n int, err error) {
	n, err = r.readerAt.ReadAt(p, r.offset)
	r.offset += int64(n)
	if errors.Is(err, io.EOF) && n > 0 {
		return n, nil
	}
	return n, err
}
