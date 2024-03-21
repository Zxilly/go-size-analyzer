package printer

import (
	"cmp"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/dustin/go-humanize"
	"golang.org/x/exp/maps"
	"slices"
)

func PrintResult(r *internal.Result) {
	fmt.Println("Binary: ", r.Name)
	fmt.Println("Size: ", humanize.Bytes(r.Size))

	fmt.Println("Packages:")

	pkgs := maps.Values(r.Packages)

	slices.SortFunc(pkgs, func(a, b *internal.Package) int {
		return -cmp.Compare(a.Size, b.Size)
	})

	for _, p := range pkgs {
		fmt.Println("  ", p.Name, humanize.Bytes(p.Size))
	}
}
