package fastdeepcopy

import (
	"testing"

	"github.com/antlabs/deepcopy"
)

type testData struct {
	ID    int
	Int8  int8
	Int16 int16
	Int32 int32
	Int64 int64

	Uint8  uint8
	Uint16 uint16
	Uint32 uint32
	Uint64 uint64
	S      string
	Slice  []string
}

func defaultTestData() (src testData) {
	src.ID = 3
	src.Int8 = 8
	src.Int16 = 16
	src.Int32 = 32
	src.Int64 = 64

	src.Uint8 = 18
	src.Uint16 = 116
	src.Uint32 = 132
	src.Uint64 = 164
	src.S = "hello world"
	src.Slice = []string{"123", "456", "789"}
	return
}

var td = defaultTestData()

func Benchmark_Use_Ptr_fastdeepcopy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var dst testData
		Copy(&dst, &td).Do()
	}
}

func Benchmark_Use_reflectValue_DeepCopy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var dst testData
		deepcopy.Copy(&dst, &td).Do()
	}
}
