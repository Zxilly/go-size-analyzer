package knowninfo

import (
	"errors"
	"log/slog"
	"runtime/debug"

	"github.com/ZxillyFork/gore"
	"github.com/ZxillyFork/gosym"
	"github.com/ZxillyFork/trie"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

// Dependencies a pseudo package for the whole binary
type Dependencies struct {
	k *KnownInfo

	TopPkgs entity.PackageMap
	Trie    *trie.PathTrie[*entity.Package]
}

func NewDependencies(k *KnownInfo) *Dependencies {
	return &Dependencies{
		TopPkgs: make(entity.PackageMap),
		k:       k,
		Trie:    trie.NewPathTrie[*entity.Package](),
	}
}

func (m *Dependencies) GetPackage(name string) (*entity.Package, bool) {
	p := m.Trie.Get(name)
	if p == nil {
		return nil, false
	}
	return p, true
}

func (m *Dependencies) Functions(yield func(*entity.Function) bool) {
	_ = m.Trie.Walk(func(_ string, p *entity.Package) error {
		for f := range p.Functions {
			if !yield(f) {
				return errors.New("stop walk")
			}
		}
		return nil
	})
}

func (m *Dependencies) AddModules(mods []*debug.Module, typ entity.PackageType) {
	for _, mod := range mods {
		old := m.Trie.Get(mod.Path)
		if old != nil {
			continue
		}
		p := entity.NewPackage()
		p.Name = utils.Deduplicate(mod.Path)
		p.Type = typ
		m.Trie.Put(mod.Path, p)
	}
}

func (m *Dependencies) FinishLoad() {
	type pair struct {
		m  entity.PackageMap
		tc *trie.PathTrie[*entity.Package]
	}

	// load generated packages, they don't have a path
	if m.Trie.Value != nil {
		m.TopPkgs[""] = *m.Trie.Value
	}

	pending := []pair{{m.TopPkgs, m.Trie}}

	load := func(packageMap entity.PackageMap, p *trie.PathTrie[*entity.Package]) {
		for part, nxt := range p.RecursiveDirectChildren() {
			packageMap[part] = *nxt.Value
			cc := nxt.RecursiveDirectChildren()
			if len(cc) > 0 {
				pending = append(pending, pair{packageMap[part].SubPackages, nxt})
			}
		}
	}

	for len(pending) > 0 {
		p := pending[0]
		pending = pending[1:]
		load(p.m, p.tc)
	}

	// clear caches
	_ = m.Trie.Walk(func(_ string, value *entity.Package) error {
		value.ClearCache()
		return nil
	})
}

func (m *Dependencies) AddFromPclntab(gp *gore.Package, typ entity.PackageType, pclntab *gosym.Table) {
	name := utils.UglyGuess(gp.Name)

	p := entity.NewPackageWithGorePackage(gp, name, typ, pclntab)

	// update addrs
	for f := range p.Functions {
		m.k.KnownAddr.InsertTextFromPclnTab(f.Addr, f.CodeSize, f)
	}

	// we need merge since the gore relies on the broken std PackageName() function
	old := m.Trie.Get(name)
	if old != nil {
		// merge the old one
		p.Merge(old)
	}
	m.Trie.Put(name, p)
}

func (k *KnownInfo) LoadPackages(f *gore.GoFile) error {
	slog.Info("Loading packages...")

	pkgs := NewDependencies(k)
	k.Deps = pkgs

	pclntab, err := f.PCLNTab()
	if err != nil {
		return err
	}

	self, err := f.GetPackages()
	if err != nil {
		return err
	}
	for _, p := range self {
		pkgs.AddFromPclntab(p, entity.PackageTypeMain, pclntab)
	}

	grStd, _ := f.GetSTDLib()
	for _, p := range grStd {
		pkgs.AddFromPclntab(p, entity.PackageTypeStd, pclntab)
	}

	grVendor, _ := f.GetVendors()
	for _, p := range grVendor {
		pkgs.AddFromPclntab(p, entity.PackageTypeVendor, pclntab)
	}

	grGenerated, _ := f.GetGeneratedPackages()
	for _, p := range grGenerated {
		pkgs.AddFromPclntab(p, entity.PackageTypeGenerated, pclntab)
	}

	grUnknown, _ := f.GetUnknown()
	for _, p := range grUnknown {
		pkgs.AddFromPclntab(p, entity.PackageTypeUnknown, pclntab)
	}

	if err = k.RequireModInfo(); err == nil {
		pkgs.AddModules(k.BuildInfo.ModInfo.Deps, entity.PackageTypeVendor)
		pkgs.AddModules([]*debug.Module{&k.BuildInfo.ModInfo.Main}, entity.PackageTypeVendor)
	}

	slog.Info("Loaded packages done")

	return nil
}
