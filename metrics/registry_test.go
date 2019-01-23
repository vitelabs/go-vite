package metrics

import (
	"reflect"
	"sync"
	"testing"
)

func BenchmarkRegistry(b *testing.B) {
	r := NewRegistry()
	r.Register("foo", NewCounter())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Each(func(string, interface{}) {})
	}
}

func BenchmarkRegistryParallel(b *testing.B) {
	r := NewRegistry()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.GetOrRegister("foo", NewCounter())
		}
	})
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	r.Register("foo", NewCounter())
	i := 0
	r.Each(func(name string, iface interface{}) {
		i++
		if "foo" != name {
			t.Fatal(name)
		}
		if _, ok := iface.(Counter); !ok {
			t.Fatal(iface)
		}
	})
	if 1 != i {
		t.Fatal(i)
	}
	r.Unregister("foo")
	i = 0
	r.Each(func(string, interface{}) { i++ })
	if 0 != i {
		t.Fatal(i)
	}
}

func TestRegistryDuplicate(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("foo", NewCounter()); nil != err {
		t.Fatal(err)
	}
	if err := r.Register("foo", NewGauge()); nil == err {
		t.Fatal(err)
	}
	i := 0
	r.Each(func(name string, iface interface{}) {
		i++
		if _, ok := iface.(Counter); !ok {
			t.Fatal(iface)
		}
	})
	if 1 != i {
		t.Fatal(i)
	}
}

func TestRegistryGet(t *testing.T) {
	r := NewRegistry()
	r.Register("foo", NewCounter())
	if count := r.Get("foo").(Counter).Count(); 0 != count {
		t.Fatal(count)
	}
	r.Get("foo").(Counter).Inc(1)
	if count := r.Get("foo").(Counter).Count(); 1 != count {
		t.Fatal(count)
	}
}

func TestRegistryGetOrRegister(t *testing.T) {
	r := NewRegistry()

	// First metric wins with GetOrRegister
	_ = r.GetOrRegister("foo", NewCounter())
	m := r.GetOrRegister("foo", NewGauge())
	if _, ok := m.(Counter); !ok {
		t.Fatal(m)
	}

	i := 0
	r.Each(func(name string, iface interface{}) {
		i++
		if name != "foo" {
			t.Fatal(name)
		}
		if _, ok := iface.(Counter); !ok {
			t.Fatal(iface)
		}
	})
	if i != 1 {
		t.Fatal(i)
	}
}

func TestGetOrRegister_NilMetrics(t *testing.T) {
	r := NewRegistry()
	m := r.GetOrRegister("Oooooops", newStandardMeter())
	if _, ok := m.(*NilMeter); ok {
		t.Log("new object is a nil object")
		return
	}
	if _, ok := m.(*StandardMeter); !ok {
		t.Fatal(m)
	}
	r.Each(func(name string, iface interface{}) {
		if meter, ok := iface.(*StandardMeter); ok {
			meter.Mark(1)
			t.Logf("name:%v meter.Count():%v\n", name, meter.Count())
			if m != meter {
				t.Fatal("Meter references don't match")
			}
		}
	})
	t.Logf("meter.Count():%v\n", r.Get("Oooooops").(*StandardMeter).Snapshot().Count())
}

func TestRegistryGetOrRegisterWithLazyInstantiation(t *testing.T) {
	r := NewRegistry()

	// First metric wins with GetOrRegister
	_ = r.GetOrRegister("foo", NewCounter)
	m := r.GetOrRegister("foo", NewGauge)
	if _, ok := m.(Counter); !ok {
		t.Fatal(m)
	}

	i := 0
	r.Each(func(name string, iface interface{}) {
		i++
		if name != "foo" {
			t.Fatal(name)
		}
		if _, ok := iface.(Counter); !ok {
			t.Fatal(iface)
		}
	})
	if i != 1 {
		t.Fatal(i)
	}
}

func TestRegistryUnregister(t *testing.T) {
	l := len(arbiter.meters)
	r := NewRegistry()
	r.Register("foo", NewCounter())
	r.Register("bar", NewMeter())
	r.Register("baz", NewTimer())
	if len(arbiter.meters) != l+2 {
		t.Errorf("arbiter.meters: %d != %d\n", l+2, len(arbiter.meters))
	}
	r.Unregister("foo")
	r.Unregister("bar")
	r.Unregister("baz")
	if len(arbiter.meters) != l {
		t.Errorf("arbiter.meters: %d != %d\n", l+2, len(arbiter.meters))
	}
}

