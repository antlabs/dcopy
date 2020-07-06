package fastdeepcopy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	need interface{}
	got  interface{}
}

type dstTestData struct {
	ID    int
	Int8  int8
	Int16 int16
	Int32 int32
	Int64 int64

	Uint8        uint8
	Uint16       uint16
	Uint32       uint32
	Uint64       uint64
	S            string
	StringSlice  []string
	StringArray  [3]string
	SliceToArray [4]int
}

type srcTestData struct {
	ID    int
	Int8  int8
	Int16 int16
	Int32 int32
	Int64 int64

	Uint8        uint8
	Uint16       uint16
	Uint32       uint32
	Uint64       uint64
	S            string
	StringSlice  []string
	StringArray  [4]string
	SliceToArray []int
}

func Test_FastDeepCopy(t *testing.T) {
	var dst dstTestData
	var src srcTestData

	src.ID = 3
	src.Int8 = 8
	src.Int16 = 16
	src.Int32 = 32
	src.Int64 = 64

	src.Uint8 = 18
	src.Uint16 = 116
	src.Uint32 = 132
	src.Uint64 = 164
	src.S = "hello"
	src.StringSlice = []string{"hello", "world"}
	src.StringArray = [4]string{"1", "2", "3", "4"}
	src.SliceToArray = []int{1, 2, 3, 4, 5}

	err := Copy(&dst, &src).Do()
	assert.NoError(t, err)
	assert.Equal(t, dst.ID, 3)
	assert.Equal(t, dst.Int8, int8(8))
	assert.Equal(t, dst.Int16, int16(16))
	assert.Equal(t, dst.Int32, int32(32))
	assert.Equal(t, dst.Int64, int64(64))

	assert.Equal(t, dst.Uint8, uint8(18))
	assert.Equal(t, dst.Uint16, uint16(116))
	assert.Equal(t, dst.Uint32, uint32(132))
	assert.Equal(t, dst.Uint64, uint64(164))
	assert.Equal(t, dst.S, "hello")
	assert.Equal(t, dst.StringSlice, []string{"hello", "world"})
	assert.Equal(t, dst.StringArray, [3]string{"1", "2", "3"})
	assert.Equal(t, dst.SliceToArray, [4]int{1, 2, 3, 4})

}
