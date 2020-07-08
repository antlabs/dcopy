package fastdeepcopy

import (
	"reflect"
	"sync"
	"unsafe"
)

type args struct {
	dstType reflect.Type
	srcType reflect.Type
	dstAddr unsafe.Pointer
	srcAddr unsafe.Pointer
}

var argsPool = &sync.Pool{
	New: func() interface{} {
		return &args{}
	},
}
