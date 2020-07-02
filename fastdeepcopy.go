package fastdeepcopy

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

type emptyInterface struct {
	typ  *struct{}
	word unsafe.Pointer
}

type fastDeepCopy struct {
	dstValue reflect.Value
	srcValue reflect.Value
	err      error
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

	return &fastDeepCopy{dstValue: dstValue, srcValue: srcValue}
}

func (f *fastDeepCopy) Do() error {
	if f.err != nil {
		return f.err
	}

	return f.fastDeepCopy(f.dstValue.Elem().Type(), f.srcValue.Elem().Type(),
		unsafe.Pointer(f.dstValue.Elem().UnsafeAddr()),
		unsafe.Pointer(f.srcValue.Elem().UnsafeAddr()))
}

func (f *fastDeepCopy) cpyDefault(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer) error {
	if dst.Kind() != src.Kind() {
		return nil
	}

	set := getSetFunc(src.Kind())
	set(dstAddr, srcAddr)
	return nil
}

func (f *fastDeepCopy) cpyPtr(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer) error {
	if dst.Kind() != src.Kind() {
		return nil
	}

	dst = dst.Elem()
	src = src.Elem()

	f.fastDeepCopy(dst, src, dstAddr, srcAddr)
	return nil
}

func (f *fastDeepCopy) cpySliceArray(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer) error {
	if dst.Kind() != reflect.Slice {
		return nil
	}

	dstSliceHeader := (*reflect.SliceHeader)(dstAddr)
	srcSliceHeader := (*reflect.SliceHeader)(srcAddr)

	if srcSliceHeader.Len == 0 {
		return nil
	}

	if dstSliceHeader.Cap == 0 {
		// reflect.MakeSlice是不能取得UnsafeAddr()的指针的
		newAddrInterface := reflect.MakeSlice(src, srcSliceHeader.Len, srcSliceHeader.Len).Interface()
		slicePtr := (*emptyInterface)(unsafe.Pointer(&newAddrInterface)).word
		newSliceHeader := (*reflect.SliceHeader)(slicePtr)
		dstSliceHeader.Data = newSliceHeader.Data
		dstSliceHeader.Len = newSliceHeader.Len
		dstSliceHeader.Cap = newSliceHeader.Cap
	}

	l := srcSliceHeader.Len
	if dstSliceHeader.Cap < l {
		l = dstSliceHeader.Cap
	}

	elemType := dst.Elem()
	for i := 0; i < l; i++ {
		dstElemAddr := unsafe.Pointer(uintptr(dstSliceHeader.Data) + uintptr(i)*elemType.Size())
		srcElemAddr := unsafe.Pointer(uintptr(srcSliceHeader.Data) + uintptr(i)*elemType.Size())
		err := f.fastDeepCopy(src.Elem(), dst.Elem(), dstElemAddr, srcElemAddr)
		if err != nil {
			return err
		}
	}

	dstSliceHeader.Len = l
	return nil
}

func (f *fastDeepCopy) cpyStruct(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer) error {
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

		//dstSf, ok := dstMap[sf.Name]
		dstSf, ok := dst.FieldByName(sf.Name)
		if !ok {
			continue
		}

		err := f.fastDeepCopy(dstSf.Type, sf.Type, unsafe.Pointer(uintptr(dstAddr)+dstSf.Offset),
			unsafe.Pointer(uintptr(srcAddr)+sf.Offset))
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *fastDeepCopy) fastDeepCopy(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer) error {
	if f.err != nil {
		return f.err
	}

	switch src.Kind() {
	case reflect.Struct:
		return f.cpyStruct(dst, src, dstAddr, srcAddr)
	case reflect.Ptr:
		return f.cpyPtr(dst, src, dstAddr, srcAddr)
	case reflect.Slice:
		return f.cpySliceArray(dst, src, dstAddr, srcAddr)
	default:
		return f.cpyDefault(dst, src, dstAddr, srcAddr)
	}
}
