package dcopy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCacheData struct {
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
	Array  [4]int //相同尺寸数据
	//slice 拷贝 slice
	//slice 对 array
	//array 对 slice
}

func defaultTestCacheData() (src testCacheData) {
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
	src.Array = [4]int{1, 2, 3}
	return
}

func Test_Cache(t *testing.T) {
	var dst testCacheData

	OpenCache = true
	defer func() { OpenCache = false }()

	src := defaultTestCacheData()

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
	assert.Equal(t, dst.S, "hello world")
	assert.Equal(t, dst.Array, [4]int{1, 2, 3})
}
