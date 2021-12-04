package typemap

import (
	"math"
	"unsafe"
)

// INT_PHI is for scrambling the keys.
const INT_PHI = 0x9E3779B9

// FREE_KEY is the 'free' key.
const FREE_KEY = 0

func phiMix(x int64) uint64 {
	h := x * INT_PHI
	return uint64(h ^ (h >> 16))
}

func nextPowerOfTwo(x int) int {
	if x == 0 {
		return 1
	}

	x--
	x |= x >> 1
	x |= x >> 2
	x |= x >> 4
	x |= x >> 8
	x |= x >> 16

	return x + 1
}

func arraySize(exp int, fill float64) int {
	s := int(math.Ceil(float64(exp) / fill))
	s = nextPowerOfTwo(s)
	if s < 2 {
		s = 2
	}
	return s
}

func calcThreshold(capacity int, fillFactor float64) int {
	return int(math.Floor(float64(capacity) * fillFactor))
}

// interfaceEntry represents a key value pair in a interfaceMap or TypeMap.
type interfaceEntry struct {
	K int64
	V interface{}
}

func newInterfaceMap(size int, fillFactor float64) *interfaceMap {
	capacity := arraySize(size, fillFactor)
	threshold := calcThreshold(capacity, fillFactor)
	mask := capacity - 1
	data := make([]interfaceEntry, capacity)
	return &interfaceMap{
		data:       data,
		dptr:       unsafe.Pointer(&data[0]),
		fillFactor: fillFactor,
		threshold:  threshold,
		size:       0,
		mask:       uint64(mask),
	}
}

type interfaceMap struct {
	data []interfaceEntry
	dptr unsafe.Pointer

	fillFactor float64
	threshold  int
	size       int
	mask       uint64
}

// Size returns the size of the map.
func (m *interfaceMap) Size() int {
	return m.size
}

// getK helps to eliminate slice bounds checking
func (m *interfaceMap) getK(ptr uint64) *int64 {
	return (*int64)(unsafe.Pointer(uintptr(m.dptr) + uintptr(ptr)*3*ptrsize))
}

// getV helps to eliminate slice bounds checking
func (m *interfaceMap) getV(ptr uint64) *interface{} {
	return (*interface{})(unsafe.Pointer(uintptr(m.dptr) + uintptr(ptr)*3*ptrsize + ptrsize))
}

// Get returns the value if the key is found, else it returns nil.
// It will be inlined by the compiler.
func (m *interfaceMap) Get(key int64) interface{} {
	// manually inline phiMix to help inlining
	h := uint64(key) * INT_PHI
	ptr := h ^ (h >> 16)

	for {
		ptr &= m.mask
		// manually inline m.getK and m.getV
		k := *(*int64)(unsafe.Pointer(uintptr(m.dptr) + uintptr(ptr)*3*ptrsize))
		if k == key {
			return *(*interface{})(unsafe.Pointer(uintptr(m.dptr) + uintptr(ptr)*3*ptrsize + ptrsize))
		}
		if k == 0 {
			return nil
		}
		ptr += 1
	}
}

// Has tells whether a key is found in the map.
// It will be inlined into the callers.
func (m *interfaceMap) Has(key int64) bool {
	// manually inline phiMix to help inlining
	h := uint64(key) * INT_PHI
	ptr := h ^ (h >> 16)

	for {
		ptr &= m.mask
		// manually inline m.getK and m.getV
		k := *(*int64)(unsafe.Pointer(uintptr(m.dptr) + uintptr(ptr)*3*ptrsize))
		if k == key {
			return true
		}
		if k == 0 {
			return false
		}
		ptr += 1
	}
}

// Set adds or updates key with value to the interfaceMap.
func (m *interfaceMap) Set(key int64, val interface{}) {
	ptr := phiMix(key)
	for {
		ptr &= m.mask
		k := *m.getK(ptr)
		if k == FREE_KEY {
			*m.getK(ptr) = key
			*m.getV(ptr) = val
			if m.size >= m.threshold {
				m.rehash()
			} else {
				m.size++
			}
			return
		}
		if k == key {
			*m.getV(ptr) = val
			return
		}
		ptr += 1
	}
}

func (m *interfaceMap) rehash() {
	newCapacity := len(m.data) * 2
	m.threshold = calcThreshold(newCapacity, m.fillFactor)
	m.mask = uint64(newCapacity - 1)

	data := m.data
	m.data = make([]interfaceEntry, newCapacity)
	m.dptr = unsafe.Pointer(&m.data[0])
	m.size = 0

	var i int64
COPY:
	for i = 0; i < int64(len(data)); i++ {
		e := data[i]
		if e.K == FREE_KEY {
			continue
		}

		// Manually inline the Set function to avoid unnecessary calculation.
		ptr := phiMix(e.K)
		for {
			ptr &= m.mask
			k := *m.getK(ptr)
			if k == FREE_KEY {
				*m.getK(ptr) = e.K
				*m.getV(ptr) = e.V
				m.size++
				continue COPY
			}
			ptr += 1
		}
	}
}

func (m *interfaceMap) Delete(key int64) {
	ptr := phiMix(key)
	for {
		ptr &= m.mask
		k := *m.getK(ptr)
		if k == key {
			m.shiftKeys(ptr)
			m.size--
			return
		}
		if k == FREE_KEY {
			return
		}
		ptr += 1
	}
}

func (m *interfaceMap) shiftKeys(pos uint64) uint64 {
	var last, slot uint64
	var k int64
	for {
		last = pos
		pos = last + 1
		for {
			pos &= m.mask
			k = *m.getK(pos)
			if k == FREE_KEY {
				*m.getK(last) = FREE_KEY
				*m.getV(last) = nil
				return last
			}

			slot = phiMix(k) & m.mask
			if last <= pos {
				if last >= slot || slot > pos {
					break
				}
			} else {
				if last >= slot && slot > pos {
					break
				}
			}
			pos += 1
		}
		*(m.getK(last)) = *m.getK(pos)
		*(m.getV(last)) = *m.getV(pos)
	}
}

// Copy returns a copy of a interfaceMap, if the map's size triggers it's
// threshold, the new map's capacity will be twice of the old.
func (m *interfaceMap) Copy() *interfaceMap {
	capacity := cap(m.data)
	if m.size >= m.threshold {
		capacity *= 2
	}
	mask := capacity - 1
	data := make([]interfaceEntry, capacity)
	newMap := &interfaceMap{
		data:       data,
		dptr:       unsafe.Pointer(&data[0]),
		fillFactor: m.fillFactor,
		threshold:  m.threshold,
		size:       0,
		mask:       uint64(mask),
	}
	for _, e := range m.data {
		if e.K == FREE_KEY {
			continue
		}
		newMap.Set(e.K, e.V)
	}
	return newMap
}

func (m *interfaceMap) Keys() []int64 {
	keys := make([]int64, 0, m.size+1)
	data := m.data
	for i := 0; i < len(data); i++ {
		if data[i].K == FREE_KEY {
			continue
		}
		keys = append(keys, data[i].K)
	}
	return keys
}

func (m *interfaceMap) Items() []interfaceEntry {
	items := make([]interfaceEntry, 0, m.size+1)
	data := m.data
	for i := 0; i < len(data); i++ {
		if data[i].K == FREE_KEY {
			continue
		}
		items = append(items, data[i])
	}
	return items
}
