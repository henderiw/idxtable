package id16

import (
	"fmt"
	"math/bits"

	"github.com/henderiw/idxtable/pkg/tree"
)

const _leftmostBit = uint16(1 << 15)

func IsLeftBitSet(id uint64) bool {
	return uint16(id) >= _leftmostBit
}

type myid16 struct {
	id     uint16
	length uint8
}

func NewID(id uint16, length uint8) tree.ID {
	return myid16{
		id:     id,
		length: length,
	}
}

func (r myid16) Copy() tree.ID {
	return &myid16{
		id:     r.id,
		length: r.length,
	}
}

func (r myid16) Length() uint8 {
	return r.length
}

func (r myid16) ID() uint64 {
	return uint64(r.id)
}

// ShiftLeft shifts the address to the left
func (r myid16) ShiftLeft(shiftCount uint8) tree.ID {
	r.id <<= shiftCount
	r.length -= shiftCount
	return r
}

// IsLeftBitSet returns whether the leftmost bit is set
func (r myid16) IsLeftBitSet() bool {
	return r.id >= _leftmostBit
}

// String returns a string version of this ID.
func (r myid16) String() string {
	return fmt.Sprintf("%d/%d", r.id, r.length)
}

func (r myid16) Matches(id uint64) uint8 {
	return uint8(bits.LeadingZeros16(uint16(id) ^ r.id))
}

func (r myid16) Overlaps(b tree.ID) bool {
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
	idb := uint16(b.ID()) & mask6[minbits]

	//fmt.Println("overlaps", ida, idb)
	return ida == idb
}
// Compare returns an integer comparing two IDs.
// The result will be 0 if id == id2, -1 if id < id2, and +1 if id > id2.
// The definition of "less than" is the same as the [Addr.Less] method.
func (r myid16) Compare(id2 tree.ID) int {
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
func (r myid16) Less(id2 tree.ID) bool { return r.Compare(id2) == -1 }

// Next returns the address following ip.
// If there is none, it returns the zero [Addr].
func (id myid16) Next() tree.ID {
	id.id = uint16(myuint16(id.id).addOne())
	if id.id == 0 {
		// Overflowed.
		return myid16{}
	}

	return id
}

// Prev returns the ID before id.
// If there is none, it returns the ID zero value.
func (id myid16) Prev() tree.ID {
	if id.id == 0 {
		return myid16{}
	}
	id.id = uint16(myuint16(id.id).subOne())
	return id
}

// Prev returns the ID before id.
// If there is none, it returns the ID zero value.
func (id myid16) Mask(l uint8) (tree.ID, error) {
	if l > 16 {
		return nil, fmt.Errorf("length is too large, max 16, got: %d", l)
	}
	newid := uint16(myuint16(id.id).and(myuint16(mask6[uint16(l)])))
	return myid16{id: newid, length: l}, nil
}

func (id myid16) Masked() tree.ID {
	mid, _ := id.Mask(id.length)
	return mid
}