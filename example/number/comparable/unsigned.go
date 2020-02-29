// Package comparable will convert a signed number to an unsigned one, but the order of numbers will not change
// run: gg -i unsigned.go -t SignedType=int32 -t UnsignedType=uint32
package comparable

import "unsafe"

type (
	// SignedType for input
	SignedType int64
	// UnsignedType for output
	UnsignedType uint64
)

// FYI: https://github.com/facebook/mysql-5.6/wiki/MyRocks-record-format#memcomparable-format

const mask = UnsignedType(1) << (unsafe.Sizeof(SignedType(0))*8 - 1)

// Unsigned will make negative values have all bits negated plus +1 while positive values have the high bit turned on
func Unsigned(in SignedType) (out UnsignedType) {
	// Converting a SignedType to UnsignedType does not change the memory layout, only the type.

	out = UnsignedType(in) ^ mask
	return
}
