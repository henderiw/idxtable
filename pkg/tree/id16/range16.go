package id16

import (
	"math/bits"

	"github.com/henderiw/idxtable/pkg/tree"
)

type Range struct {
	from uint16
	to   uint16
}

func IPRangeFrom(from, to uint16) Range {
	return Range{
		from: from,
		to:   to,
	}
}

func (r Range) AppendIDs(dst []tree.ID) []tree.ID {
	return appendRangeIDs(dst, r.makeID, myuint16(r.from), myuint16(r.to))
}

func (r Range) makeID(id myuint16, bits uint8) tree.ID {
	bits -= 4
	return &myid16{id: uint16(id), length: bits}
}

type idMaker func(a myuint16, bits uint8) tree.ID

func u16CommonMask(a, b myuint16) uint8 {
	return uint8(bits.LeadingZeros16(uint16(a ^ b)))
}

func appendRangeIDs(dst []tree.ID, makePrefix idMaker, a, b myuint16) []tree.ID {
	common, ok := compareIDs(a, b)
	if ok {
		// a to b represents a whole range, like 10.50.0.0/16.
		// (a being 10.50.0.0 and b being 10.50.255.255)
		return append(dst, makePrefix(a, common))
	}
	// Otherwise recursively do both halves.
	dst = appendRangeIDs(dst, makePrefix, a, a.bitsSetFrom(common+1))
	dst = appendRangeIDs(dst, makePrefix, b.bitsClearedFrom(common+1), b)
	return dst
}

// aZeroBSet is whether, after the common bits, a is all zero bits and
// b is all set (one) bits.
func compareIDs(a, b myuint16) (common uint8, aZeroBSet bool) {
	common = u16CommonMask(a, b)

	// See whether a and b, after their common shared bits, end
	// in all zero bits or all one bits, respectively.
	if common == 16 {
		return common, true
	}

	m := mask6[common]

	ma := myuint16(a)
	mb := myuint16(b)
	mm := myuint16(m)

	return common, (ma.xor(ma.and(mm)).isZero() &&
		mb.or(mm) == myuint16(uint16(^uint16(0))))
}
