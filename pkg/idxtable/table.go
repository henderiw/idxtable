package idxtable

import (
	"fmt"
	"sort"
	"sync"
)

type Table[T1 any] interface {
	Get(id int64) (Entry[T1], error)
	Claim(id int64, d T1) error
	ClaimDynamic(d T1) (Entry[T1], error)
	ClaimRange(start, size int64, d T1) error
	ClaimSize(size int64, d T1) (Entries[T1], error)
	Release(id int64) error
	Update(id int64, d T1) error

	Iterate() *Iterator[T1]
	IterateFree() *Iterator[T1]

	Size() int
	Has(id int64) bool

	IsFree(id int64) bool
	FindFree() (int64, error)
	FindFreeRange(min, size int64) ([]int64, error)
	FindFreeSize(size int64) ([]int64, error)

	GetAll() Entries[T1]
}

func NewTable[T1 any](size int64) Table[T1] {
	r := &table[T1]{
		m:     new(sync.RWMutex),
		table: map[int64]Entry[T1]{},
		size:  size,
	}

	return r
}

type table[T1 any] struct {
	m     *sync.RWMutex
	table map[int64]Entry[T1]
	size  int64
}

func (r *table[T1]) validate(id int64) error {
	if id > r.size-1 {
		return fmt.Errorf("id %d is bigger then max allowed entries: %d", id, r.size-1)
	}
	if id < 0 {
		return fmt.Errorf("id %d is smaller then min allowed entries: %d", id, 0)
	}
	return nil
}

func (r *table[T1]) Get(id int64) (Entry[T1], error) {
	r.m.RLock()
	defer r.m.RUnlock()

	if err := r.validate(id); err != nil {
		return nil, err
	}

	e, ok := r.table[id]
	if !ok {
		return nil, fmt.Errorf("no entry found for: %d", id)
	}
	return e, nil
}

func (r *table[T1]) Claim(id int64, d T1) error {
	r.m.Lock()
	defer r.m.Unlock()

	return r.add(NewEntry(id, d))
}

func (r *table[T1]) ClaimDynamic(d T1) (Entry[T1], error) {
	r.m.Lock()
	defer r.m.Unlock()

	free := r.iterateFree()
	if free.Next() {
		e := NewEntry(free.ID(), d)
		if err := r.add(e); err != nil {
			return nil, err
		}
		return e, nil
	}
	return nil, fmt.Errorf("no free entry found")
}

func (r *table[T1]) ClaimRange(start, size int64, d T1) error {
	r.m.Lock()
	defer r.m.Unlock()

	ids, err := r.findFreeRange(start, size)
	if err != nil {
		return err
	}
	for _, id := range ids {
		id := id
		// getting an error is unlikely as we have a lock
		if err := r.add(NewEntry(id, d)); err != nil {
			return err
		}
	}
	return nil
}

