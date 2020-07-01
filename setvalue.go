package fastdeepcopy

import (
	"reflect"
	"unsafe"
)

type setFunc func(dstAddr, srcAddr unsafe.Pointer)
type setFuncTab map[reflect.Kind]setFunc

var copyTab = setFuncTab{
	reflect.Int:    setInt,
	reflect.Int8:   setInt8,
	reflect.Int16:  setInt16,
	reflect.Int32:  setInt32,
	reflect.Int64:  setInt64,
	reflect.Uint:   setUint,
	reflect.Uint8:  setUint8,
	reflect.Uint16: setUint16,
	reflect.Uint32: setUint32,
	reflect.Uint64: setUint64,
}

func getSetFunc(t reflect.Kind) setFunc {
	f, _ := copyTab[t]
	return f
}

func setInt(dstAddr, srcAddr unsafe.Pointer) {
	*(*int)(dstAddr) = *(*int)(srcAddr)
}

func setInt8(dstAddr, srcAddr unsafe.Pointer) {
	*(*int8)(dstAddr) = *(*int8)(srcAddr)
}

func setInt16(dstAddr, srcAddr unsafe.Pointer) {
	*(*int16)(dstAddr) = *(*int16)(srcAddr)
}

func setInt32(dstAddr, srcAddr unsafe.Pointer) {
	*(*int32)(dstAddr) = *(*int32)(srcAddr)
}

func setInt64(dstAddr, srcAddr unsafe.Pointer) {
	*(*int64)(dstAddr) = *(*int64)(srcAddr)
}

func setUint(dstAddr, srcAddr unsafe.Pointer) {
	*(*uint)(dstAddr) = *(*uint)(srcAddr)
}

func setUint8(dstAddr, srcAddr unsafe.Pointer) {
	*(*uint8)(dstAddr) = *(*uint8)(srcAddr)
}

func setUint16(dstAddr, srcAddr unsafe.Pointer) {
	*(*uint16)(dstAddr) = *(*uint16)(srcAddr)
}

func setUint32(dstAddr, srcAddr unsafe.Pointer) {
	*(*uint32)(dstAddr) = *(*uint32)(srcAddr)
}

func setUint64(dstAddr, srcAddr unsafe.Pointer) {
	*(*uint64)(dstAddr) = *(*uint64)(srcAddr)
}