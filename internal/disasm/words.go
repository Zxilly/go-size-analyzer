package disasm

type wordKind uint8

const (
	wordUnknown wordKind = iota
	wordConstant
	wordAddress
	wordLoad
)

type trackedWord struct {
	kind  wordKind
	value uint64
	age   uint64
}

type memoryWord struct {
	base   int
	offset int64
	word   trackedWord
}

// Track only a short straight-line span. Calls, branches and unknown writes
// invalidate evidence; unrelated instructions cannot extend its lifetime.
type wordTracker struct {
	regs   [32]trackedWord
	stores [16]memoryWord
	clock  uint64
	result map[PossibleStr]struct{}
}

func newWordTracker() wordTracker               { return wordTracker{result: make(map[PossibleStr]struct{})} }
func (s *wordTracker) reset()                   { s.regs = [32]trackedWord{}; s.stores = [16]memoryWord{} }
func (s *wordTracker) valid(w trackedWord) bool { return w.kind != wordUnknown && s.clock-w.age <= 12 }
func (s *wordTracker) get(r int) trackedWord {
	if r < 0 || r >= len(s.regs) || !s.valid(s.regs[r]) {
		return trackedWord{}
	}
	return s.regs[r]
}

func (s *wordTracker) set(r int, w trackedWord) {
	if r < 0 || r >= len(s.regs) {
		return
	}
	for i := range s.stores {
		if s.stores[i].base == r {
			s.stores[i].word = trackedWord{}
		}
	}
	w.age = s.clock
	s.regs[r] = w
}

func (s *wordTracker) pair(ptr, length trackedWord) {
	if !s.valid(ptr) || !s.valid(length) {
		return
	}
	switch {
	case ptr.kind == wordAddress && length.kind == wordConstant && length.value > 0:
		s.result[PossibleStr{Addr: ptr.value, Size: length.value}] = struct{}{}
	case ptr.kind == wordLoad && length.kind == wordLoad && ptr.value <= ^uint64(0)-8 && length.value == ptr.value+8:
		s.result[PossibleStr{Addr: ptr.value, Header: true}] = struct{}{}
	case ptr.kind == wordAddress && length.kind == wordAddress:
		s.result[PossibleStr{Addr: length.value, Header: true, TypeAddr: ptr.value}] = struct{}{}
	}
}

func (s *wordTracker) registers(abi []int) {
	for i := 1; i < len(abi); i++ {
		s.pair(s.get(abi[i-1]), s.get(abi[i]))
	}
}

func (s *wordTracker) store(base int, offset int64, w trackedWord) {
	if base < 0 {
		return
	}
	w.age = s.clock
	index := 0
	for i, item := range s.stores {
		if item.base == base && item.offset == offset {
			index = i
			break
		}
		if item.word.age < s.stores[index].word.age {
			index = i
		}
	}
	s.stores[index] = memoryWord{base: base, offset: offset, word: w}
	for _, item := range s.stores {
		if item.base != base || !s.valid(item.word) {
			continue
		}
		if item.offset == offset-8 {
			s.pair(item.word, w)
		}
		if item.offset == offset+8 {
			s.pair(w, item.word)
		}
	}
}

func (s *wordTracker) candidates() []PossibleStr {
	out := make([]PossibleStr, 0, len(s.result))
	for p := range s.result {
		out = append(out, p)
	}
	return out
}
