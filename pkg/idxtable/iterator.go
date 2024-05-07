package idxtable

type Iterator[T1 any] struct {
	current uint64
	keys    []uint64
	table   map[uint64]Entry[T1]
}

func (r *Iterator[T1]) Value() Entry[T1] {
	return r.table[r.keys[r.current]]
}

func (r *Iterator[T1]) ID() uint64 {
	return r.keys[r.current]
}

func (r *Iterator[T1]) Next() bool {
	r.current++
	return r.current < uint64(len(r.keys))
}

func (r *Iterator[T1]) IsConsecutive() bool {
	//fmt.Println("id:", r.current, "prevId", r.keys[r.current-1], "currId", r.keys[r.current]-1)
	if r.current < 1 {
		return false
	}
	return r.keys[r.current-1] == r.keys[r.current]-1
}
