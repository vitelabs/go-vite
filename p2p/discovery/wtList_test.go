package discovery

import (
	"testing"
	"time"
)

var list = newWtList()

func wtListRemoveTest(t *testing.T) {
	const n = 10
	for i := 0; i < n; i++ {
		list.append(&wait{
			expectCode: packetCode(i),
			expiration: time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	time.Sleep(2 * time.Second)

	i := 0
	list.traverse(func(prev, current *wait) {
		if current.expiration.Before(time.Now()) {
			i++
			list.remove(prev, current)
		}
	})

	if i != n {
		t.Fail()
	}
}