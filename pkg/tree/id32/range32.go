package id32

import (
	"fmt"
	"math/bits"
	"sort"
	"strconv"
	"strings"

	"github.com/henderiw/idxtable/pkg/tree"
)

type Range struct {
	from tree.ID
	to   tree.ID
}

func RangeFrom(from, to uint32) Range {
	return Range{
		from: NewID(from, 32),
		to:   NewID(to, 32),
	}
}

// From returns the lower bound of r.
func (r Range) From() tree.ID { return r.from }

// To returns the upper bound of r.
func (r Range) To() tree.ID { return r.to }

func ParseRange(s string) (Range, error) {
	var r Range
	h := strings.IndexByte(s, '-')
	if h == -1 {
		return r, fmt.Errorf("no hyphen in range %q", s)
	}
	from, to := s[:h], s[h+1:]
	fromUint32, err := strconv.ParseUint(from, 10, 32)
	if err != nil {
		return r, fmt.Errorf("invalid from id %q in range %q", from, s)
	}
	toUint32, err := strconv.ParseUint(to, 10, 32)
	if err != nil {
		return r, fmt.Errorf("invalid to id %q in range %q", to, s)
	}
	return Range{
		from: NewID(uint32(fromUint32), 32),
		to:   NewID(uint32(toUint32), 32),
	}, nil
}

func (r Range) String() string {
	return fmt.Sprintf("%d-%d", r.from.ID(), r.to.ID())
}

func (r Range) IsValid() bool {
	return r.from.Length() == r.to.Length() &&
		!(r.to.ID() < r.From().ID())
}

func (r Range) IsZero() bool {
	return r == Range{}
}

func (r Range) less(other Range) bool {
	if cmp := r.from.Compare(other.from); cmp != 0 {
		return cmp < 0
	}
	return other.to.Less(r.to)
}

func (r Range) IDs() []tree.ID {
	return r.AppendIDs(nil)
}

func (r Range) AppendIDs(dst []tree.ID) []tree.ID {
	return appendRangeIDs(dst, r.makeID, myuint32(r.from.ID()), myuint32(r.to.ID()))
}

func (r Range) makeID(id myuint32, bits uint8) tree.ID {
	return &myid32{id: uint32(id), length: bits}
}

// entirelyBefore returns whether r lies entirely before other in IP
// space.
func (r Range) entirelyBefore(other Range) bool {
	return r.to.Less(other.from)
}

func lessOrEq(id1, id2 tree.ID) bool { return id1.Compare(id2) <= 0 }

// entirelyWithin returns whether r is entirely contained within
// other.
func (r Range) coveredBy(other Range) bool {
	return lessOrEq(other.from, r.from) && lessOrEq(r.to, other.to)
}

// inMiddleOf returns whether r is inside other, but not touching the
// edges of other.
func (r Range) inMiddleOf(other Range) bool {
	return other.from.Less(r.from) && r.to.Less(other.to)
}

// overlapsStartOf returns whether r entirely overlaps the start of
// other, but not all of other.
func (r Range) overlapsStartOf(other Range) bool {
	return lessOrEq(r.from, other.from) && r.to.Less(other.to)
}

// overlapsEndOf returns whether r entirely overlaps the end of
// other, but not all of other.
func (r Range) overlapsEndOf(other Range) bool {
	return other.from.Less(r.from) && lessOrEq(other.to, r.to)
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
	if common == 32 {
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
func mergeRanges(rr []Range) (out []Range, valid bool) {
	// Always return a copy of r, to avoid aliasing slice memory in
	// the caller.
	switch len(rr) {
	case 0:
		return nil, true
	case 1:
		return []Range{rr[0]}, true
	}

	sort.Slice(rr, func(i, j int) bool { return rr[i].less(rr[j]) })
	out = make([]Range, 1, len(rr))
	out[0] = rr[0]
	for _, r := range rr[1:] {
		prev := &out[len(out)-1]
		switch {
		case !r.IsValid():
			// Invalid ranges make no sense to merge, refuse to
			// perform.
			return nil, false
		case prev.to.Next() == r.from:
			// prev and r touch, merge them.
			//
			//   prev     r
			// f------tf-----t
			prev.to = r.to
		case prev.to.Less(r.from):
			// No overlap and not adjacent (per previous case), no
			// merging possible.
			//
			//   prev       r
			// f------t  f-----t
			out = append(out, r)
		case prev.to.Less(r.to):
			// Partial overlap, update prev
			//
			//   prev
			// f------t
			//     f-----t
			//        r
			prev.to = r.to
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
func RangeOfID(id tree.ID) Range {
	id = id.Masked()
	if id == nil {
		return Range{}
	}
	return RangeFrom(uint32(id.ID()), uint32(LastID(id).ID()))
}

func LastID(id tree.ID) tree.ID {
	if id == nil {
		return nil
	}
	var a4 [4]byte
	bePutUint32(a4[:], uint32(id.ID()))
	for b := uint8(id.Length()); b < 32; b++ {
		byteNum, bitInByte := b/8, 7-(b%8)
		a4[byteNum] |= 1 << uint(bitInByte)
	}
	return NewID(beUint32(a4[:]), 32)
}
