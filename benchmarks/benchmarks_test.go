package benchmarks

import (
	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid"
	"testing"
)
import fid "github.com/amterp/flexid"

func BenchmarkFlexId(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = fid.MustGenerate()
	}
}

func BenchmarkUuidV4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = uuid.New().String()
	}
}

func BenchmarkNanoid(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = gonanoid.MustID(21)
	}
}
