package sort

import (
	"testing"

	"gotest.tools/assert"
)

func TestPartition(t *testing.T) {

	{
		data := [][]DT{
			[]DT{5, 4, 3, 2, 1},
			[]DT{1, 2, 3, 4, 5},
			[]DT{1, 3, 2, 5, 4},
		}

		for _, ns := range data {
			i := 3
			v := ns[i]
			pos := PartitionLT(ns, i)
			verifyLT(t, ns, v, pos)

		}
	}

	{
		data := [][]DT{
			[]DT{5, 4, 3, 2, 1},
			[]DT{1, 2, 3, 4, 5},
			[]DT{1, 3, 2, 5, 4},
		}

		for _, ns := range data {
			i := 3
			v := ns[i]
			pos := PartitionGT(ns, i)
			verifyGT(t, ns, v, pos)
		}
	}

	{
		data := [][]DT{
			[]DT{5, 4, 3, 2, 1},
			[]DT{1, 2, 3, 4, 5},
			[]DT{1, 3, 2, 5, 4},
		}

		for _, ns := range data {
			ks := KSmallest(ns, 3)
			verifyKS(t, ks, ns, 3)
		}
	}

	{
		data := [][]DT{
			[]DT{5, 4, 3, 2, 1},
			[]DT{1, 2, 3, 4, 5},
			[]DT{1, 3, 2, 5, 4},
		}

		for _, ns := range data {
			kl := KLargest(ns, 3)
			verifyKL(t, kl, ns, 3)
		}
	}

}

func verifyKL(t *testing.T, ks, ns []DT, k int) {
	assert.Assert(t, len(ks) == k, "len(ks) != k")
	m := make(map[DT]bool)
	for _, v := range ns {
		m[v] = true
	}

	for _, v := range ks {
		delete(m, v)
	}

	for _, v1 := range ks {
		for v2 := range m {
			assert.Assert(t, v2 < v1, "v2>v1")
		}
	}
}

func verifyKS(t *testing.T, ks, ns []DT, k int) {
	assert.Assert(t, len(ks) == k, "len(ks) != k")
	m := make(map[DT]bool)
	for _, v := range ns {
		m[v] = true
	}

	for _, v := range ks {
		delete(m, v)
	}

	for _, v1 := range ks {
		for v2 := range m {
			assert.Assert(t, v2 > v1, "v2<v1")
		}
	}
}

func verifyLT(t *testing.T, ns []DT, v DT, pos int) {
	assert.Assert(t, v == ns[pos], "v != ns[pos]")
	for i := 0; i < len(ns); i++ {
		if i < pos {
			assert.Assert(t, ns[i] < v, "left element not smaller")
		} else if i > pos {
			assert.Assert(t, ns[i] > v, "right element not larger")
		}
	}
}

func verifyGT(t *testing.T, ns []DT, v DT, pos int) {
	assert.Assert(t, v == ns[pos], "v != ns[pos]")
	for i := 0; i < len(ns); i++ {
		if i < pos {
			assert.Assert(t, ns[i] > v, "left element not larger")
		} else if i > pos {
			assert.Assert(t, ns[i] < v, "right element not smaller")
		}
	}
}
