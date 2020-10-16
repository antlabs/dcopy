package dcopy

import (
	"reflect"
	"sync"
	"unsafe"
)

var (
	cacheAllFunc map[dstSrcType]*allFieldFunc = make(map[dstSrcType]*allFieldFunc)
	rdlock       sync.RWMutex
	OpenCache    bool
)

type dstSrcType struct {
	dst reflect.Type
	src reflect.Type
}

type allFieldFunc struct {
	fieldFuncs []*offsetAndFunc
}

type offsetAndFunc struct {
	srcKind   reflect.Kind
	dstOffset int
	srcOffset int

	set setFunc
}

func saveToCache(a *args, fieldFunc *allFieldFunc) {
	rdlock.Lock()
	defer rdlock.Unlock()

	cacheAllFunc[dstSrcType{dst: a.dstType, src: a.srcType}] = fieldFunc
}

func getSetFromCacheAndRun(a *args) (exist bool) {
	rdlock.RLock()
	cacheFunc, ok := cacheAllFunc[dstSrcType{dst: a.dstType, src: a.srcType}]
	if !ok {
		rdlock.RUnlock()
		return ok
	}
	rdlock.RUnlock()

	cacheFunc.do(0, a.dstAddr, a.srcAddr)
	return true
}

func add(addr unsafe.Pointer, offset int) unsafe.Pointer {
	return unsafe.Pointer(uintptr(addr) + uintptr(offset))
}

func newAllFieldFunc() *allFieldFunc {
	return &allFieldFunc{fieldFuncs: make([]*offsetAndFunc, 0, 8)}
}

func (af *allFieldFunc) append(a *args) {
	fieldFunc := a.offsetAndFunc
	af.fieldFuncs = append(af.fieldFuncs, fieldFunc)
}

func (c *allFieldFunc) do(start int, dstAddr, srcAddr unsafe.Pointer) {
	for k, v := range c.fieldFuncs[start:] {
		switch v.srcKind {
		case reflect.Array, reflect.Slice:
			c.do(k+1, add(dstAddr, v.dstOffset), add(srcAddr, v.srcOffset))
			return
		default:
			//fmt.Printf("%d:%d:%p:%p:%v\n", v.dstOffset, v.srcOffset, add(dstAddr, v.dstOffset), add(srcAddr, v.srcOffset), v.srcKind)
			v.set(add(dstAddr, v.dstOffset), add(srcAddr, v.srcOffset))
		}
	}
}
