package flexid_test

import (
	"testing"

	fid "github.com/amterp/flexid"
)

var sink string

func BenchmarkDefaultGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sink = fid.MustGenerate()
	}
}
