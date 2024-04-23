package idxtable

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

type Table[T1 any] interface {
	Get(id int64) (T1, error)
	Claim(id int64, d T1) error
	ClaimDynamic(d T1) (int64, error)
	ClaimRange(start, size int64, d T1) error
	ClaimSize(size int64, d T1) error
	Release(id int64) error
	Update(id int64, d T1) error

	Iterate() *Iterator[T1]
	IterateFree() *Iterator[T1]

	Count() int
	Has(id int64) bool

	IsFree(id int64) bool
	FindFree() (int64, error)
	FindFreeRange(min, size int64) (map[int64]T1, error)
	FindFreeSize(size int64) (map[int64]T1, error)

	GetAll() map[int64]T1
}

type ValidationFn func(id int64) error

func NewTable[T1 any](s int64, initEntries map[int64]T1, v ValidationFn) (Table[T1], error) {
	r := &table[T1]{
		m:          new(sync.RWMutex),
		table:      map[int64]T1{},
		size:       s,
		validateFn: v,
	}

	var errm error
	for id, d := range initEntries {
		id := id
		if err := r.add(id, d, true); err != nil {
			errm = errors.Join(errm, err)
		}
	}

	return r, errm
}

type table[T1 any] struct {
	m          *sync.RWMutex
	table      map[int64]T1
	size       int64
	validateFn ValidationFn
}

func (r *table[T1]) validate(id int64, init bool) error {
	if id > r.size-1 {
		return fmt.Errorf("id %d is bigger then max allowed entries: %d", id, r.size-1)

	}
	if r.validateFn != nil && !init {
		if err := r.validateFn(id); err != nil {
			return err
		}
	}
	return nil
}

func (r *table[T1]) Get(id int64) (T1, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	var d T1

	if err := r.validate(id, false); err != nil {
		return d, err
	}

	d, ok := r.table[id]
	if !ok {
		return d, fmt.Errorf("no match found for: %v", id)
	}
	return d, nil
}

func (r *table[T1]) Claim(id int64, d T1) error {
	r.m.Lock()
	defer r.m.Unlock()

	return r.add(id, d, false)
}

func (r *table[T1]) ClaimDynamic(d T1) (int64, error) {
	r.m.Lock()
	defer r.m.Unlock()

	free := r.iterateFree()
	if free.Next() {
		if err := r.add(free.ID(), d, false); err != nil {
			return 0, err
		}
		return free.ID(), nil
	}
	return 0, fmt.Errorf("no free entry found")
}

func (r *table[T1]) ClaimRange(start, size int64, d T1) error {
	r.m.Lock()
	defer r.m.Unlock()

	entries, err := r.findFreeRange(start, size)
	if err != nil {
		return err
	}
	for id := range entries {
		// getting an error is unlikely as we have a lock
		if err := r.add(id, d, false); err != nil {
			return err
		}
	}
	return nil
}

func (r *table[T1]) ClaimSize(size int64, d T1) error {
	r.m.Lock()
	defer r.m.Unlock()

	entries, err := r.findFreeSize(size)
	if err != nil {
		return err
	}
	for id, entry := range entries {
		id := id
		// getting an error is unlikely as we have a lock
		if err := r.add(id, entry, false); err != nil {
			return err
		}
	}
	return nil
}

func (r *table[T1]) Release(id int64) error {
	r.m.Lock()
	defer r.m.Unlock()

	return r.delete(id)
}

func (r *table[T1]) Update(id int64, d T1) error {
	r.m.Lock()
	defer r.m.Unlock()

	return r.update(id, d)
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

func (r *table[T1]) Count() int {
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

func (r *table[T1]) FindFreeRange(start, size int64) (map[int64]T1, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.findFreeRange(start, size)
}

func (r *table[T1]) findFreeRange(start, size int64) (map[int64]T1, error) {
	end := start + size - 1

	if start > r.size-1 {
		return nil, fmt.Errorf("start %d is bigger then max allowed entries: %d", start, r.size)
	}
	if end > r.size-1 {
		return nil, fmt.Errorf("end %d is bigger then max allowed entries: %d", end, r.size)
	}

	entries := map[int64]T1{}
	free := r.iterateFree()
	for free.Next() {
		if free.ID() < start {
			continue
		}
		switch {
		case free.ID() == start:
			entries[free.ID()] = free.Value()
		case free.ID() > start && free.ID() < end:
			if !free.IsConsecutive() {
				return nil, fmt.Errorf("entry %d in use in range: start: %d, end %d", free.ID(), start, end)
			}
			entries[free.ID()] = free.Value()
		default:
			entries[free.ID()] = free.Value()
			return entries, nil
		}
	}
	return nil, fmt.Errorf("could not find free range that fit in start %d, size %d", start, size)
}

func (r *table[T1]) FindFreeSize(size int64) (map[int64]T1, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.findFreeSize(size)
}

func (r *table[T1]) findFreeSize(size int64) (map[int64]T1, error) {
	if size > r.size {
		return nil, fmt.Errorf("size %d is bigger then max allowed entries: %d", size, r.size)
	}
	entries := map[int64]T1{}
	free := r.iterateFree()
	i := int64(0)
	for free.Next() {
		i++
		entries[free.ID()] = free.Value()
		if i > size-1 {
			return entries, nil
		}
	}
	return nil, fmt.Errorf("could not find free entries that fit in size %d", size)
}

func (r *table[T1]) add(id int64, d T1, init bool) error {
	if err := r.validate(id, init); err != nil {
		return err
	}
	if !r.isFree(id) {
		return fmt.Errorf("entry %d already exists", id)
	}
	r.table[id] = d
	return nil
}

func (r *table[T1]) update(id int64, d T1) error {
	if err := r.validate(id, false); err != nil {
		return err
	}
	if r.isFree(id) {
		return fmt.Errorf("entry %d not found", id)
	}
	r.table[id] = d
	return nil
}

func (r *table[T1]) delete(id int64) error {
	if err := r.validate(id, false); err != nil {
		return err
	}
	delete(r.table, id)
	return nil
}

func (r *table[T1]) GetAll() map[int64]T1 {
	r.m.RLock()
	defer r.m.RUnlock()

	entries := make(map[int64]T1, len(r.table))

	iter := r.Iterate()
	for iter.Next() {
		entries[iter.ID()] = iter.Value()
	}
	return entries
}
