package phimap

import (
	"errors"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"unsafe"
)

var (
	testTypeMapValues1 = []any{
		TestType1{},
		TestType2{},
		TestType3{},
		TestType4{},
		TestType5{},
		TestType6{},
	}
	testTypeMapValues2 = []any{
		TestType7{},
		TestType8{},
		TestType9{},
		TestType10{},
		TestType11{},
		TestType12{},
	}
)

func TestTypeMap(t *testing.T) {
	m := NewTypeMap[int]()
	builder := func(x int) func() (int, error) {
		return func() (int, error) { return x, nil }
	}

	for _, val := range testTypeMapValues1[:3] {
		ret, err := m.SetByType(reflect.TypeOf(val), builder(1))
		if err != nil {
			t.Errorf("got unexpected error: %v", err)
		}
		if ret != 1 {
			t.Errorf("expacted value 1, got %v", ret)
		}
	}
	for _, val := range testTypeMapValues1[3:] {
		typeptr := (*(*[2]uintptr)(unsafe.Pointer(&val)))[0]
		ret, err := m.SetByUintptr(typeptr, builder(1))
		if err != nil {
			t.Errorf("got unexpected error: %v", err)
		}
		if ret != 1 {
			t.Errorf("expected value 1, got %v", ret)
		}
	}

	m.calibrate(true)

	for _, val := range testTypeMapValues1 {
		got := m.GetByType(reflect.TypeOf(val))
		if got != 1 {
			t.Errorf("expected value 1, got %v", got)
		}
	}
	for _, val := range testTypeMapValues2 {
		got := m.GetByType(reflect.TypeOf(val))
		if got != 0 {
			t.Errorf("expected zero, got %v", got)
		}
	}
}

func TestTypeMap_Error(t *testing.T) {
	m := NewTypeMap[int]()
	builderErr := func() func() (int, error) {
		return func() (int, error) { return 0, errors.New("test error") }
	}
	builderSucc := func(x int) func() (int, error) {
		return func() (int, error) { return x, nil }
	}

	var wg sync.WaitGroup
	for i := 0; i < 5000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			j := rand.Intn(len(testTypeMapValues1))
			ret, err := m.SetByType(reflect.TypeOf(testTypeMapValues1[j]), builderErr())
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			if ret != 0 {
				t.Errorf("expected value 0, got %v", ret)
			}
		}()
	}
	for i := 0; i < 5000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			j := rand.Intn(len(testTypeMapValues2))
			ret, err := m.SetByType(reflect.TypeOf(testTypeMapValues2[j]), builderSucc(1))
			if err != nil {
				t.Errorf("got unexpected error: %v", err)
			}
			if ret != 1 {
				t.Errorf("expected value 1, got %v", ret)
			}
		}()
	}
	wg.Wait()

	m.calibrate(true)

	for _, val := range testTypeMapValues1 {
		got := m.GetByType(reflect.TypeOf(val))
		if got != 0 {
			t.Errorf("expected value 0, got %v", got)
		}
	}
	for _, val := range testTypeMapValues2 {
		got := m.GetByType(reflect.TypeOf(val))
		if got != 1 {
			t.Errorf("expected value 1, got %v", got)
		}
	}
}

type TestType1 struct{ A int }
type TestType2 struct{ B int32 }
type TestType3 struct{ C int64 }
type TestType4 struct{ D int8 }
type TestType5 struct{ E int }
type TestType6 struct{ F int }
type TestType7 struct{ G string }
type TestType8 struct{ H []byte }
type TestType9 struct{ I string }
type TestType10 struct{ J uint }
type TestType11 struct{ K uint }
type TestType12 struct{ L uint }
