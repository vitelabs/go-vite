package database

import (
	"log"
	"math/big"
	"testing"
)

func TestNewDbComparer(t *testing.T) {
	comparer := NewDbComparer("test")
	a, _ := EncodeKey(1, big.NewInt(1))
	b, _ := EncodeKey(1, big.NewInt(2))
	log.Println(comparer.Compare(a, b))

	a1, _ := EncodeKey(1, big.NewInt(1), []byte{12})
	b1, _ := EncodeKey(1, big.NewInt(1), []byte{13})
	log.Println(comparer.Compare(a1, b1))

	a2, _ := EncodeKey(3, big.NewInt(1), []byte{12})
	b2, _ := EncodeKey(2, big.NewInt(1), []byte{12})
	log.Println(comparer.Compare(a2, b2))

	a3, _ := EncodeKey(3, big.NewInt(1))
	b3, _ := EncodeKey(3, big.NewInt(1), []byte{12})
	log.Println(comparer.Compare(a3, b3))

	a4, _ := EncodeKey(3, big.NewInt(0))
	b4, _ := EncodeKey(3, big.NewInt(0))
	log.Println(comparer.Compare(a4, b4))

	a5, _ := EncodeKey(3, big.NewInt(0), "KEY_MAX")
	b5, _ := EncodeKey(3, big.NewInt(0), []byte{121, 212, 2, 3, 4, 5})
	log.Println(comparer.Compare(a5, b5))
}

func BenchmarkNewDbComparer(bench *testing.B) {
	comparer := NewDbComparer("test")

	//for i := 0; i < 100000; i++ {
	a0, _ := EncodeKey(1, big.NewInt(1))
	b0, _ := EncodeKey(1, big.NewInt(2))
	comparer.Compare(a0, b0)

	a1, _ := EncodeKey(1, big.NewInt(1), []byte{12})
	b1, _ := EncodeKey(1, big.NewInt(1), []byte{13})
	comparer.Compare(a1, b1)

	a2, _ := EncodeKey(3, big.NewInt(1), []byte{12})
	b2, _ := EncodeKey(2, big.NewInt(1), []byte{12})
	comparer.Compare(a2, b2)

	a3, _ := EncodeKey(3, big.NewInt(1))
	b3, _ := EncodeKey(3, big.NewInt(1), []byte{12})
	comparer.Compare(a3, b3)

	a4, _ := EncodeKey(3, big.NewInt(0))
	b4, _ := EncodeKey(3, big.NewInt(0))
	comparer.Compare(a4, b4)

	a5, _ := EncodeKey(3, big.NewInt(0), "KEY_MAX")
	b5, _ := EncodeKey(3, big.NewInt(0), []byte{121, 212, 2, 3, 4, 5})
	comparer.Compare(a5, b5)
	//}
}
