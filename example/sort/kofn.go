package sort

// DT for data type
// can be replace by https://github.com/zhiqiangxu/gg
type DT uint64

// KSmallest for k smallest
// mutates ns
// not sorted
func KSmallest(ns []DT, k int) []DT {

	i := len(ns) / 2

	pos := PartitionLT(ns, i) + 1
	if pos == k {
		return ns[0:k]
	}

	if pos < k {
		return append(ns[0:pos], KSmallest(ns[pos+1:], k-pos)...)
	}

	// pos > k

	return KSmallest(ns[0:pos-1], k)
}

// KLargest for k largest
// mutates ns
func KLargest(ns []DT, k int) []DT {
	i := len(ns) / 2

	pos := PartitionGT(ns, i) + 1
	if pos == k {
		return ns[0:k]
	}

	if pos < k {
		return append(ns[0:pos], KLargest(ns[pos+1:], k-pos)...)
	}

	// pos > k

	return KLargest(ns[0:pos-1], k)
}

// PartitionLT partitions array by i-th element
// mutates ns so that all values less than i-th element are on the left
// assume values are distinct
// returns the pos of i-th element
func PartitionLT(ns []DT, i int) (pos int) {
	e := ns[i]
	for _, v := range ns {
		if v < e {
			pos++
		}
	}

	if i != pos {
		ns[i], ns[pos] = ns[pos], ns[i]
	}

	ri := pos + 1
	if ri == len(ns) {
		return
	}
	for li := 0; li < pos; li++ {
		if ns[li] > e {
			for {
				if ns[ri] < e {
					ns[li], ns[ri] = ns[ri], ns[li]
					ri++
					break
				} else {
					ri++
				}
			}
			if ri == len(ns) {
				return
			}
		}
	}

	return
}

func PartitionGT(ns []DT, i int) (pos int) {
	e := ns[i]
	for _, v := range ns {
		if v > e {
			pos++
		}
	}

	if i != pos {
		ns[i], ns[pos] = ns[pos], ns[i]
	}

	ri := pos + 1
	if ri == len(ns) {
		return
	}
	for li := 0; li < pos; li++ {
		if ns[li] < e {
			for {
				if ns[ri] > e {
					ns[li], ns[ri] = ns[ri], ns[li]
					ri++
					break
				} else {
					ri++
				}
			}
			if ri == len(ns) {
				return
			}
		}
	}

	return
}
