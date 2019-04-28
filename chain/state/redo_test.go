package chain_state

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"testing"
)

func TestGob(t *testing.T) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	for i := uint64(1); i <= 100; i++ {
		data := &LogItem{
			Height: 1,
		}

		if err := enc.Encode(data); err != nil {
			t.Fatal(err)
		}

	}

	fmt.Println()
	index := 1
	for {
		data1 := &LogItem{}
		if err := dec.Decode(data1); err != nil {
			t.Fatal(err)
		}
		fmt.Println(index, data1)
		index++
	}

}
