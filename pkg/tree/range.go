package tree

type Range interface {
	From() ID
	To() ID
	SetTo(ID) Range
	SetFrom(ID) Range
	String() string
	IsValid() bool
	IsZero() bool
	IDs() []ID
	Less(other Range) bool
	AppendIDs(dst []ID) []ID

	EntirelyBefore(other Range) bool
	CoveredBy(other Range) bool
	InMiddleOf(other Range) bool
	OverlapsStartOf(other Range) bool
	OverlapsEndOf(other Range) bool
}
