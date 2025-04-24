package stid_test

import (
	"testing"

	"github.com/amterp/stid"
)

var sink string

func BenchmarkDefaultGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sink = stid.MustGenerate()
	}
}
