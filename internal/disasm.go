package internal

import (
	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	"log/slog"
)

func (k *KnownInfo) Disasm() error {
	slog.Info("Disassemble functions...")

	fns := k.Deps.GetFunctions()

	pb := utils.NewPb(int64(len(fns)), "Disassembling...")

	e, err := disasm.NewExtractor(k.wrapper, k.Size)
	if err != nil {
		return err
	}

	type result struct {
		addr, size uint64
		fn         *entity.Function
	}

	possibles := lo.Flatten(lop.Map(fns, func(fn *entity.Function, index int) []result {
		candidates := e.Extract(fn.Addr, fn.Addr+fn.Size)
		candidates = lo.Filter(candidates, func(p disasm.PossibleStr, _ int) bool {
			return e.AddrIsString(p.Addr, int64(p.Size))
		})
		_ = pb.Add(1)
		return lo.Map(candidates, func(p disasm.PossibleStr, _ int) result {
			return result{
				addr: p.Addr,
				size: p.Size,
				fn:   fn,
			}
		})
	}))

	lo.ForEach(possibles, func(p result, _ int) {
		k.KnownAddr.InsertDisasm(p.addr, p.size, p.fn)
	})

	slog.Info("Disassemble done")

	return nil
}
