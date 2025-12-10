package knowninfo

import (
	"log/slog"

	"github.com/ZxillyFork/gore"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

type VersionFlag struct {
	Leq118 bool
	Meq120 bool
	Meq125 bool // Go 1.25+ changed wasm pclntab to store PC_F instead of full PC
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

func (k *KnownInfo) convertAddr(addr uint64) uint64 {
	if w, ok := k.Wrapper.(*wrapper.MachoWrapper); ok {
		return w.SlidePointer(addr)
	}
	return addr
}
