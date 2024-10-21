package knowninfo

import (
	"fmt"
	"log/slog"

	"github.com/ZxillyFork/gore"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

type VersionFlag struct {
	Leq118 bool
	Meq120 bool
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

func (k *KnownInfo) LoadGoreInfo(f *gore.GoFile) error {
	err := k.LoadPackages(f)
	if err != nil {
		return err
	}

	slog.Info("Loading version flag and meta info...")
	k.VersionFlag = UpdateVersionFlag(f)
	slog.Info("Loaded version flag")
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
	}
}

func (k *KnownInfo) RequireModInfo() error {
	if k.BuildInfo == nil {
		return fmt.Errorf("no build info")
	}
	return nil
}

func (k *KnownInfo) convertAddr(addr uint64) uint64 {
	if w, ok := k.Wrapper.(*wrapper.MachoWrapper); ok {
		return w.SlidePointer(addr)
	}
	return addr
}
