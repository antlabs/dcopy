package fastdeepcopy

import (
	"reflect"
	"sync"
	"unsafe"
)

var (
	cacheFunc map[twoType]cacheSet
	rdlock    sync.RWMutex
	OpenCache bool
)

type twoType struct {
	dst reflect.Type
	src reflect.Type
}

type cacheSet struct {
	setFuncs []*cacheElem
}

type cacheElem struct {
	dstOffset int
	srcOffset int

	set setFunc
}

func add(addr unsafe.Pointer, offset int) unsafe.Pointer {
	return unsafe.Pointer(uintptr(addr) + uintptr(offset))
}

func (c *cacheSet) do(dstAddr, srcAddr unsafe.Pointer) {
	for _, v := range c.setFuncs {
		v.set(add(dstAddr, v.dstOffset), add(srcAddr, v.srcOffset))
	}
}
