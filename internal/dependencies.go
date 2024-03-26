package internal

import (
	"debug/gosym"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/goretk/gore"
	"runtime/debug"
	"strings"
)

// Dependencies a pres-udo package for the whole binary
type Dependencies struct {
	link entity.PackageMap
	k    *KnownInfo

	topPkgs entity.PackageMap
}

func NewDependencies(k *KnownInfo) *Dependencies {
	return &Dependencies{
		topPkgs: make(entity.PackageMap),
		link:    make(entity.PackageMap),
		k:       k,
	}
}

func (m *Dependencies) GetPackage(name string) (*entity.Package, bool) {
	p, ok := m.link[name]
	return p, ok
}

func (m *Dependencies) GetFunctions() []*entity.Function {
	funcs := make([]*entity.Function, 0)
	for _, p := range m.topPkgs {
		funcs = append(funcs, p.GetFunctions(true)...)
	}
	return funcs
}

func (m *Dependencies) PushUpUnloadPacakge(modules []*debug.Module) {
	noTypPkgs := make([]*entity.Package, 0)
	for _, module := range modules {
		parts := strings.Split(module.Path, "/")
		if len(parts) == 0 {
			continue
		}
		firstPart := parts[0]
		if _, ok := m.topPkgs[firstPart]; !ok {
			continue // can this happen?
		}
		p := m.topPkgs[firstPart]
		for _, part := range parts[1:] {
			if _, ok := p.SubPackages[part]; !ok {
				goto next
			}
			p = p.SubPackages[part]
		}
		p.Pseudo = true
		p.Name = module.Path

		// after subpackages are loaded, need to determine a type for these
		noTypPkgs = append(noTypPkgs, p)
		// should also update link for this
		m.link[module.Path] = p

	next:
	}

	partMerge := func(part ...string) string {
		return strings.Join(part, "/")
	}

	var expand func(p *entity.Package, part string) (shouldExpand bool, expanded entity.PackageMap)
	expand = func(p *entity.Package, part string) (bool, entity.PackageMap) {
		newSubs := make(entity.PackageMap)
		for subPart, subPackage := range p.SubPackages {
			shouldExpand, expanded := expand(subPackage, subPart)
			if !shouldExpand {
				newSubs[subPart] = subPackage
			} else {
				for ek, ev := range expanded {
					newSubs[partMerge(subPart, ek)] = ev
				}
			}
		}

		if p.Loaded || p.Pseudo {
			p.SubPackages = newSubs
			return false, nil
		} else {
			return true, newSubs
		}
	}

	newPackages := make(entity.PackageMap)
	for part, p := range m.topPkgs {
		shouldExpand, expanded := expand(p, part)
		if shouldExpand {
			for k, v := range expanded {
				newPackages[partMerge(part, k)] = v
			}
		} else {
			newPackages[part] = p
		}
	}

	// We can load a type now
	for _, p := range noTypPkgs {
		if len(p.SubPackages) > 0 {
			typ, err := p.SubPackages.GetType()
			if err != nil {
				panic(fmt.Errorf("package %s has %s", p.Name, err))
			}
			if p.Type == "" {
				p.Type = typ
			} else if p.Type != typ {
				panic(fmt.Errorf("package %s has multiple type %s and %s", p.Name, p.Type, typ))
			}
		}
		if p.Type == "" {
			panic(fmt.Errorf("package %s has no type", p.Name))
		}
	}

	m.topPkgs = newPackages
}

func (m *Dependencies) Add(gp *gore.Package, typ entity.PackageType, pclntab *gosym.Table) {
	name := gp.Name
	if typ == entity.PackageTypeVendor {
		name = utils.UglyGuess(gp.Name)
	}

	parts := strings.Split(name, "/")

	if len(parts) == 0 {
		panic("empty package name " + gp.Name)
	}
	var container = m.topPkgs
	for i, p := range parts {
		if i == len(parts)-1 {
			break
		}

		if _, ok := container[p]; !ok {
			container[p] = entity.NewPackage()
		}
		container = container[p].SubPackages
	}

	id := parts[len(parts)-1]

	p := entity.NewPackageWithGorePackage(gp, name, typ, pclntab)

	// update addrs
	for _, f := range p.GetFunctions(false) {
		m.k.KnownAddr.InsertPclntab(f.Addr, f.Size, f, entity.GoPclntabMeta{
			FuncName:    utils.Deduplicate(f.Name),
			PackageName: utils.Deduplicate(p.Name),
			Type:        utils.Deduplicate(f.Type),
			Receiver:    utils.Deduplicate(f.Receiver),
			Filepath:    utils.Deduplicate(f.File.FilePath),
		})
	}

	p.Merge(container[id])

	container[id] = p
	// also update the link
	m.link[name] = p
}
