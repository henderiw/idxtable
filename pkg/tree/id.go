package tree

type ID interface {
	Copy() ID
	Length() uint8
	ID() uint64
	Matches(id uint64) uint8
	ShiftLeft(shiftCount uint8) ID
	IsLeftBitSet() bool
	String() string
	Overlaps(ID) bool
	Compare(ID) int
	Less(ID) bool
	Next() ID
	Prev() ID
	Mask(l uint8) (ID, error)
	Masked() ID
}
