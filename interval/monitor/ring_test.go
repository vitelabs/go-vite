package monitor

import "testing"

func TestRing(t *testing.T) {
	r := newRing(10)

	r.add(0)
	r.add(1)
	r.add(2)
	all := r.all()
	if len(all) != 3 {
		t.Error("error len.")
	}
	for _, v := range all {
		println(v.(int))
	}

	r.reset()

	all = r.all()

	if len(all) != 0 {
		t.Error("error len.")
	}

	r.add(10)
	r.add(11)

	for _, v := range r.all() {
		println(v.(int))
	}

	println("------------------")
	r.reset()
	for i := 0; i < 12; i++ {
		r.add(i)
	}

	all = r.all()
	if len(all) != 10 {
		t.Error("error len.")
	}

	for _, v := range all {
		println(v.(int))
	}

}
