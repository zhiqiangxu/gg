package copyslice

import "testing"

func TestCopySlice(t *testing.T) {
	out := CopySlice(nil)
	if out != nil {
		t.Fail()
	}

	out = CopySlice([]T{})
	if out == nil {
		t.Fail()
	}
}
