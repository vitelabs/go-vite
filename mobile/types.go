package mobile

import (
	"github.com/vitelabs/go-vite/common/types"
)

type Address struct {
	address types.Address
}

func NewAddressFromByte(binary []byte) (address *Address, _ error) {
	addresses, e := types.BytesToAddress(binary)
	if e != nil {
		return nil, e
	}
	a := new(Address)
	a.address = addresses
	return a, nil
}

func NewAddressFromString(s string) (address *Address, _ error) {
	addresses, e := types.HexToAddress(s)
	if e != nil {
		return nil, e
	}
	a := new(Address)
	a.address = addresses
	return a, nil
}

func (a *Address) SetBytes(address []byte) error {
	return a.address.SetBytes(address)
}

func (a *Address) GetBytes() []byte {
	return a.address[:]
}

func (a *Address) SetHex(s string) error {
	addresses, e := types.HexToAddress(s)
	if e != nil {
		return e
	}
	a.address = addresses
	return nil
}

func (a *Address) GetHex() string {
	return a.address.Hex()
}