func TestPrefixedChildRegistryGetOrRegister(t *testing.T) {
	r := NewRegistry()
	pr := NewPrefixedChildRegistry(r, "prefix.")

	_ = pr.GetOrRegister("foo", NewCounter())

	i := 0
	r.Each(func(name string, m interface{}) {
		i++
		if name != "prefix.foo" {
			t.Fatal(name)
		}
	})
	if i != 1 {
		t.Fatal(i)
	}
}

func TestPrefixedRegistryGetOrRegister(t *testing.T) {
	r := NewPrefixedRegistry("prefix.")

	_ = r.GetOrRegister("foo", NewCounter())

	i := 0
	r.Each(func(name string, m interface{}) {
		i++
		if name != "prefix.foo" {
			t.Fatal(name)
		}
	})
	if i != 1 {
		t.Fatal(i)
	}
}

func TestPrefixedRegistryRegister(t *testing.T) {
	r := NewPrefixedRegistry("prefix.")
	err := r.Register("foo", NewCounter())
	c := NewCounter()
	Register("bar", c)
	if err != nil {
		t.Fatal(err.Error())
	}

	i := 0
	r.Each(func(name string, m interface{}) {
		i++
		if name != "prefix.foo" {
			t.Fatal(name)
		}
	})
	if i != 1 {
		t.Fatal(i)
	}
}

func TestPrefixedRegistryUnregister(t *testing.T) {
	r := NewPrefixedRegistry("prefix.")

	_ = r.Register("foo", NewCounter())

	i := 0
	r.Each(func(name string, m interface{}) {
		i++
		if name != "prefix.foo" {
			t.Fatal(name)
		}
	})
	if i != 1 {
		t.Fatal(i)
	}

	r.Unregister("foo")

	i = 0
	r.Each(func(name string, m interface{}) {
		i++
	})

	if i != 0 {
		t.Fatal(i)
	}
}

func TestPrefixedRegistryGet(t *testing.T) {
	pr := NewPrefixedRegistry("prefix.")
	name := "foo"
	pr.Register(name, NewCounter())

	fooCounter := pr.Get(name)
	if fooCounter == nil {
		t.Fatal(name)
	}
}

func TestPrefixedChildRegistryGet(t *testing.T) {
	r := NewRegistry()
	pr := NewPrefixedChildRegistry(r, "prefix.")
	name := "foo"
	pr.Register(name, NewCounter())
	fooCounter := pr.Get(name)
	if fooCounter == nil {
		t.Fatal(name)
	}
}

func TestChildPrefixedRegistryRegister(t *testing.T) {
	r := NewPrefixedChildRegistry(DefaultRegistry, "prefix.")
	err := r.Register("foo", NewCounter())
	c := NewCounter()
	Register("bar", c)
	if err != nil {
		t.Fatal(err.Error())
	}

	i := 0
	r.Each(func(name string, m interface{}) {
		i++
		if name != "prefix.foo" {
			t.Fatal(name)
		}
	})
	if i != 1 {
		t.Fatal(i)
	}
}

func TestChildPrefixedRegistryOfChildRegister(t *testing.T) {
	r := NewPrefixedChildRegistry(NewRegistry(), "prefix.")
	r2 := NewPrefixedChildRegistry(r, "prefix2.")
	err := r.Register("foo2", NewCounter())
	if err != nil {
		t.Fatal(err.Error())
	}
	err = r2.Register("baz", NewCounter())
	c := NewCounter()
	Register("bars", c)

	i := 0
	r2.Each(func(name string, m interface{}) {
		i++
		if name != "prefix.prefix2.baz" {
			t.Fatal(name)
		}
	})
	if i != 1 {
		t.Fatal(i)
	}
}

func TestWalkRegistries(t *testing.T) {
	r := NewPrefixedChildRegistry(NewRegistry(), "prefix.")
	r2 := NewPrefixedChildRegistry(r, "prefix2.")
	err := r.Register("foo2", NewCounter())
	if err != nil {
		t.Fatal(err.Error())
	}
	err = r2.Register("baz", NewCounter())
	c := NewCounter()
	Register("bars", c)

	_, prefix := findPrefix(r2, "")
	if "prefix.prefix2." != prefix {
		t.Fatal(prefix)
	}
}

