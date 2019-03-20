package chain_utils

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func TestCreateAccountIdKey(t *testing.T) {
	fmt.Println(CreateAccountIdKey(8))
}

func BenchmarkCreateAccountAddressKey(b *testing.B) {
	addr, _, err := types.CreateAddress()
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Method 1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			addrBytes := addr.Bytes()
			key := make([]byte, 0, 1+types.AddressSize+types.AddressSize+types.AddressSize)
			key = append(append(append(append(key, AccountAddressKeyPrefix), addrBytes...), addrBytes...), addrBytes...)

		}
	})

	b.Run("Method 2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			addrBytes := addr.Bytes()
			key := make([]byte, 0, 1+types.AddressSize+types.AddressSize+types.AddressSize)
			key = append(key, AccountAddressKeyPrefix)
			key = append(key, addrBytes...)
			key = append(key, addrBytes...)
			key = append(key, addrBytes...)
		}
	})
}
