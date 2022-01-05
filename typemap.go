package typemap

import (
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	fillFactor       = 0.6
	slowHitThreshold = 128
	ptrsize          = unsafe.Sizeof(uintptr(0))
)

// TypeMap provides a lockless copy-on-write map mainly to use for type
// information cache, such as runtime generated encoders and decoders.
//
// TypeMap is safe to use concurrently, it will grow as needed.
type TypeMap struct {
	m unsafe.Pointer // *interfaceMap

	lock uint32
	m2   sync.Map // uintptr -> *dirtyEntry

	slowHit uint32
}

type dirtyEntry struct {
	once sync.Once
	err  error
	val  atomic.Value // interface{}
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
//
// This is the fast path, it will be inlined into the caller.
func (m *TypeMap) GetByType(key reflect.Type) interface{} {

	// type iface { tab  *itab, data unsafe.Pointer }

	//typeptr := (*(*[2]uintptr)(unsafe.Pointer(&key)))[1]
	//imap := (*interfaceMap)(atomic.LoadPointer(&m.m))
	//return imap.Get(int64(typeptr))

	return (*interfaceMap)(atomic.LoadPointer(&m.m)).Get(int64((*(*[2]uintptr)(unsafe.Pointer(&key)))[1]))
}

// GetByUintptr returns value for the given uintptr key.
// If key is not found in the map, it returns nil.
//
// This is the fast path, it will be inlined into the caller.
func (m *TypeMap) GetByUintptr(key uintptr) interface{} {

	//imap := (*interfaceMap)(atomic.LoadPointer(&m.m))
	//return imap.Get(int64(key))

	return (*interfaceMap)(atomic.LoadPointer(&m.m)).Get(int64(key))
}

// SetByType checks whether the given key is in the slow path,
// if the key exists it returns the cached value, else it builds the value
// by calling f, it then caches and returns the value.
//
// By accepting a function instead of a pre-built value, it guarantees that
// f will be called exactly once, which may be expensive.
//
// This function will trigger a calibrating to move data from the slow path
// to the fast path if needed.
func (m *TypeMap) SetByType(key reflect.Type, f func() (interface{}, error)) (interface{}, error) {
	// type iface { tab  *itab, data unsafe.Pointer }
	typeptr := (*(*[2]uintptr)(unsafe.Pointer(&key)))[1]
	return m.SetByUintptr(typeptr, f)
}

// SetByUintptr checks whether the given key is in the slow path,
// if the key exists it returns the cached value, else it builds the value
// by calling f, it then caches and returns the value.
//
// By accepting a function instead of a pre-built value, it guarantees that
// f will be called exactly once, which may be expensive.
//
// This function will trigger a calibrating to move data from the slow path
// to the fast path if needed.
func (m *TypeMap) SetByUintptr(key uintptr, f func() (interface{}, error)) (interface{}, error) {
	x, _ := m.m2.LoadOrStore(key, &dirtyEntry{})
	entry := x.(*dirtyEntry)
	entry.once.Do(func() {
		val, err := f()
		entry.err = err
		if err == nil {
			entry.val.Store(val)
		}
	})
	err := entry.err
	if err != nil {
		return nil, err
	}
	val := entry.val.Load()
	if atomic.AddUint32(&m.slowHit, 1) > slowHitThreshold {
		m.calibrate(false)
	}
	return val, nil
}

func (m *TypeMap) calibrate(wait bool) {
	if !atomic.CompareAndSwapUint32(&m.lock, 0, 1) {
		return
	}

	atomic.StoreUint32(&m.slowHit, 0)
	done := make(chan struct{})

	go func() {
		var newMap *interfaceMap
		imap := (*interfaceMap)(atomic.LoadPointer(&m.m))
		keys := make([]interface{}, 0)
		m.m2.Range(func(key, value interface{}) bool {
			if imap.Get(int64(key.(uintptr))) == nil {
				entry := value.(*dirtyEntry)
				val := entry.val.Load()
				if val != nil {
					if newMap == nil {
						newMap = imap.Copy()
					}
					newMap.Set(int64(key.(uintptr)), val)
					keys = append(keys, key)
				}
			} else {
				keys = append(keys, key)
			}
			return true
		})
		if newMap != nil {
			atomic.StorePointer(&m.m, unsafe.Pointer(newMap))
		}
		for _, k := range keys {
			m.m2.Delete(k)
		}
		atomic.StoreUint32(&m.lock, 0)
		close(done)
	}()

	// help testing
	if wait {
		<-done
	}
}
