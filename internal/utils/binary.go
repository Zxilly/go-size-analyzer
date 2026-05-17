package utils

import (
	"io"
	"log/slog"
	"os"
	"sync"

	"golang.org/x/exp/mmap"
)

type BinaryFile interface {
	io.ReaderAt
	io.Closer
	Len() int
}

type fileBinary struct {
	f    *os.File
	size int
}

func (b *fileBinary) ReadAt(p []byte, off int64) (int, error) { return b.f.ReadAt(p, off) }
func (b *fileBinary) Close() error                            { return b.f.Close() }
func (b *fileBinary) Len() int                                { return b.size }

var mmapFallbackWarn sync.Once

// OpenBinary opens path for read. It tries mmap first and falls back to a
// regular file handle when mmap fails — for example on WSL2's 9p mounts
// under /mnt, where mmap returns ENODEV. Callers wrap the returned error.
func OpenBinary(path string) (BinaryFile, error) {
	r, mmapErr := mmap.Open(path)
	if mmapErr == nil {
		return r, nil
	}
	mmapFallbackWarn.Do(func() {
		slog.Warn("mmap failed, falling back to regular file read", "path", path, "error", mmapErr)
	})

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return &fileBinary{f: f, size: int(info.Size())}, nil
}
