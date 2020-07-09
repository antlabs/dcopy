package fastdeepcopy

import (
	"reflect"
	"sync"
	"unsafe"
)

type args struct {
	srcName string //debug
	dstType reflect.Type
	srcType reflect.Type
	dstAddr unsafe.Pointer
	srcAddr unsafe.Pointer
	*offsetAndFunc
}

var argsPool = &sync.Pool{
	New: func() interface{} {
		return &args{}
	},
}
