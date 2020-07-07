package fastdeepcopy

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

const (
	noDepthLimited = -1
	noTagLimit     = ""
)

type visit struct {
	addr uintptr
	typ  reflect.Type
}

var ErrCircularReference = errors.New("deepcopy.Copy:Circular reference")

type emptyInterface struct {
	typ  *struct{}
	word unsafe.Pointer
}

type fastDeepCopy struct {
	dstValue reflect.Value
	srcValue reflect.Value
	err      error

	tagName  string
	maxDepth int
	visited  map[visit]struct{}
}

func Copy(dst, src interface{}) *fastDeepCopy {
	if dst == nil || src == nil {
		return &fastDeepCopy{err: errors.New("Unsupported type:nil")}
	}

	dstValue := reflect.ValueOf(dst)
	srcValue := reflect.ValueOf(src)

	if dstValue.Kind() != reflect.Ptr || srcValue.Kind() != reflect.Ptr {
		return &fastDeepCopy{err: errors.New("Unsupported type: Not a pointer")}
	}

	if !dstValue.Elem().CanAddr() {
		return &fastDeepCopy{err: fmt.Errorf("dst:%T value cannot take address", dstValue.Type())}
	}

	if !srcValue.Elem().CanAddr() {
		return &fastDeepCopy{err: fmt.Errorf("src:%T value cannot take address", dstValue.Type())}
	}

	return &fastDeepCopy{
		maxDepth: noDepthLimited,
		dstValue: dstValue,
		srcValue: srcValue,
		visited:  make(map[visit]struct{}, 10),
	}
}

// 设置最多递归的层次
func (f *fastDeepCopy) MaxDepth(maxDepth int) *fastDeepCopy {
	f.maxDepth = maxDepth
	return f
}

// 设置tag name，结构体的tag等于RegisterTagName注册的tag，才会copy值
func (f *fastDeepCopy) RegisterTagName(tagName string) *fastDeepCopy {
	f.tagName = tagName
	return f
}

// 需要的tag name
func haveTagName(curTabName string) bool {
	return len(curTabName) > 0
}

func (f *fastDeepCopy) Do() error {
	if f.err != nil {
		return f.err
	}

	return f.fastDeepCopy(f.dstValue.Elem().Type(), f.srcValue.Elem().Type(),
		unsafe.Pointer(f.dstValue.Elem().UnsafeAddr()),
		unsafe.Pointer(f.srcValue.Elem().UnsafeAddr()), 0)
}

func (f *fastDeepCopy) cpyDefault(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer, depth int) error {
	if dst.Kind() != src.Kind() {
		return nil
	}

	set := getSetFunc(src.Kind())
	if set == nil {
		return nil
	}

	set(dstAddr, srcAddr)
	return nil
}

func (f *fastDeepCopy) cpyPtr(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer, depth int) error {
	if dst.Kind() != src.Kind() {
		return nil
	}

	dst = dst.Elem()
	src = src.Elem()

	return f.fastDeepCopy(dst, src, dstAddr, srcAddr, depth)
}

func getHeader(typ reflect.Type, addr unsafe.Pointer) *reflect.SliceHeader {
	var header reflect.SliceHeader
	if typ.Kind() == reflect.Array {
		header.Data = uintptr(addr)
		header.Len = typ.Len()
		header.Cap = typ.Len()
		return &header
	}

	return (*reflect.SliceHeader)(addr)

}

// 支持异构copy, slice to slice, array to slice, slice to array
func (f *fastDeepCopy) cpySliceArray(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer, depth int) error {

	if dst.Kind() == reflect.Array && dst.Len() == 0 || dst.Kind() != reflect.Array && dst.Kind() != reflect.Slice {
		return nil
	}

	srcHeader := getHeader(src, srcAddr)
	dstHeader := getHeader(dst, dstAddr)

	if srcHeader.Len == 0 {
		return nil
	}

	if dstHeader.Cap == 0 {
		newAddr := reflect.MakeSlice(src, srcHeader.Len, srcHeader.Cap).Pointer()
		dstHeader.Data = newAddr
		dstHeader.Len = srcHeader.Len
		dstHeader.Cap = srcHeader.Cap
	}

	l := srcHeader.Len
	if dstHeader.Cap < l {
		l = dstHeader.Cap
	}

	elemType := dst.Elem()
	for i := 0; i < l; i++ {
		dstElemAddr := unsafe.Pointer(uintptr(dstHeader.Data) + uintptr(i)*elemType.Size())
		srcElemAddr := unsafe.Pointer(uintptr(srcHeader.Data) + uintptr(i)*elemType.Size())
		err := f.fastDeepCopy(src.Elem(), dst.Elem(), dstElemAddr, srcElemAddr, depth)
		if err != nil {
			return err
		}
	}

	dstHeader.Len = l
	return nil
}

