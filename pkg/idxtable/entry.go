package idxtable

type Entry[T1 any] interface {
	ID()  uint64
	Data() T1
}

type entry[T1 any] struct {
	id   uint64
	data T1
	// table
}
type Entries[T1 any] []Entry[T1]

func (r entry[T1]) ID() uint64        { return r.id }
func (r entry[T1]) Data() T1         { return r.data }

func NewEntry[T1 any](id uint64, d T1) Entry[T1] {
	return entry[T1]{
		id:    id,
		data:  d,
	}
}

type Enries[T1 any] []Entry[T1]
