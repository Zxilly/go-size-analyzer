package internal

import (
	"errors"
	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	"log/slog"
)

func (k *KnownInfo) Disasm() error {
	fns := k.Deps.GetFunctions()

	e, err := disasm.NewExtractor(k.wrapper, k.Size)
	if err != nil {
		if errors.Is(err, disasm.ErrArchNotSupported) {
			slog.Warn("Warning: disassembler not supported for this architecture")
			return nil
		}
		return err
	}

	type result struct {
		addr, size uint64
		fn         *entity.Function
	}

	slog.Info("Disassemble functions...")

	possibles := lo.Flatten(lop.Map(fns, func(fn *entity.Function, index int) []result {
		candidates := e.Extract(fn.Addr, fn.Addr+fn.Size)
		candidates = lo.Filter(candidates, func(p disasm.PossibleStr, _ int) bool {
			if p.Size <= 2 {
				return false
			}

			_, ok := e.LoadAddrString(p.Addr, int64(p.Size))
			return ok
		})
		return lo.Map(candidates, func(p disasm.PossibleStr, _ int) result {
			return result{
				addr: p.Addr,
				size: p.Size,
				fn:   fn,
			}
		})
	}))

	lo.ForEach(possibles, func(p result, _ int) {
		s, _ := e.LoadAddrString(p.addr, int64(p.size))
		k.KnownAddr.InsertDisasm(p.addr, p.size, p.fn, entity.DisasmMeta{Value: s})
	})

	slog.Info("Disassemble done")

	return nil
}
