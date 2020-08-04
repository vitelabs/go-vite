package chain_utils

import (
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common"

	"github.com/vitelabs/go-vite/common/types"
)

func TestCreateAccountIdKey(t *testing.T) {
	fmt.Println(CreateAccountIdKey(8).Bytes())
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

func TestRefill(t *testing.T) {
	key := AccountBlockHashKey{}
	hash := types.HexToHashPanic("ab750df7fa9736f445572ed9c9c979c61368907d4b9f2f81fbf1822b2d947b50")
	fmt.Printf("%v\n", key.Bytes())
	key.HashRefill(hash)
	fmt.Printf("%v\n", key.Bytes())
}

func TestCopy(t *testing.T) {
	addr := types.HexToAddressPanic("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	key := CreateStorageValueKey(&addr, []byte{1, 1}).Bytes()
	fmt.Printf("%v\n", key)
	copy(key[1+types.AddressSize:], common.RightPadBytes([]byte{1, 2, 3}, 32))
	fmt.Printf("%v\n", key)
}

func TestCopy2(t *testing.T) {
	fmt.Printf("%v\n", []byte("111"))
	fmt.Printf("%v\n", string([]byte{49}))
	fmt.Printf("%v\n", string(common.RightPadBytes([]byte{49}, 32)))
}

func TestCreateAccountBlockHashKey(t *testing.T) {
	hash := types.HexToHashPanic("ab750df7fa9736f445572ed9c9c979c61368907d4b9f2f81fbf1822b2d947b50")
	key := CreateAccountBlockHashKey(&hash)
	//key1 := CreateAccountBlockHashKey1(&hash)

	fmt.Printf("%v\n", key.Bytes())
	//fmt.Printf("%v\n", key1)
}

func TestRefillHeight(t *testing.T) {
	addr := types.HexToAddressPanic("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	key := CreateStorageValueKey(&addr, []byte{2, 4})
	key.KeyRefill(StorageRealKey{}.Construct([]byte{5, 1}))
	key2 := CreateStorageValueKey(&addr, []byte{5, 1})

	fmt.Printf("%v, %d\n", key.Bytes(), len(key.Bytes()))
	fmt.Printf("%v, %d\n", key2.Bytes(), len(key2.Bytes()))
	//fmt.Printf("%v\n", key1)
}

func TestCreateBalanceKey(t *testing.T) {
	addr := types.HexToAddressPanic("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	key := CreateBalanceKey(addr, ledger.ViteTokenId)
	fmt.Printf("%v, %d\n", key.Bytes(), len(key.Bytes()))
}

func TestCreateBalanceHistoryKey(t *testing.T) {
	addr := types.HexToAddressPanic("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	key := CreateHistoryBalanceKey(addr, ledger.ViteTokenId, 123)
	fmt.Printf("%v, %d\n", key.Bytes(), len(key.Bytes()))
}
