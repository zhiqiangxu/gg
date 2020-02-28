package mpsc

import "testing"

func TestMPSC(t *testing.T) {
	q := New()

	q.Push(1)
	q.Push(2)
	if q.Pop() != 1 {
		t.Fail()
	}
	if q.Pop() != 2 {
		t.Fail()
	}
	if !q.Empty() {
		t.Fail()
	}
}
