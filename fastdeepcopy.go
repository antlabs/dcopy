package fastdeepcopy

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

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
	return f.fastDeepCopy(f.dstValue.Elem().Type(), f.srcValue.Elem().Type(), unsafe.Pointer(f.dstValue.Elem().UnsafeAddr()), unsafe.Pointer(f.srcValue.Elem().UnsafeAddr()))
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

func (f *fastDeepCopy) cpyStruct(dst, src reflect.Type, dstAddr, srcAddr unsafe.Pointer) error {

	n := src.NumField()
	for i := 0; i < n; i++ {

		sf := src.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

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
	default:
		return f.cpyDefault(dst, src, dstAddr, srcAddr)
	}
}
