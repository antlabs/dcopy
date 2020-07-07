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

// 最大深度
func Test_MaxDepth(t *testing.T) {
	type depth struct {
		First string
		Data  struct {
			Result string
		}
		Err struct {
			ErrMsg struct {
				Message string
			}
		}
	}

	src := depth{}
	src.First = "first"
	src.Data.Result = "test"
	src.Err.ErrMsg.Message = "good"

	for _, tc := range []testCase{
		func() testCase {
			d := depth{}
			err := Copy(&d, &src).MaxDepth(2).Do()
			assert.NoError(t, err)
			if err != nil {
				return testCase{}
			}
			need := depth{}
			need.First = "first"
			need.Data.Result = "test"
			return testCase{got: d, need: need}
		}(),
		func() testCase {
			d := depth{}
			Copy(&d, &src).MaxDepth(1).Do()
			need := depth{}
			need.First = "first"
			return testCase{got: d, need: need}
		}(),
		func() testCase {
			d := depth{}
			Copy(&d, &src).MaxDepth(3).Do()
			need := depth{}
			need.First = "first"
			need.Data.Result = "test"
			need.Err.ErrMsg.Message = "good"
			return testCase{got: d, need: need}
		}(),
	} {
		assert.Equal(t, tc.need, tc.got)
	}
}

// 测试设置tag的情况
func Test_TagName(t *testing.T) {
	type tagName struct {
		First string `copy:"first"`
		Data  struct {
			Result string
		}
	}

	src := tagName{}
	src.First = "first"
	src.Data.Result = "test"

	for _, tc := range []testCase{
		func() testCase {
			d := tagName{}
			Copy(&d, &src).RegisterTagName("copy").Do()
			need := tagName{}
			need.First = "first"
			return testCase{got: d, need: need}
		}(),
	} {
		assert.Equal(t, tc.need, tc.got)
	}
}

// 下面的test case 确保不panic
func Test_Special(t *testing.T) {
	for _, tc := range []testCase{
		func() testCase {
			// src有的字段, dst里面没有
			type src struct {
				Sex string
			}

			type dst struct {
				ID string
			}

			d := dst{}
			s := src{Sex: "m"}
			Copy(&d, &s).Do()
			return testCase{got: d, need: d}
		}(),
		func() testCase {
			// 同样的字段不同数据类型，不拷贝
			type src struct {
				Sex string
			}

			type dst struct {
				Sex int
			}

			d := dst{}
			s := src{Sex: "m"}
			Copy(&d, &s).Do()
			return testCase{got: d, need: d}
		}(),
		func() testCase {
			Copy(new(int), nil).Do()
			return testCase{got: true, need: true}
		}(),
	} {
		assert.Equal(t, tc.need, tc.got)
	}
}

// 测试循环引用
func Test_Cycle(t *testing.T) {
	for _, e := range []error{
		func() error {
			type src2 struct {
				P1 *src2
				ID string
			}

			// p1指向自己，构造一个环
			s := src2{}
			s.P1 = &s

			d := src2{}
			return Copy(&d, &s).Do()
		}(),
	} {
		assert.Error(t, e)
	}
}
