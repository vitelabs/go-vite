package onroad

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

type tMap map[string][]*big.Int

func (m tMap) Add(key string, value *big.Int) {
	if l, ok := m[key]; ok && l != nil {
		p := m[key]
		p = append(p, value)
		if &p != &l {
			fmt.Printf("copy, not pointer")
		}
	} else {
		new_l := make([]*big.Int, 0)
		new_l = append(new_l, value)
		m[key] = new_l
	}
}

func TestPendingCache_Map(t *testing.T) {
	tmap := make(tMap)
	strList := []string{"a", "b", "c", "d"}

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < len(strList)-1; i++ {
		l := make([]*big.Int, 0)
		for j := 0; j < 2; j++ {
			l = append(l, big.NewInt(int64(i)))
		}
		tmap[strList[rand.Intn(len(strList))]] = l
	}

	for k, l := range tmap {
		for _, v := range l {
			fmt.Printf("tMap[%v]=%v\t", k, v)
		}
	}

	addKey := strList[rand.Intn(len(strList))]
	fmt.Printf("befor %v\n", len(tmap[addKey]))
	tmap.Add(addKey, big.NewInt(9))
	fmt.Printf("after %v\n", len(tmap[addKey]))
	if _, ok := tmap[addKey]; ok {
		for _, v := range tmap[addKey] {
			fmt.Printf("%v \t", v)
		}
	}
}