// 使用type + address 转成 reflect.Value
func typePtrToValue(typ reflect.Type, addr unsafe.Pointer) reflect.Value {
	i := reflect.New(typ).Interface()
	ei := (*emptyInterface)(unsafe.Pointer(&i))
	ei.word = addr
	return reflect.ValueOf(i).Elem()
}

func getPtrFromVal(v *reflect.Value) unsafe.Pointer {
	ei := (*emptyInterface)(unsafe.Pointer(v))
	return ei.word
}

func (f *fastDeepCopy) cpyMap(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer, depth int) error {
	if dst.Kind() != reflect.Map || src.Kind() != reflect.Map {
		return nil
	}

	// 检查value是否相同
	if dst.Elem().Kind() != src.Elem().Kind() {
		return nil
	}

	// 检查key是否相同
	if dst.Key().Kind() != src.Key().Kind() {
		return nil
	}

	dstVal := typePtrToValue(dst, dstAddr)
	srcVal := typePtrToValue(src, srcAddr)

	if dstVal.IsNil() {
		newMap := reflect.MakeMapWithSize(src, srcVal.Len())
		dstVal.Set(newMap)
	}

	iter := srcVal.MapRange()
	for iter.Next() {
		k := iter.Key()
		v := iter.Value()

		newKey := reflect.New(k.Type()).Elem()
		if err := f.fastDeepCopy(newKey.Type(), k.Type(), getPtrFromVal(&newKey), getPtrFromVal(&k), depth); err != nil {
			return err
		}

		newVal := reflect.New(v.Type()).Elem()
		if err := f.fastDeepCopy(newVal.Type(), v.Type(), getPtrFromVal(&newVal), getPtrFromVal(&v), depth); err != nil {
			return err
		}

		dstVal.SetMapIndex(newKey, newVal)
	}

	return nil
}

func (f *fastDeepCopy) cpyStruct(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer, depth int) error {
	/*
		dstLen := dst.NumField()
		dstMap := make(map[string]*reflect.StructField, dstLen)
		for i := 0; i < dstLen; i++ {
			sf := dst.Field(i)
			if sf.PkgPath != "" && !sf.Anonymous {
				continue
			}

			dstMap[sf.Name] = &sf
		}
	*/

	n := src.NumField()
	for i := 0; i < n; i++ {

		sf := src.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

		if len(f.tagName) > 0 && !haveTagName(sf.Tag.Get(f.tagName)) {
			continue
		}

		//dstSf, ok := dstMap[sf.Name]
		dstSf, ok := dst.FieldByName(sf.Name)
		if !ok {
			continue
		}
		srcFieldAddr := unsafe.Pointer(uintptr(srcAddr) + sf.Offset)
		if err := f.checkCycle(sf.Type, srcFieldAddr); err != nil {
			return err
		}

		err := f.fastDeepCopy(dstSf.Type, sf.Type, unsafe.Pointer(uintptr(dstAddr)+dstSf.Offset), srcFieldAddr, depth+1)
		if err != nil {
			return err
		}
	}

	return nil
}

// 检查循环引用
func (f *fastDeepCopy) checkCycle(typ reflect.Type, addr unsafe.Pointer) error {

	v := visit{addr: uintptr(addr), typ: typ}

	if _, ok := f.visited[v]; ok {
		return ErrCircularReference
	}

	f.visited[v] = struct{}{}

	return nil
}

func (f *fastDeepCopy) cpyInterface(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer, depth int) error {
	if dst.Kind() != src.Kind() {
		return nil
	}

	dstInterfaceValue := typePtrToValue(dst, dstAddr)
	srcInterfaceValue := typePtrToValue(src, srcAddr)

	srcVal := srcInterfaceValue.Elem()

	newDst := reflect.New(srcVal.Type()).Elem()

	if srcVal.CanAddr() {
		f.fastDeepCopy(newDst.Type(), srcVal.Type(), unsafe.Pointer(newDst.UnsafeAddr()), unsafe.Pointer(srcVal.UnsafeAddr()), depth)
	}

	newDst.Set(srcVal)

	dstInterfaceValue.Set(newDst)
	return nil
}

func (f *fastDeepCopy) fastDeepCopy(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer, depth int) error {
	if f.err != nil {
		return f.err
	}

	if f.maxDepth != noDepthLimited && depth > f.maxDepth {
		return nil
	}

	switch src.Kind() {
	case reflect.Slice, reflect.Array:
		return f.cpySliceArray(dst, src, dstAddr, srcAddr, depth)
	case reflect.Map:
		return f.cpyMap(dst, src, dstAddr, srcAddr, depth)
	case reflect.Struct:
		return f.cpyStruct(dst, src, dstAddr, srcAddr, depth)
	case reflect.Interface:
		return f.cpyInterface(dst, src, dstAddr, srcAddr, depth)
	case reflect.Ptr:
		return f.cpyPtr(dst, src, dstAddr, srcAddr, depth)
	default:
		return f.cpyDefault(dst, src, dstAddr, srcAddr, depth)
	}

	return nil
}
