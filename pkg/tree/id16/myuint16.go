package id16

type myuint16 uint16

func (u myuint16) isZero() bool { return u == 0 }

// and returns the bitwise AND of u and m (u&m).
func (u myuint16) and(m myuint16) myuint16 {
	return myuint16(u & m)
}

// xor returns the bitwise XOR of u and m (u^m).
func (u myuint16) xor(m myuint16) myuint16 {
	return myuint16(u ^ m)
}

// or returns the bitwise OR of u and m (u|m).
func (u myuint16) or(m myuint16) myuint16 {
	return myuint16(u | m)
}

// not returns the bitwise NOT of u.
func (u myuint16) not() myuint16 {
	return myuint16(^u)
}

// subOne returns u - 1.
func (u myuint16) subOne() myuint16 {
	lo, borrow := Sub16(uint16(u), 1, 0)
	return myuint16(lo - borrow)
}

// addOne returns u + 1.
func (u myuint16) addOne() myuint16 {
	lo, carry := Add16(uint16(u), 1, 0)
	return myuint16(lo + carry)
}

// Sub32 returns the difference of x, y and borrow, diff = x - y - borrow.
// The borrow input must be 0 or 1; otherwise the behavior is undefined.
// The borrowOut output is guaranteed to be 0 or 1.
//
// This function's execution time does not depend on the inputs.
func Sub16(x, y, borrow uint16) (diff, borrowOut uint16) {
	diff = x - y - borrow
	// The difference will underflow if the top bit of x is not set and the top
	// bit of y is set (^x & y) or if they are the same (^(x ^ y)) and a borrow
	// from the lower place happens. If that borrow happens, the result will be
	// 1 - 1 - 1 = 0 - 0 - 1 = 1 (& diff).
	borrowOut = ((^x & y) | (^(x ^ y) & diff)) >> 15
	return
}

// Add32 returns the sum with carry of x, y and carry: sum = x + y + carry.
// The carry input must be 0 or 1; otherwise the behavior is undefined.
// The carryOut output is guaranteed to be 0 or 1.
//
// This function's execution time does not depend on the inputs.
func Add16(x, y, carry uint16) (sum, carryOut uint16) {
	sum64 := uint64(x) + uint64(y) + uint64(carry)
	sum = uint16(sum64)
	carryOut = uint16(sum64 >> 16)
	return
}

// bitsSetFrom returns a copy of u with the given bit
// and all subsequent ones set.
func (u myuint16) bitsSetFrom(bit uint8) myuint16 {
	return u.or(myuint16(mask6[bit]).not())
}

// bitsClearedFrom returns a copy of u with the given bit
// and all subsequent ones cleared.
func (u myuint16) bitsClearedFrom(bit uint8) myuint16 {
	return u.and(myuint16(mask6[bit]))
}

var mask6 = [...]uint16{
	0x0000, //0
	0x8000, //1
	0xc000, //2
	0xe000, //3
	0xf000, //4
	0xf800, //5
	0xfc00, //6
	0xfe00, //7
	0xff00, //8
	0xff80, //9
	0xffc0, //10
	0xffe0, //11
	0xfff0, //12
	0xfff8, //13
	0xfffc, //14
	0xfffe, //15
	0xffff, //16
}


func bePutUint16(b []byte, v uint16) {
	_ = b[1] // early bounds check to guarantee safety of writes below
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

func beUint16(b []byte) uint16 {
	_ = b[1] // bounds check hint to compiler; see golang.org/issue/14808
	return uint16(b[1]) | uint16(b[0])<<8
}
