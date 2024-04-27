package id32

import (
	"fmt"
	"math/bits"

	"github.com/henderiw/idxtable/pkg/tree"
)

const _leftmostBit = uint32(1 << 31)

func IsLeftBitSet(id uint64) bool {
	return uint32(id) >= _leftmostBit
}

type myid32 struct {
	id     uint32
	length uint8
}

func NewID(id uint32, length uint8) tree.ID {
	return myid32{
		id:     id,
		length: length,
	}
}

func (r myid32) Copy() tree.ID {
	return &myid32{
		id:     r.id,
		length: r.length,
	}
}

func (r myid32) Length() uint8 {
	return r.length
}

func (r myid32) ID() uint64 {
	return uint64(r.id)
}

// ShiftLeft shifts the address to the left
func (r myid32) ShiftLeft(shiftCount uint8) tree.ID {
	r.id <<= shiftCount
	r.length -= shiftCount
	return r
}

// IsLeftBitSet returns whether the leftmost bit is set
func (r myid32) IsLeftBitSet() bool {
	return r.id >= _leftmostBit
}

// String returns a string version of this ID.
func (r myid32) String() string {
	return fmt.Sprintf("%d/%d", r.id, r.length)
}

func (r myid32) Matches(id uint64) uint8 {
	return uint8(bits.LeadingZeros32(uint32(id) ^ r.id))
}

func (r myid32) Overlaps(b tree.ID) bool {
	var minbits uint8
	if r.Length() < b.Length() {
		minbits = r.Length()
	} else {
		minbits = b.Length()
	}
	if minbits == 0 {
		return true
	}
	ida := r.id & mask6[minbits]
	idb := uint32(b.ID()) & mask6[minbits]

	return ida == idb
}

// Compare returns an integer comparing two IPs.
// The result will be 0 if ip == ip2, -1 if ip < ip2, and +1 if ip > ip2.
// The definition of "less than" is the same as the [Addr.Less] method.
func (r myid32) Compare(id2 tree.ID) int {
	f1, f2 := r.Length(), id2.Length()
	if f1 < f2 {
		return -1
	}
	if f1 > f2 {
		return 1
	}
	if r.ID() < id2.ID() {
		return -1
	}
	if r.ID() > id2.ID() {
		return 1
	}
	return 0
}

// Less reports whether ip sorts before ip2.
// IP addresses sort first by length, then their address.
// IPv6 addresses with zones sort just after the same address without a zone.
func (r myid32) Less(id2 tree.ID) bool { return r.Compare(id2) == -1 }

// Next returns the address following ip.
// If there is none, it returns the zero [Addr].
func (id myid32) Next() tree.ID {
	id.id = uint32(myuint32(id.id).addOne())
	if id.id == 0 {
		// Overflowed.
		return myid32{}
	}

	return id
}

// Prev returns the ID before id.
// If there is none, it returns the ID zero value.
func (id myid32) Prev() tree.ID {
	if id.id == 0 {
		return myid32{}
	}
	id.id = uint32(myuint32(id.id).subOne())
	return id
}

// Prev returns the ID before id.
// If there is none, it returns the ID zero value.
func (id myid32) Mask(l uint8) (tree.ID, error) {
	if l > 32 {
		return nil, fmt.Errorf("length is too large, max 32, got: %d", l)
	}
	newid := uint32(myuint32(id.id).and(myuint32(mask6[uint32(l)])))
	return myid32{id: newid, length: l}, nil
}

func (id myid32) Masked() tree.ID {
	mid, _ := id.Mask(id.length)
	return mid
}

