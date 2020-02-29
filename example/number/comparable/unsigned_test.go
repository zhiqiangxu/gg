package comparable

import (
	"testing"
	"unsafe"
)

func TestUnsigned(t *testing.T) {
	i := int16(-1)
	if uint16(i) != 0xffff {
		t.Fail()
	}

	if unsafe.Sizeof(int64(0)) != 8 {
		t.Fail()
	}

	if mask != 0x8000000000000000 {
		t.Fail()
	}

	if Unsigned(1) <= Unsigned(0) {
		t.Fail()
	}

	if Unsigned(1) <= Unsigned(-1) {
		t.Fail()
	}
}
