package wrapper

import "strconv"

// sectionNames allocates stable, unique names while remembering each base's
// next suffix. Explicit names such as "data#2" occupy the same namespace.
type sectionNames struct {
	used map[string]bool
	next map[string]int
}

func (s *sectionNames) unique(base string) string {
	if s.used == nil {
		s.used = make(map[string]bool)
		s.next = make(map[string]int)
	}
	name := base
	if s.used[name] {
		n := max(2, s.next[base])
		for {
			name = base + "#" + strconv.Itoa(n)
			n++
			if !s.used[name] {
				break
			}
		}
		s.next[base] = n
	}
	s.used[name] = true
	return name
}
