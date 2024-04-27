package id32

import "math/bits"

type myuint32 uint32

func (u myuint32) isZero() bool { return u == 0 }

// and returns the bitwise AND of u and m (u&m).
func (u myuint32) and(m myuint32) myuint32 {
	return myuint32(u & m)
}

// xor returns the bitwise XOR of u and m (u^m).
func (u myuint32) xor(m myuint32) myuint32 {
	return myuint32(u ^ m)
}

// or returns the bitwise OR of u and m (u|m).
func (u myuint32) or(m myuint32) myuint32 {
	return myuint32(u | m)
}

// not returns the bitwise NOT of u.
func (u myuint32) not() myuint32 {
	return myuint32(^u)
}

// subOne returns u - 1.
func (u myuint32) subOne() myuint32 {
	lo, borrow := bits.Sub32(uint32(u), 1, 0)
	return myuint32(lo - borrow)
}

// addOne returns u + 1.
func (u myuint32) addOne() myuint32 {
	lo, carry := bits.Add32(uint32(u), 1, 0)
	return myuint32(lo + carry)
}

// bitsSetFrom returns a copy of u with the given bit
// and all subsequent ones set.
func (u myuint32) bitsSetFrom(bit uint8) myuint32 {
	return u.or(myuint32(mask6[bit]).not())
}

// bitsClearedFrom returns a copy of u with the given bit
// and all subsequent ones cleared.
func (u myuint32) bitsClearedFrom(bit uint8) myuint32 {
	return u.and(myuint32(mask6[bit]))
}

var mask6 = [...]uint32{
	0x00000000, //0
	0x80000000, //1
	0xc0000000, //2
	0xe0000000, //3
	0xf0000000, //4
	0xf8000000, //5
	0xfc000000, //6
	0xfe000000, //7
	0xff000000, //8
	0xff800000, //9
	0xffc00000, //10
	0xffe00000, //11
	0xfff00000, //12
	0xfff80000, //13
	0xfffc0000, //14
	0xfffe0000, //15
	0xffff0000, //16
	0xffff8000, //17
	0xffffc000, //18
	0xffffe000, //19
	0xfffff000, //20
	0xfffff800, //21
	0xfffffc00, //22
	0xfffffe00, //23
	0xffffff00, //24
	0xffffff80, //25
	0xffffffc0, //26
	0xffffffe0, //27
	0xfffffff0, //28
	0xfffffff8, //29
	0xfffffffc, //30
	0xfffffffe, //31
	0xffffffff, //32
}


func bePutUint32(b []byte, v uint32) {
	_ = b[3] // early bounds check to guarantee safety of writes below
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

func beUint32(b []byte) uint32 {
	_ = b[3] // bounds check hint to compiler; see golang.org/issue/14808
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24 
}