package typemap

import (
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	fillFactor = 0.6
	ptrsize    = unsafe.Sizeof(uintptr(0))
)

// TypeMap provides a lockless copy-on-write map mainly to use for type
// information cache, such as runtime generated encoders and decoders.
// TypeMap is safe to use concurrently, when SetByUintptr, SetByType are
// called, the underlying map will be copied.
//
// The fill factor used for TypeMap is 0.6. A TypeMap will grow as needed.
type TypeMap struct {
	m unsafe.Pointer // *interfaceMap

	lock uint32
	m2   sync.Map
}

// New returns a new TypeMap with 8 as initial capacity.
func New() *TypeMap {
	size := 8
	imap := newInterfaceMap(size, fillFactor)
	return &TypeMap{m: unsafe.Pointer(imap)}
}

// Size returns size of the map.
func (m *TypeMap) Size() int {
	return (*interfaceMap)(atomic.LoadPointer(&m.m)).Size()
}

// GetByType returns value for the given reflect.Type.
// If key is not found in the map, it returns nil.
func (m *TypeMap) GetByType(key reflect.Type) interface{} {

	// type iface { tab  *itab, data unsafe.Pointer }

	//typeptr := (*(*[2]uintptr)(unsafe.Pointer(&key)))[1]
	//imap := (*interfaceMap)(atomic.LoadPointer(&m.m))
	//return imap.Get(int64(typeptr))

	return (*interfaceMap)(atomic.LoadPointer(&m.m)).Get(int64((*(*[2]uintptr)(unsafe.Pointer(&key)))[1]))
}

// GetByUintptr returns value for the given uintptr key.
// If key is not found in the map, it returns nil.
func (m *TypeMap) GetByUintptr(key uintptr) interface{} {

	//imap := (*interfaceMap)(atomic.LoadPointer(&m.m))
	//return imap.Get(int64(key))

	return (*interfaceMap)(atomic.LoadPointer(&m.m)).Get(int64(key))
}

// SetByType adds or updates value to the map using reflect.Type key.
// If the key value is not present in the underlying map, it will copy the
// map and add the key value to the copy, then swap to the new map using
// atomic operation.
func (m *TypeMap) SetByType(key reflect.Type, val interface{}) {
	// type iface { tab  *itab, data unsafe.Pointer }
	typeptr := (*(*[2]uintptr)(unsafe.Pointer(&key)))[1]
	m.SetByUintptr(typeptr, val)
}

// SetByUintptr adds or updates value to the map using uintptr key.
// If the key value is not present in the underlying map, it will copy the
// map and add the key value to the copy, then swap to the new map using
// atomic operation.
func (m *TypeMap) SetByUintptr(key uintptr, val interface{}) {
	m.m2.Store(key, val)
	if atomic.CompareAndSwapUint32(&m.lock, 0, 1) {
		go m.calibrate()
	}
}

func (m *TypeMap) calibrate() {
	var newMap *interfaceMap
	imap := (*interfaceMap)(atomic.LoadPointer(&m.m))
	keys := make([]interface{}, 0)
	m.m2.Range(func(key, value interface{}) bool {
		if imap.Get(int64(key.(uintptr))) == nil {
			if newMap == nil {
				newMap = imap.Copy()
			}
			newMap.Set(int64(key.(uintptr)), value)
			keys = append(keys, key)
		}
		return true
	})
	if newMap != nil {
		atomic.StorePointer(&m.m, unsafe.Pointer(newMap))
		for _, k := range keys {
			m.m2.Delete(k)
		}
	}
	atomic.StoreUint32(&m.lock, 0)
}
