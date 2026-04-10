package entity

import (
	"reflect"
	"testing"
)

func TestSectionStore_AssertSize(t *testing.T) {
	type fields struct {
		Sections map[string]*Section
	}
	type args struct {
		size uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "AssertSize works correctly",
			fields: fields{
				Sections: map[string]*Section{
					"section1": {
						FileSize: 10,
					},
				},
			},
			args: args{
				size: 20,
			},
			wantErr: false,
		},
		{
			name: "AssertSize throws error",
			fields: fields{
				Sections: map[string]*Section{
					"section1": {
						FileSize: 10,
					},
					"section2": {
						FileSize: 15,
					},
				},
			},
			args: args{
				size: 20,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				Sections: tt.fields.Sections,
			}
			if err := s.AssertSize(tt.args.size); (err != nil) != tt.wantErr {
				t.Errorf("AssertSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSectionStore_FindSection(t *testing.T) {
	type fields struct {
		Sections map[string]*Section
	}
	type args struct {
		addr uint64
		size uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Section
	}{
		{
			name: "FindSection failed",
			fields: fields{
				Sections: map[string]*Section{
					"section1": {
						Debug:   true,
						Addr:    100,
						AddrEnd: 200,
					},
				},
			},
			args: args{
				addr: 150,
			},
			want: nil,
		},
		{
			name: "FindSection ignores OnlyInMemory section",
			fields: fields{
				Sections: map[string]*Section{
					".bss": {
						OnlyInMemory: true,
						ContentType:  SectionContentData,
						Addr:         0x1000,
						AddrEnd:      0x3000000,
					},
				},
			},
			args: args{
				addr: 0x1000,
				size: 0x100,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				Sections: tt.fields.Sections,
			}
			s.BuildCache()
			if got := s.FindSection(tt.args.addr, tt.args.size); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindSection() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOnlyInMemorySymbolNotCounted is a regression test for
// https://github.com/Zxilly/go-size-analyzer/issues/522.
// Large BSS symbols (e.g. var a [1<<25]byte) must not be attributed to any
// package because they live only in memory and take no space in the binary.
func TestOnlyInMemorySymbolNotCounted(t *testing.T) {
	// ELF case: BSS is a dedicated section with OnlyInMemory=true.
	t.Run("ELF BSS section", func(t *testing.T) {
		bssSection := &Section{
			Name:         ".bss",
			Size:         0x2000000, // 32 MB
			Addr:         0x1000,
			AddrEnd:      0x2001000,
			OnlyInMemory: true,
			ContentType:  SectionContentData,
		}
		store := &Store{
			Sections: map[string]*Section{".bss": bssSection},
		}
		store.BuildCache()

		ka := NewKnownAddr(store)
		sym := NewSymbol("main.a", 0x1000, 0x2000000, AddrTypeData)
		pkg := NewPackage()
		pkg.Name = "main"

		if ap := ka.InsertSymbol(sym, pkg); ap != nil {
			t.Errorf("InsertSymbol returned non-nil for a BSS (OnlyInMemory) symbol: %v", ap)
		}
		if store.IsData(0x1000, 0x100) {
			t.Error("IsData() returned true for an address inside an OnlyInMemory section")
		}
	})

	// PE case: BSS is the tail of .data where VirtualSize >> FileSize (raw size).
	// The 32 MB zero-initialized array lives in the virtual-only part of .data.
	t.Run("PE .data section with BSS tail", func(t *testing.T) {
		const (
			baseAddr    = 0x14011a000
			fileSize    = 0x6E00     // 28 KB raw data
			virtualSize = 0x2000000  // 32 MB virtual (includes BSS)
		)
		dataSection := &Section{
			Name:         ".data",
			Size:         virtualSize,
			FileSize:     fileSize,
			Addr:         baseAddr,
			AddrEnd:      baseAddr + virtualSize,
			OnlyInMemory: false,
			ContentType:  SectionContentData,
		}
		store := &Store{
			Sections: map[string]*Section{".data": dataSection},
		}
		store.BuildCache()

		ka := NewKnownAddr(store)
		pkg := NewPackage()
		pkg.Name = "main"

		// Symbol inside the BSS tail (beyond FileSize) must not be counted.
		bssAddr := uint64(baseAddr + fileSize + 0x100)
		sym := NewSymbol("main.a", bssAddr, 0x2000000-fileSize-0x100, AddrTypeData)
		if ap := ka.InsertSymbol(sym, pkg); ap != nil {
			t.Errorf("InsertSymbol returned non-nil for a PE BSS symbol: %v", ap)
		}
		if store.IsData(bssAddr, 0x100) {
			t.Error("IsData() returned true for a PE BSS address (beyond FileSize)")
		}

		// Symbol inside the file-backed part must still be counted.
		fileAddr := uint64(baseAddr + 0x10)
		symFile := NewSymbol("runtime.something", fileAddr, 0x10, AddrTypeData)
		if ap := ka.InsertSymbol(symFile, pkg); ap == nil {
			t.Error("InsertSymbol returned nil for a file-backed PE symbol")
		}
		if !store.IsData(fileAddr, 0x10) {
			t.Error("IsData() returned false for a file-backed PE address")
		}
	})
}
