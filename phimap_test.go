package phimap

import (
	"context"
	"strconv"
	"testing"
)

func TestPhiMapSimple(t *testing.T) {
	m := NewPhiMap[any]()
	var i uint64
	var v any

	// --------------------------------------------------------------------
	// Set() and Get()

	for i = 1; i < 20001; i += 2 {
		m.Set(i, i)
	}
	for i = 1; i < 20001; i += 2 {
		if v = m.Get(i); v != i {
			t.Errorf("didn't get expected value")
		}
		if v = m.Get(i + 1); v != nil {
			t.Errorf("didn't get expected 'not found' flag")
		}
	}

	if m.Size() != 20000/2 {
		t.Errorf("size (%d) is not right, should be %d", m.Size(), 20000/2)
	}

	// --------------------------------------------------------------------
	// Keys()

	m0 := make(map[uint64]uint64, 1000)
	for i = 1; i < 20001; i += 2 {
		m0[i] = i
	}
	n := len(m0)

	for _, k := range m.Keys() {
		m0[k] = -k
	}
	if n != len(m0) {
		t.Errorf("get unexpected more keys")
	}

	for k, v := range m0 {
		if k != -v {
			t.Errorf("didn't get expected changed value")
		}
	}

	// --------------------------------------------------------------------
	// Items()

	m0 = make(map[uint64]uint64, 1000)
	for i = 1; i < 20001; i += 2 {
		m0[i] = i
	}
	n = len(m0)

	for _, kv := range m.Items() {
		m0[kv.K] = -(kv.V.(uint64))
		if kv.K != kv.V {
			t.Errorf("didn't get expected key-value pair")
		}
	}
	if n != len(m0) {
		t.Errorf("get unexpected more keys")
	}

	for k, v := range m0 {
		if k != -v {
			t.Errorf("didn't get expected changed value")
		}
	}

	// --------------------------------------------------------------------
	// Delete()

	for i = 1; i < 10001; i += 2 {
		m.Delete(i)
	}
	for i = 1; i < 10001; i += 2 {
		if v = m.Get(i); v != nil {
			t.Errorf("didn't get expected 'not found' flag")
		}
		if v = m.Get(i + 1); v != nil {
			t.Errorf("didn't get expected 'not found' flag")
		}
	}
	for i = 10001; i < 20001; i += 2 {
		if v = m.Get(i); v != i {
			t.Errorf("didn't get expected value")
		}
		if v = m.Get(i + 1); v != nil {
			t.Errorf("didn't get expected 'not found' flag")
		}
	}

	for i = 10001; i < 20001; i += 2 {
		m.Delete(i)
	}
	for i = 10001; i < 20001; i += 2 {
		if v = m.Get(i); v != nil {
			t.Errorf("didn't get expected 'not found' flag")
		}
	}

	// --------------------------------------------------------------------
	// Set() and Get()

	for i = 1; i < 20001; i += 2 {
		m.Set(i, i*2)
	}
	for i = 1; i < 20001; i += 2 {
		if v = m.Get(i); v != i*2 {
			t.Errorf("didn't get expected value")
		}
		if v = m.Get(i + 1); v != nil {
			t.Errorf("didn't get expected 'not found' flag")
		}
	}
}

type AStruct struct {
	A int64
	B string
	C func(ctx context.Context) (int64, error)
	D []byte
	E int
	F [32]int64
}

func TestPhiMap_Types(t *testing.T) {
	testData := make([]*AStruct, initSize)
	for i := 0; i < initSize; i++ {
		x := i
		obj := &AStruct{
			A: int64(x),
			B: strconv.Itoa(x),
			C: func(ctx context.Context) (int64, error) {
				return int64(x), nil
			},
			D: []byte(strconv.Itoa(x)),
			E: x,
		}
		for j := 0; j < len(obj.F); j++ {
			obj.F[j] = int64(x)
		}
		testData[x] = obj
	}

	t.Run("pointer", func(t *testing.T) {
		m := NewPhiMap[*AStruct]()
		for i := 0; i < initSize; i++ {
			m.Set(uint64(i), testData[i])
			for j := 0; j < i; j++ {
				got := m.Get(uint64(j))
				assertEqual(t, int64(j), got.A)
				assertEqual(t, strconv.Itoa(j), got.B)
				cRet, err := got.C(context.Background())
				if err != nil {
					t.Errorf("got unexpected error: %v", err)
				}
				assertEqual(t, int64(j), cRet)
				assertEqual(t, strconv.Itoa(j), string(got.D))
				assertEqual(t, j, got.E)
				assertEqual(t, 32, len(got.F))
				for k := 0; k < len(got.F); k++ {
					assertEqual(t, int64(j), got.F[k])
				}
			}
		}
	})

	t.Run("struct", func(t *testing.T) {
		m := NewPhiMap[AStruct]()
		for i := 0; i < initSize; i++ {
			m.Set(uint64(i), *testData[i])
			for j := 0; j < i; j++ {
				for j := 0; j < i; j++ {
					got := m.Get(uint64(j))
					assertEqual(t, int64(j), got.A)
					assertEqual(t, strconv.Itoa(j), got.B)
					cRet, err := got.C(context.Background())
					if err != nil {
						t.Errorf("got unexpected error: %v", err)
					}
					assertEqual(t, int64(j), cRet)
					assertEqual(t, strconv.Itoa(j), string(got.D))
					assertEqual(t, j, got.E)
					assertEqual(t, 32, len(got.F))
					for k := 0; k < len(got.F); k++ {
						assertEqual(t, int64(j), got.F[k])
					}
				}
			}
		}
	})
}

func assertEqual[T comparable](t *testing.T, left, right T) {
	t.Helper()
	if left != right {
		t.Errorf("values not equal, left= %v, right= %v", left, right)
	}
}
