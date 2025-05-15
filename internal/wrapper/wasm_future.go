//go:build use_this_if_buildinfo_was_enabled_on_wasm

package wrapper

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"runtime/debug"
)

// following copied from debug/buildinfo/buildinfo.go
var buildInfoMagic = []byte("\xff Go buildinf:")

const (
	buildInfoAlign      = 16
	buildInfoHeaderSize = 32
)

var errInvalidBuildInfo = errors.New("invalid build info")

func (w *WasmWrapper) GetModInfo() *debug.BuildInfo {
	data := w.memory

	var addr int

	for len(data) > 0 {
		size := len(data)

		i := bytes.Index(data, buildInfoMagic)
		if i < 0 {
			slog.Warn("wasm module does not contain build info")
			return nil
		}
		if size-i < buildInfoHeaderSize {
			return nil
		}
		if i%buildInfoAlign != 0 {
			next := (i + buildInfoAlign - 1) &^ (buildInfoAlign - 1)
			if next > size {
				slog.Warn("buildinfo align out of range")
				return nil
			}
			data = data[next:]
			continue
		}

		addr = i
		break
	}

	header := data[addr : addr+buildInfoHeaderSize]

	readData := func(offset int, size int) ([]byte, error) {
		if offset+size > len(data) {
			return nil, io.ErrUnexpectedEOF
		}
		return data[offset : offset+size], nil
	}

	decodeString := func(offset int) (string, int, error) {
		b, err := readData(offset, binary.MaxVarintLen64)
		if err != nil {
			return "", 0, err
		}

		length, n := binary.Uvarint(b)
		if n <= 0 {
			return "", 0, errInvalidBuildInfo
		}
		offset += n

		b, err = readData(offset, int(length))
		if err != nil {
			return "", 0, err
		}
		if len(b) < int(length) {
			return "", 0, errInvalidBuildInfo
		}

		return string(b), offset + int(length), nil
	}

	const (
		ptrSizeOffset = 14
		flagsOffset   = 15
		versPtrOffset = 16

		flagsEndianMask   = 0x1
		flagsEndianLittle = 0x0
		flagsEndianBig    = 0x1

		flagsVersionMask = 0x2
		flagsVersionPtr  = 0x0
		flagsVersionInl  = 0x2
	)

	var vers, mod string
	var err error

	_ = vers

	flags := header[flagsOffset]
	if flags&flagsVersionMask == flagsVersionInl {
		vers, addr, err = decodeString(addr + buildInfoHeaderSize)
		if err != nil {
			slog.Warn(err.Error())
			return nil
		}
		mod, _, err = decodeString(addr)
		if err != nil {
			slog.Warn(err.Error())
			return nil
		}
	} else {
		slog.Warn("wasm buildinfo parse was not supported on go<1.18")
		return nil
	}

	if len(mod) >= 33 && mod[len(mod)-17] == '\n' {
		mod = mod[16 : len(mod)-16]
	} else {
		slog.Warn("wasm buildinfo parse mod error")
		return nil
	}
	return nil
}
