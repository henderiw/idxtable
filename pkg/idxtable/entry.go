package idxtable

type Entry[T1 any] interface {
	ID() int64
	Data() T1
}

type entry[T1 any] struct {
	id   int64
	data T1
	// table
	start int64
	table Table[T1]
}
type Entries[T1 any] []Entry[T1]

func (r entry[T1]) ID() int64        { return r.id }
func (r entry[T1]) Data() T1         { return r.data }
func (r entry[T1]) Table() Table[T1] { return r.table }
func (r entry[T1]) HasTable() bool   { return r.table != nil }

func NewEntry[T1 any](id int64, d T1) Entry[T1] {
	return entry[T1]{
		id:    id,
		data:  d,
		table: nil,
	}
}

func NewTableEntry[T1 any](id int64, d T1, t Table[T1]) Entry[T1] {
	return entry[T1]{
		id:    id,
		data:  d,
		table: t,
	}
}

type Enries[T1 any] []Entry[T1]
