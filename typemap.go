package phimap

import (
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"
)

const slowHitThreshold = 128

// TypeMap is a lockless copy-on-write map designed for type
// information cache, such as runtime generated encoders and decoders.
//
// TypeMap is safe to use concurrently, it grows as needed.
type TypeMap[T any] struct {
	m unsafe.Pointer // *PhiMap[T]

	lock uint32
	m2   sync.Map // uint64 -> *dirtyEntry

	slowHit uint32
}

type dirtyEntry struct {
	once sync.Once
	err  error
	val  atomic.Value // any
}

// NewTypeMap creates a new TypeMap.
func NewTypeMap[T any]() *TypeMap[T] {
	imap := NewPhiMap[T]()
	return &TypeMap[T]{m: unsafe.Pointer(imap)}
}

// Size returns size of the map.
func (m *TypeMap[T]) Size() int {
	return (*PhiMap[T])(atomic.LoadPointer(&m.m)).Size()
}

// GetByType returns value for the given reflect.Type.
// If key is not found in the map, it returns zero value of T.
//
// This is the fast path, it is optimized to be inline-able.
func (m *TypeMap[T]) GetByType(key reflect.Type) T {

	// type iface { tab  *itab, data unsafe.Pointer }

	/*
		typeptr := (*(*[2]uintptr)(unsafe.Pointer(&key)))[1]
		imap := (*PhiMap)(atomic.LoadPointer(&m.m))
		return imap.Get(uint64(typeptr))
	*/
	return (*PhiMap[T])(atomic.LoadPointer(&m.m)).
		Get(uint64((*(*[2]uintptr)(unsafe.Pointer(&key)))[1]))
}

// GetByUintptr returns value for the given uintptr key.
// If key is not found in the map, it returns zero value of T.
//
// This is the fast path, it is optimized to be inline-able.
func (m *TypeMap[T]) GetByUintptr(key uintptr) T {
	/*
		imap := (*PhiMap)(atomic.LoadPointer(&m.m))
		return imap.Get(uint64(key))
	*/
	return (*PhiMap[T])(atomic.LoadPointer(&m.m)).Get(uint64(key))
}

// SetByType checks whether the given key is in the slow path,
// if the key exists it returns the cached value, else it builds the value
// by calling f, it then caches and returns the value.
//
// By accepting a function instead of a pre-built value, it guarantees that
// f is called exactly once to avoid unnecessary cost, which may be expensive.
//
// This function triggers a calibrating to move data from the slow path
// to the fast path if needed.
func (m *TypeMap[T]) SetByType(key reflect.Type, f func() (T, error)) (T, error) {
	// type iface { tab  *itab, data unsafe.Pointer }
	typeptr := (*(*[2]uintptr)(unsafe.Pointer(&key)))[1]
	return m.SetByUintptr(typeptr, f)
}

// SetByUintptr checks whether the given key is in the slow path,
// if the key exists it returns the cached value, else it builds the value
// by calling f, it then caches and returns the value.
//
// By accepting a function instead of a pre-built value, it guarantees that
// f is called exactly once to avoid unnecessary cost, which may be expensive.
//
// This function triggers a calibrating to move data from the slow path
// to the fast path if needed.
func (m *TypeMap[T]) SetByUintptr(key uintptr, f func() (T, error)) (T, error) {
	var zero T
	x, _ := m.m2.LoadOrStore(uint64(key), &dirtyEntry{})
	called := false
	entry := x.(*dirtyEntry)
	entry.once.Do(func() {
		called = true
		val, err := f()
		entry.err = err
		if err == nil {
			entry.val.Store(val)
		}
	})
	val := entry.val.Load()
	if val == nil { // error occurred
		if called {
			return zero, entry.err
		}
		val1, err1 := f() // retry calling f
		if err1 != nil {
			return zero, err1
		}
		entry.val.Store(val1)
		val = val1
	}
	if atomic.AddUint32(&m.slowHit, 1) > slowHitThreshold {
		m.calibrate(false)
	}
	return val.(T), nil
}

func (m *TypeMap[T]) calibrate(wait bool) {
	if !atomic.CompareAndSwapUint32(&m.lock, 0, 1) {
		return
	}

	atomic.StoreUint32(&m.slowHit, 0)
	done := make(chan struct{})

	go func() {
		var newMap *PhiMap[T]
		imap := (*PhiMap[T])(atomic.LoadPointer(&m.m))
		delKeys := make([]any, 0)
		m.m2.Range(func(key, value any) bool {
			if imap.Has(key.(uint64)) {
				delKeys = append(delKeys, key)
				return true
			}
			entry := value.(*dirtyEntry)
			val := entry.val.Load()
			if val != nil {
				if newMap == nil {
					newMap = imap.Copy()
				}
				newMap.Set(key.(uint64), val.(T))
				delKeys = append(delKeys, key)
			}
			return true
		})
		if newMap != nil {
			atomic.StorePointer(&m.m, unsafe.Pointer(newMap))
		}
		for _, k := range delKeys {
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