func TestNewPrefixedRegistry(t *testing.T) {
	r0 := NewPrefixedRegistry("a.")
	NewPrefixedChildRegistry(r0, "b.c.")
	r0sr, name0 := findPrefix(r0, "b.")
	if r0sr == nil {
		t.Fatalf("findPrefix failed")
	}
	counter := NewRegisteredCounter("counter", r0sr)
	counter.Inc(50)
	r1sr, _ := findPrefix(r0, "b.c.")
	if r1sr == nil {
		t.Fatalf("findPrefix failed")
	}
	c, ok := r1sr.Get("counter").(Counter)
	if !ok {
		t.Fatal("r1 conversion failed")
	} else {
		t.Logf("r0sr.(type:%v, value:%v, name:%v)\n", reflect.TypeOf(r0sr), c.Count(), name0)
	}
}

// test NewPrefixedRegistry whether underlying Registry is only relate to prefix and the root PrefixedRegistry
func TestPrefixedRegistry_GetOrRegister(t *testing.T) {
	r0 := NewPrefixedRegistry("prefix1.")
	r1 := NewPrefixedChildRegistry(r0, "prefix2.")
	r2 := NewPrefixedChildRegistry(r0, "prefix2.prefix3.")

	r11 := NewPrefixedChildRegistry(r1, "prefix3.")
	r12 := NewPrefixedChildRegistry(r11, "prefix4.")
	r21 := NewPrefixedChildRegistry(r2, "prefix4.")
	if r12 != r21 {
		t.Logf("NewPrefixedChildRegistry not the same")
	}
	fdRsty1, _ := findPrefix(r11, "prefix4")
	fdRsty2, _ := findPrefix(r2, "prefix4")
	if fdRsty1 != fdRsty2 {
		t.Fatalf("findPrefix registry not the same")
	} else {
		c := r12.GetOrRegister("c", NewCounter())
		c.(Counter).Inc(2)
		if getC, ok := r21.Get("c").(Counter); !ok {
			t.Fatalf("get or convert failed")
		} else {
			if getC.Count() != 2 {
				t.Fatalf("not the same object")
			}
		}
	}

	r1sr, _ := findPrefix(r1, "prefix3.")
	r2sr, _ := findPrefix(r2, "")
	if r1sr != r2sr {
		t.Fatalf("findPrefix registry not the same")
	} else {
		gauge := NewRegisteredGauge("gauge", r2)
		gauge.Update(50)
		g1, ok1 := r1.Get("prefix3.gauge").(Gauge)
		if !ok1 {
			t.Fatalf("r1sr conversion failed")
		}
		g2, ok2 := r2.Get("gauge").(Gauge)
		if !ok2 {
			t.Fatalf("r2sr conversion failed")
		}
		if g1.Value() != g2.Value() {
			t.Fatalf("\ng1.Value:%v\ng2.Value:%v\n", g1.Value(), g2.Value())
		}
		ret := GetOrRegister("prefix1.prefix2.prefix3.", NewGauge())
		if retGauge, ok := ret.(Gauge); !ok {
			t.Fatal("type doesn't match Counter")
		} else {
			if retGauge.Value() != gauge.Value() {
				t.Fatalf("\ngauge.Value:%v\nretGauge.Value:%v\n", gauge.Value(), retGauge.Value())
			}
		}
	}
}

func TestConcurrentRegistryAccess(t *testing.T) {
	r := NewRegistry()

	counter := NewCounter()

	signalChan := make(chan struct{})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(dowork chan struct{}) {
			defer wg.Done()
			iface := r.GetOrRegister("foo", counter)
			retCounter, ok := iface.(Counter)
			if !ok {
				t.Fatal("Expected a Counter type")
			}
			if retCounter != counter {
				t.Fatal("Counter references don't match")
			}
		}(signalChan)
	}

	close(signalChan) // Closing will cause all go routines to execute at the same time
	wg.Wait()         // Wait for all go routines to do their work

	// At the end of the test we should still only have a single "foo" Counter
	i := 0
	r.Each(func(name string, iface interface{}) {
		i++
		if "foo" != name {
			t.Fatal(name)
		}
		if _, ok := iface.(Counter); !ok {
			t.Fatal(iface)
		}
	})
	if 1 != i {
		t.Fatal(i)
	}
	r.Unregister("foo")
	i = 0
	r.Each(func(string, interface{}) { i++ })
	if 0 != i {
		t.Fatal(i)
	}
}
