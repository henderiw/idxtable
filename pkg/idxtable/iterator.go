package idxtable

type Iterator[T1 any] struct {
	current int64
	keys    []int64
	table   map[int64]Entry[T1]
}

func (r *Iterator[T1]) Value() Entry[T1] {
	return r.table[r.keys[r.current]]
}

func (r *Iterator[T1]) ID() int64 {
	return r.keys[r.current]
}

func (r *Iterator[T1]) Next() bool {
	r.current++
	return r.current < int64(len(r.keys))
}

func (r *Iterator[T1]) IsConsecutive() bool {
	//fmt.Println("id:", r.current, "prevId", r.keys[r.current-1], "currId", r.keys[r.current]-1)
	if r.current < 1 {
		return false
	}
	return r.keys[r.current-1] == r.keys[r.current]-1
}
