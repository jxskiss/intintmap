package typemap

import "testing"

func TestInterfaceMapSimple(t *testing.T) {
	m := newInterfaceMap(10, 0.99)
	var i int64
	var v interface{}

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

	m0 := make(map[int64]int64, 1000)
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

	m0 = make(map[int64]int64, 1000)
	for i = 1; i < 20001; i += 2 {
		m0[i] = i
	}
	n = len(m0)

	for _, kv := range m.Items() {
		m0[kv.K] = -(kv.V.(int64))
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
