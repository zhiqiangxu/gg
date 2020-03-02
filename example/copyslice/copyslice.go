// Package copyslice copy a slice perfectly
// run: gg -i copyslice.go -t T=byte
package copyslice

// T for template
type T byte

// FYI: https://github.com/go101/go101/wiki/How-to-perfectly-clone-a-slice%3F

// CopySlice will return nil for nil, empty slice for empty slice
func CopySlice(in []T) []T {
	return append(in[:0:0], in...)
}
