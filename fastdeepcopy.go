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
		//visited:  make(map[visit]struct{}, 10),
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

	arg := argsPool.Get().(*args)
	defer argsPool.Put(arg)

	arg.dstType = f.dstValue.Elem().Type()
	arg.srcType = f.srcValue.Elem().Type()
	arg.dstAddr = unsafe.Pointer(f.dstValue.Elem().UnsafeAddr())
	arg.srcAddr = unsafe.Pointer(f.srcValue.Elem().UnsafeAddr())

	return f.fastDeepCopy(arg, 0)
}

func (f *fastDeepCopy) cpyDefault(a *args, depth int) error {
	dst := a.dstType
	src := a.srcType
	dstAddr := a.dstAddr
	srcAddr := a.srcAddr
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

func (f *fastDeepCopy) cpyPtr(a *args, depth int) error {
	dst := a.dstType
	src := a.srcType

	if dst.Kind() != src.Kind() {
		return nil
	}

	arg := argsPool.Get().(*args)
	defer argsPool.Put(arg)

	arg.dstType = dst.Elem()
	arg.srcType = src.Elem()
	arg.dstAddr = a.dstAddr
	arg.srcAddr = a.srcAddr

	//dst = dst.Elem()
	//src = src.Elem()

	return f.fastDeepCopy(arg, depth)

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
func (f *fastDeepCopy) cpySliceArray(a *args, depth int) error {

	dst := a.dstType
	src := a.srcType
	dstAddr := a.dstAddr
	srcAddr := a.srcAddr
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

		err := func() error {
			arg := argsPool.Get().(*args)
			defer argsPool.Put(arg)

			arg.dstType = dst.Elem()
			arg.srcType = src.Elem()
			arg.dstAddr = dstElemAddr
			arg.srcAddr = srcElemAddr
			return f.fastDeepCopy(arg, depth)
		}()

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

func (f *fastDeepCopy) cpyMap(a *args, depth int) error {
	dst := a.dstType
	src := a.srcType
	dstAddr := a.dstAddr
	srcAddr := a.srcAddr

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
		err := func() error {
			arg := argsPool.Get().(*args)
			defer argsPool.Put(arg)

			arg.dstType = newKey.Type()
			arg.srcType = k.Type()
			arg.dstAddr = getPtrFromVal(&newKey)
			arg.srcAddr = getPtrFromVal(&k)

			return f.fastDeepCopy(arg, depth)
		}()
		if err != nil {
			return err
		}

		newVal := reflect.New(v.Type()).Elem()

		err = func() error {
			arg := argsPool.Get().(*args)
			defer argsPool.Put(arg)

			arg.dstType = newVal.Type()
			arg.srcType = v.Type()
			arg.dstAddr = getPtrFromVal(&newVal)
			arg.srcAddr = getPtrFromVal(&v)
			return f.fastDeepCopy(arg, depth)
		}()
		if err != nil {
			return err
		}

		dstVal.SetMapIndex(newKey, newVal)
	}

	return nil
}

func (f *fastDeepCopy) cpyStruct(a *args, depth int) error {

	dst := a.dstType
	src := a.srcType
	dstAddr := a.dstAddr
	srcAddr := a.srcAddr

	n := src.NumField()
	for i := 0; i < n; i++ {

		err := func() error {
			sf := src.Field(i)
			if sf.PkgPath != "" && !sf.Anonymous {
				return nil
			}

			if len(f.tagName) > 0 && !haveTagName(sf.Tag.Get(f.tagName)) {
				return nil
			}

			dstSf, ok := dst.FieldByName(sf.Name)
			if !ok {
				return nil
			}

			srcFieldAddr := unsafe.Pointer(uintptr(srcAddr) + sf.Offset)
			dstFieldAddr := unsafe.Pointer(uintptr(dstAddr) + dstSf.Offset)

			arg := argsPool.Get().(*args)
			defer argsPool.Put(arg)

			arg.dstType = dstSf.Type
			arg.srcType = sf.Type
			arg.dstAddr = dstFieldAddr
			arg.srcAddr = srcFieldAddr

			return f.fastDeepCopy(arg, depth+1)
		}()

		if err != nil {
			return err
		}
	}

	return nil
}

func (f *fastDeepCopy) cpyInterface(a *args, depth int) error {
	dst := a.dstType
	src := a.srcType
	dstAddr := a.dstAddr
	srcAddr := a.srcAddr

	if dst.Kind() != src.Kind() {
		return nil
	}

	dstInterfaceValue := typePtrToValue(dst, dstAddr)
	srcInterfaceValue := typePtrToValue(src, srcAddr)

	srcVal := srcInterfaceValue.Elem()

	newDst := reflect.New(srcVal.Type()).Elem()

	if srcVal.CanAddr() {
		arg := argsPool.Get().(*args)
		defer argsPool.Put(arg)

		arg.dstType = newDst.Type()
		arg.srcType = srcVal.Type()
		arg.dstAddr = unsafe.Pointer(newDst.UnsafeAddr())
		arg.srcAddr = unsafe.Pointer(srcVal.UnsafeAddr())

		if err := f.fastDeepCopy(arg, depth); err != nil {
			return err
		}
	}

	newDst.Set(srcVal)

	dstInterfaceValue.Set(newDst)
	return nil
}

func (f *fastDeepCopy) fastDeepCopy(a *args, depth int) error {
	if f.err != nil {
		return f.err
	}

	if f.maxDepth != noDepthLimited && depth > f.maxDepth {
		return nil
	}

	switch a.srcType.Kind() {
	case reflect.Slice, reflect.Array:
		return f.cpySliceArray(a, depth)
	case reflect.Map:
		return f.cpyMap(a, depth)
	case reflect.Struct:
		return f.cpyStruct(a, depth)
	case reflect.Interface:
		return f.cpyInterface(a, depth)
	case reflect.Ptr:
		return f.cpyPtr(a, depth)
	default:
		return f.cpyDefault(a, depth)
	}

	return nil
}
