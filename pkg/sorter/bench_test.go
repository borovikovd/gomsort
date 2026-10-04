package sorter

import (
	"fmt"
	"strings"
	"testing"
)

// BenchmarkSortManyTypes sorts files of growing size, each type with two
// methods out of order, a const between them and a constructor at the end;
// the time per type should stay flat.
func BenchmarkSortManyTypes(b *testing.B) {
	for _, types := range []int{250, 1000, 4000} {
		var src strings.Builder
		src.WriteString("package bench\n")
		for i := 0; i < types; i++ {
			fmt.Fprintf(&src, "\ntype t%d struct{}\n\nfunc (x *t%[1]d) helper() {}\n\nconst c%[1]d = %[1]d\n\nfunc (x *t%[1]d) Run() { x.helper() }\n", i)
		}
		for i := 0; i < types; i++ {
			fmt.Fprintf(&src, "\nfunc newT%d() *t%[1]d { return nil }\n", i)
		}
		source := src.String()
		b.Run(fmt.Sprintf("types=%d", types), func(b *testing.B) {
			for b.Loop() {
				sorter, err := NewFromSource(source)
				if err != nil {
					b.Fatal(err)
				}
				if _, _, err := sorter.Sort(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
