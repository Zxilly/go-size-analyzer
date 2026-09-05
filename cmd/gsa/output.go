//go:build !wasm

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

// Validate all paths before creating even a temporary output. SameFile also
// catches hard links and aliases with different spelling.
func validateOutputs(specs []outputSpec) error {
	var inputs []os.FileInfo
	for _, path := range []string{Options.Binary, Options.DiffTarget} {
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		inputs = append(inputs, info)
	}
	var outputs []os.FileInfo
	seen := make(map[string]bool)
	for i := range specs {
		s := &specs[i]
		if s.path == "" {
			continue
		}
		if s.path == "-" {
			s.path = ""
			s.writer = utils.SyncStdout
			continue
		}
		path, err := filepath.Abs(s.path)
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("output %s is not a regular file", path)
			}
			for _, input := range inputs {
				if os.SameFile(info, input) {
					return fmt.Errorf("output %s refers to an input file", path)
				}
			}
			for _, output := range outputs {
				if os.SameFile(info, output) {
					return fmt.Errorf("duplicate output file %s", path)
				}
			}
			outputs = append(outputs, info)
			path, err = filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		} else {
			// Resolve parent aliases even when the final file does not exist yet.
			dir, err := filepath.EvalSymlinks(filepath.Dir(path))
			if err != nil {
				return err
			}
			path = filepath.Join(dir, filepath.Base(path))
		}
		key := path
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			return fmt.Errorf("duplicate output path %s", path)
		}
		seen[key] = true
		s.path = path
	}
	return nil
}

func prepareOutputs(specs []outputSpec) error {
	if err := validateOutputs(specs); err != nil {
		return err
	}
	for i := range specs {
		s := &specs[i]
		if s.path == "" {
			continue
		}
		f, err := os.CreateTemp(filepath.Dir(s.path), ".gsa-*")
		if err != nil {
			closeAll(specs)
			return err
		}
		s.file = f
		s.writer = f
		s.temp = f.Name()
		if info, err := os.Stat(s.path); err == nil {
			if err = f.Chmod(info.Mode().Perm()); err != nil {
				closeAll(specs)
				return err
			}
		}
	}
	return nil
}

// All rendering must succeed before replacing any destination. Each rename
// replaces one complete report; multiple files are not a filesystem transaction.
func commitOutputs(specs []outputSpec) error {
	for i := range specs {
		if f := specs[i].file; f != nil {
			err := f.Close()
			specs[i].file = nil
			if err != nil {
				return err
			}
		}
	}
	for i := range specs {
		s := &specs[i]
		if s.temp == "" {
			continue
		}
		if err := os.Rename(s.temp, s.path); err != nil {
			return fmt.Errorf("replace output %s: %w", s.path, err)
		}
		s.temp = ""
	}
	return nil
}

func closeAll(specs []outputSpec) {
	for i := range specs {
		s := &specs[i]
		if s.file != nil {
			_ = s.file.Close()
			s.file = nil
		}
		if s.temp != "" {
			_ = os.Remove(s.temp)
			s.temp = ""
		}
	}
}
