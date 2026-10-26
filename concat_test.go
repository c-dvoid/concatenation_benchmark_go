package concat

import (
	"fmt"
	"strings"
	"testing"
)

var implementations = []struct {
	name string
	fn   func([]string) string
}{
	{"Plus", Plus},
	{"Sprintf", Sprintf},
	{"Builder", Builder},
	{"BuilderGrow", BuilderGrow},
	{"BytesBuffer", BytesBuffer},
	{"Join", Join},
}

var sizes = []int{2, 10, 100, 1000}

// makeParts returns n short strings to concatenate.
func makeParts(n int) []string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "hello"
	}
	return parts
}

func TestImplementations(t *testing.T) {
	inputs := [][]string{
		nil,
		{""},
		{"a"},
		{"hello", "world"},
		{"", "x", "", "yz", ""},
		makeParts(1000),
	}

	for _, impl := range implementations {
		for _, in := range inputs {
			want := strings.Join(in, "")
			if got := impl.fn(in); got != want {
				t.Errorf("%s(%d parts) = %q, want %q", impl.name, len(in), got, want)
			}
		}
	}
}

// result is a global variable so the result escapes to the heap
// like it would in real code and the compiler can't discard the work
var result string

func BenchmarkConcat(b *testing.B) {
	for _, n := range sizes {
		parts := makeParts(n)
		for _, impl := range implementations {
			b.Run(fmt.Sprintf("%s/n=%d", impl.name, n), func(b *testing.B) {
				for b.Loop() {
					result = impl.fn(parts)
				}
			})
		}
	}
}
