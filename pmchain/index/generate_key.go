package chain_index

import "github.com/vitelabs/go-vite/common/types"

func getAccountAddressKey(addr *types.Address) []byte {
	addrBytes := addr.Bytes()
	accountAddressKey := make([]byte, 0, 1+types.AddressSize)
	accountAddressKey = append(append(accountAddressKey, AccountAddressKeyPrefix), addrBytes...)
	return accountAddressKey
}
