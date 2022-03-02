package common

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/v2/log15"
)

func SyncMapLen(m *sync.Map) uint64 {
	if m == nil {
		return 0
	}
	i := uint64(0)
	m.Range(func(key, value interface{}) bool {
		i++
		return true
	})
	return i
}

func ToJson(item interface{}) string {
	byt, err := json.Marshal(item)
	if err != nil {
		return "err: " + err.Error()
	}
	return string(byt)
}

func Crit(msg string, ctx ...interface{}) {
	log := log15.New("moduble", "crit")
	log.Error(msg, ctx...)
	fmt.Printf("%s\n", msg)
	time.Sleep(time.Second * 2)
	log.Crit(msg, ctx...)
}