func (r *table[T1]) ClaimSize(size int64, d T1) (Entries[T1], error) {
	r.m.Lock()
	defer r.m.Unlock()

	ids, err := r.findFreeSize(size)
	if err != nil {
		return nil, err
	}
	entries := Entries[T1]{}
	for _, id := range ids {
		id := id
		e := NewEntry(id, d)
		// getting an error is unlikely as we have a lock
		if err := r.add(e); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (r *table[T1]) Release(id int64) error {
	r.m.Lock()
	defer r.m.Unlock()

	return r.delete(id)
}

func (r *table[T1]) Update(id int64, d T1) error {
	r.m.Lock()
	defer r.m.Unlock()

	return r.update(NewEntry(id, d))
}

func (r *table[T1]) Iterate() *Iterator[T1] {
	r.m.RLock()
	defer r.m.RUnlock()

	return r.iterate()
}

func (r *table[T1]) iterate() *Iterator[T1] {
	keys := make([]int64, 0, len(r.table))
	for key := range r.table {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i int, j int) bool {
		return keys[i] < keys[j]
	})

	return &Iterator[T1]{current: -1, keys: keys, table: r.table}
}

func (r *table[T1]) IterateFree() *Iterator[T1] {
	r.m.RLock()
	defer r.m.RUnlock()

	return r.iterateFree()
}

func (r *table[T1]) iterateFree() *Iterator[T1] {
	var keys []int64
	table := map[int64]T1{}

	var d T1

	for id := int64(0); id < r.size; id++ {
		_, exists := r.table[id]
		if !exists {
			keys = append(keys, id)
			table[id] = d
		}
	}
	sort.Slice(keys, func(i int, j int) bool {
		return keys[i] < keys[j]
	})

	return &Iterator[T1]{current: -1, keys: keys, table: r.table}
}

func (r *table[T1]) Size() int {
	r.m.RLock()
	defer r.m.RUnlock()

	return len(r.table)
}
func (r *table[T1]) Has(id int64) bool {
	r.m.RLock()
	defer r.m.RUnlock()

	_, ok := r.table[id]
	return ok
}

func (r *table[T1]) IsFree(id int64) bool {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.isFree(id)
}

func (r *table[T1]) isFree(id int64) bool {
	_, ok := r.table[id]
	return !ok
}

func (r *table[T1]) FindFree() (int64, error) {
	free := r.IterateFree()

	if free.Next() {
		return free.ID(), nil
	}
	return 0, fmt.Errorf("no free entry found")
}

func (r *table[T1]) FindFreeRange(start, size int64) ([]int64, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.findFreeRange(start, size)
}

func (r *table[T1]) findFreeRange(start, size int64) ([]int64, error) {
	end := start + size - 1

	if start > r.size-1 {
		return nil, fmt.Errorf("start %d is bigger then max allowed entries: %d", start, r.size)
	}
	if end > r.size-1 {
		return nil, fmt.Errorf("end %d is bigger then max allowed entries: %d", end, r.size)
	}

	entries := []int64{}
	free := r.iterateFree()
	for free.Next() {
		if free.ID() < start {
			continue
		}
		switch {
		case free.ID() == start:
			entries = append(entries, free.ID())
		case free.ID() > start && free.ID() < end:
			if !free.IsConsecutive() {
				return nil, fmt.Errorf("entry %d in use in range: start: %d, end %d", free.ID(), start, end)
			}
			entries = append(entries, free.ID())
		default:
			entries = append(entries, free.ID())
			return entries, nil
		}
	}
	return nil, fmt.Errorf("could not find free range that fit in start %d, size %d", start, size)
}

func (r *table[T1]) FindFreeSize(size int64) ([]int64, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.findFreeSize(size)
}

func (r *table[T1]) findFreeSize(size int64) ([]int64, error) {
	if size > r.size {
		return nil, fmt.Errorf("size %d is bigger then max allowed entries: %d", size, r.size)
	}
	entries := []int64{}
	free := r.iterateFree()
	i := int64(0)
	for free.Next() {
		i++
		entries = append(entries, free.ID())
		if i > size-1 {
			return entries, nil
		}
	}
	return nil, fmt.Errorf("could not find free entries that fit in size %d", size)
}

func (r *table[T1]) add(e Entry[T1]) error {
	if err := r.validate(e.ID()); err != nil {
		return err
	}
	if !r.isFree(e.ID()) {
		return fmt.Errorf("entry %d already exists", e.ID())
	}
	r.table[e.ID()] = e
	return nil
}

func (r *table[T1]) update(e Entry[T1]) error {
	if err := r.validate(e.ID()); err != nil {
		return err
	}
	if r.isFree(e.ID()) {
		return fmt.Errorf("entry %d not created", e.ID())
	}
	r.table[e.ID()] = e
	return nil
}

func (r *table[T1]) delete(id int64) error {
	if err := r.validate(id); err != nil {
		return err
	}
	delete(r.table, id)
	return nil
}

func (r *table[T1]) GetAll() Entries[T1] {
	r.m.RLock()
	defer r.m.RUnlock()

	entries := make([]Entry[T1], len(r.table))

	iter := r.Iterate()
	for iter.Next() {
		entries = append(entries, iter.Value())
	}
	return entries
}
