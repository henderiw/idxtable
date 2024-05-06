package id32

import (
	"fmt"
	"math/bits"
	"sort"
	"strconv"
	"strings"

	"github.com/henderiw/idxtable/pkg/tree"
)

type r32 struct {
	from tree.ID
	to   tree.ID
}

func RangeFrom(from, to uint32) tree.Range {
	return r32{
		from: NewID(from, IDBitSize),
		to:   NewID(to, IDBitSize),
	}
}

// From returns the lower bound of r.
func (r r32) From() tree.ID { return r.from }

// To returns the upper bound of r.
func (r r32) To() tree.ID { return r.to }

func (r r32) SetTo(id tree.ID) tree.Range{
	r.to = id
	return r
}

func (r r32) SetFrom(id tree.ID) tree.Range {
	r.from = id
	return r
}

func ParseRange(s string) (tree.Range, error) {
	var r tree.Range
	h := strings.IndexByte(s, '-')
	if h == -1 {
		return r, fmt.Errorf("no hyphen in range %q", s)
	}
	from, to := s[:h], s[h+1:]
	fromUint32, err := strconv.ParseUint(from, 10, int(IDBitSize))
	if err != nil {
		return r, fmt.Errorf("invalid from id %q in range %q", from, s)
	}
	toUint32, err := strconv.ParseUint(to, 10, int(IDBitSize))
	if err != nil {
		return r, fmt.Errorf("invalid to id %q in range %q", to, s)
	}
	return r32{
		from: NewID(uint32(fromUint32), IDBitSize),
		to:   NewID(uint32(toUint32), IDBitSize),
	}, nil
}

func (r r32) String() string {
	return fmt.Sprintf("%d-%d", r.from.ID(), r.to.ID())
}

func (r r32) IsValid() bool {
	return r.from.Length() == r.to.Length() &&
		!(r.to.ID() < r.From().ID())
}

func (r r32) IsZero() bool {
	return r == r32{}
}

func (r r32) Less(other tree.Range) bool {
	if cmp := r.from.Compare(other.From()); cmp != 0 {
		return cmp < 0
	}
	return other.To().Less(r.to)
}

func (r r32) IDs() []tree.ID {
	return r.AppendIDs(nil)
}

func (r r32) AppendIDs(dst []tree.ID) []tree.ID {
	return appendRangeIDs(dst, r.makeID, myuint32(r.from.ID()), myuint32(r.to.ID()))
}

func (r r32) makeID(id myuint32, bits uint8) tree.ID {
	return &myid32{id: uint32(id), length: bits}
}

// entirelyBefore returns whether r lies entirely before other in IP
// space.
func (r r32) EntirelyBefore(other tree.Range) bool {
	return r.to.Less(other.From())
}

func lessOrEq(id1, id2 tree.ID) bool { return id1.Compare(id2) <= 0 }

// entirelyWithin returns whether r is entirely contained within
// other.
func (r r32) CoveredBy(other tree.Range) bool {
	return lessOrEq(other.From(), r.From()) && lessOrEq(r.To(), other.To())
}

// inMiddleOf returns whether r is inside other, but not touching the
// edges of other.
func (r r32) InMiddleOf(other tree.Range) bool {
	return other.From().Less(r.from) && r.to.Less(other.To())
}

// overlapsStartOf returns whether r entirely overlaps the start of
// other, but not all of other.
func (r r32) OverlapsStartOf(other tree.Range) bool {
	return lessOrEq(r.from, other.From()) && r.to.Less(other.To())
}

// overlapsEndOf returns whether r entirely overlaps the end of
// other, but not all of other.
func (r r32) OverlapsEndOf(other tree.Range) bool {
	return other.From().Less(r.from) && lessOrEq(other.To(), r.to)
}

type idMaker func(a myuint32, bits uint8) tree.ID

func u32CommonMask(a, b myuint32) uint8 {
	return uint8(bits.LeadingZeros32(uint32(a ^ b)))
}

func appendRangeIDs(dst []tree.ID, makePrefix idMaker, a, b myuint32) []tree.ID {
	common, ok := compareIDs(a, b)
	if ok {
		return append(dst, makePrefix(a, common))
	}
	// Otherwise recursively do both halves.
	dst = appendRangeIDs(dst, makePrefix, a, a.bitsSetFrom(common+1))
	dst = appendRangeIDs(dst, makePrefix, b.bitsClearedFrom(common+1), b)
	return dst
}

// aZeroBSet is whether, after the common bits, a is all zero bits and
// b is all set (one) bits.
func compareIDs(a, b myuint32) (common uint8, aZeroBSet bool) {
	common = u32CommonMask(a, b)

	// See whether a and b, after their common shared bits, end
	// in all zero bits or all one bits, respectively.
	if common == IDBitSize {
		return common, true
	}

	m := mask6[common]

	ma := myuint32(a)
	mb := myuint32(b)
	mm := myuint32(m)

	return common, (ma.xor(ma.and(mm)).isZero() &&
		mb.or(mm) == myuint32(uint32(^uint32(0))))
}

// mergeRanges returns the minimum and sorted set of ranges that
// cover r.
func mergeRanges(rr []tree.Range) (out []tree.Range, valid bool) {
	// Always return a copy of r, to avoid aliasing slice memory in
	// the caller.
	switch len(rr) {
	case 0:
		return nil, true
	case 1:
		out = append(out, rr[0])
		return out, true
	}

	sort.Slice(rr, func(i, j int) bool { return rr[i].Less(rr[j]) })
	out = make([]tree.Range, 1, len(rr))
	out[0] = rr[0]
	for _, r := range rr[1:] {
		prev := &out[len(out)-1]
		switch {
		case !r.IsValid():
			// Invalid ranges make no sense to merge, refuse to
			// perform.
			return nil, false
		case (*prev).To().Next() == r.From():
			// prev and r touch, merge them.
			//
			//   prev     r
			// f------tf-----t
			(*prev) = (*prev).SetTo(r.To())
		case (*prev).To().Less(r.From()):
			// No overlap and not adjacent (per previous case), no
			// merging possible.
			//
			//   prev       r
			// f------t  f-----t
			out = append(out, r)
		case (*prev).To().Less(r.To()):
			// Partial overlap, update prev
			//
			//   prev
			// f------t
			//     f-----t
			//        r
			(*prev) = (*prev).SetTo(r.To())
		default:
			// r entirely contained in prev, nothing to do.
			//
			//    prev
			// f--------t
			//  f-----t
			//     r
		}
	}
	return out, true
}

// Range returns the inclusive range of IPs that p covers.
//
// If p is zero or otherwise invalid, Range returns the zero value.
func RangeOfID(id tree.ID) tree.Range {
	id = id.Masked()
	if id == nil {
		return r32{}
	}
	return RangeFrom(uint32(id.ID()), uint32(LastID(id).ID()))
}

func LastID(id tree.ID) tree.ID {
	if id == nil {
		return nil
	}
	var a4 [4]byte
	bePutUint32(a4[:], uint32(id.ID()))
	for b := uint8(id.Length()); b < IDBitSize; b++ {
		byteNum, bitInByte := b/8, 7-(b%8)
		a4[byteNum] |= 1 << uint(bitInByte)
	}
	return NewID(beUint32(a4[:]), IDBitSize)
}
