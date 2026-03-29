package knowninfo

import (
	"encoding/binary"
	"log/slog"
	"strings"

	"github.com/ZxillyFork/gore"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

// ptrSizeAndOrder returns the pointer size in bytes and byte order for the
// given Go architecture string.
func ptrSizeAndOrder(goarch string) (int, binary.ByteOrder) {
	ptrSize := 8
	switch goarch {
	case "386", "arm", "armbe", "mips", "mipsle", "mips64p32":
		ptrSize = 4
	}

	var order binary.ByteOrder = binary.LittleEndian
	switch goarch {
	case "s390x", "ppc64":
		order = binary.BigEndian
	}

	return ptrSize, order
}

type VersionFlag struct {
	Leq118 bool
	Meq120 bool
	Meq125 bool // Go 1.25+ changed wasm pclntab to store PC_F instead of full PC
}

// PclntabMeta holds addresses of pclntab sub-tables captured from the symbol table.
type PclntabMeta struct {
	FuncnametabAddr uint64
	CutabAddr       uint64
	FiletabAddr     uint64
	PctabAddr       uint64
	FunctabAddr     uint64
	PclntabEnd      uint64 // runtime.epclntab
}

type KnownInfo struct {
	Size      uint64
	BuildInfo *gore.BuildInfo
	Sects     *entity.Store
	Deps      *Dependencies
	KnownAddr *entity.KnownAddr

	GoStringSymbol *entity.AddrPos

	Coverage entity.AddrCoverage

	Gore        *gore.GoFile
	PClnTabAddr uint64
	Wrapper     wrapper.RawFileWrapper

	VersionFlag VersionFlag

	HasDWARF bool

	// PclntabSyms holds addresses of pclntab sub-tables captured from the symbol table.
	PclntabSyms PclntabMeta
}

func (k *KnownInfo) LoadGoreInfo(f *gore.GoFile, isWasm bool) error {
	slog.Info("Loading version flag")
	k.VersionFlag = UpdateVersionFlag(f)
	slog.Info("Loaded version flag")

	err := k.LoadPackages(f, isWasm)
	if err != nil {
		return err
	}

	slog.Info("Loading meta info...")
	k.PClnTabAddr = f.GetPCLNTableAddr()
	slog.Info("Loaded meta info")

	return nil
}

func UpdateVersionFlag(f *gore.GoFile) VersionFlag {
	ver, err := f.GetCompilerVersion()
	if err != nil {
		// if we can't get build info, we assume it's go1.20 plus
		return VersionFlag{
			Leq118: false,
			Meq120: true,
		}
	}

	return VersionFlag{
		Leq118: gore.GoVersionCompare(ver.Name, "go1.18.10") <= 0,
		Meq120: gore.GoVersionCompare(ver.Name, "go1.20rc1") >= 0,
		Meq125: gore.GoVersionCompare(ver.Name, "go1.25rc1") >= 0,
	}
}

// isMainModulePackage checks if a package path belongs to the main module
// by comparing against BuildInfo.ModInfo.Main.Path.
func (k *KnownInfo) isMainModulePackage(pkgName string) bool {
	if k.BuildInfo == nil || k.BuildInfo.ModInfo == nil {
		return false
	}
	mainPath := k.BuildInfo.ModInfo.Main.Path
	if mainPath == "" {
		return false
	}
	return pkgName == mainPath || strings.HasPrefix(pkgName, mainPath+"/")
}

func (k *KnownInfo) convertAddr(addr uint64) uint64 {
	if w, ok := k.Wrapper.(*wrapper.MachoWrapper); ok {
		return w.SlidePointer(addr)
	}
	return addr
}
