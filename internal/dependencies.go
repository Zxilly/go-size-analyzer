package internal

import (
	"runtime/debug"

	"github.com/ZxillyFork/gore"
	"github.com/ZxillyFork/gosym"
	"github.com/ZxillyFork/trie"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

// Dependencies a pres-udo package for the whole binary
type Dependencies struct {
	k *KnownInfo

	TopPkgs entity.PackageMap
	trie    *trie.PathTrie
}

func NewDependencies(k *KnownInfo) *Dependencies {
	return &Dependencies{
		TopPkgs: make(entity.PackageMap),
		k:       k,
		trie:    trie.NewPathTrie(),
	}
}

func (m *Dependencies) GetPackage(name string) (*entity.Package, bool) {
	p := m.trie.Get(name)
	if p == nil {
		return nil, false
	}
	return p.(*entity.Package), true
}

func (m *Dependencies) GetFunctions() []*entity.Function {
	funcs := make([]*entity.Function, 0)
	_ = m.trie.Walk(func(_ string, value any) error {
		p := value.(*entity.Package)
		funcs = append(funcs, p.GetFunctions()...)
		return nil
	})
	return funcs
}

func (m *Dependencies) AddModules(mods []*debug.Module, typ entity.PackageType) {
	for _, mod := range mods {
		old := m.trie.Get(mod.Path)
		if old != nil {
			old.(*entity.Package).DebugMod = mod
			continue
		}
		p := entity.NewPackage()
		p.Name = utils.Deduplicate(mod.Path)
		p.Type = typ
		p.DebugMod = mod
		m.trie.Put(mod.Path, p)
	}
}

func (m *Dependencies) FinishLoad() {
	type pair struct {
		m  entity.PackageMap
		tc *trie.PathTrie
	}

	// load generated packages, they don't have a path
	if m.trie.Value != nil {
		m.TopPkgs[""] = m.trie.Value.(*entity.Package)
	}

	pending := []pair{{m.TopPkgs, m.trie}}

	load := func(packageMap entity.PackageMap, p *trie.PathTrie) {
		for part, nxt := range p.RecursiveDirectChildren() {
			packageMap[part] = nxt.Value.(*entity.Package)
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
}

func (m *Dependencies) Add(gp *gore.Package, typ entity.PackageType, pclntab *gosym.Table) {
	name := utils.UglyGuess(gp.Name)

	p := entity.NewPackageWithGorePackage(gp, name, typ, pclntab)

	// update addrs
	for _, f := range p.GetFunctions() {
		m.k.KnownAddr.InsertPclntab(f.Addr, f.CodeSize, f, entity.GoPclntabMeta{
			FuncName:    utils.Deduplicate(f.Name),
			PackageName: utils.Deduplicate(p.Name),
			Type:        utils.Deduplicate(f.Type),
			Receiver:    utils.Deduplicate(f.Receiver),
			Filepath:    utils.Deduplicate(f.File.FilePath),
		})
	}

	// we need merge since the gore relies on the broken std PackageName() function
	old := m.trie.Get(name)
	if old != nil {
		// merge the old one
		p.Merge(old.(*entity.Package))
	}
	m.trie.Put(name, p)
}
